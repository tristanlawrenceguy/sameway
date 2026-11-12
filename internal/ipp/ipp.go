// Package ipp prints on a printer on the home network: the Internet
// Printing Protocol that nearly every network printer of the last fifteen
// years speaks, with no driver. A print job is one HTTP POST of a few
// attributes followed by the document; the printer answers with a status
// code. Printers announce themselves by mDNS (internal/discover), which
// says where to send and which kinds of document they take.
package ipp

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/discover"
)

// Printer is one printer found: its name as it announces itself, where a
// job goes, and the kinds of document it takes (empty when it did not say).
type Printer struct {
	Name    string
	URL     string // http://192.168.1.20:631/ipp/print
	Formats []string
}

// Takes says whether the printer takes a kind of document. One that did
// not say is let try: it answers whether it does.
func (p Printer) Takes(format string) bool {
	if len(p.Formats) == 0 {
		return true
	}
	for _, f := range p.Formats {
		if strings.EqualFold(f, format) {
			return true
		}
	}
	return false
}

// Find asks the home network for printers that take IPP, for as long as
// wait. Only _ipp._tcp: the older _printer._tcp is LPD, another protocol,
// and _ipps._tcp is the same with TLS, which a printer offers beside it.
func Find(ctx context.Context, wait time.Duration) ([]Printer, error) {
	found, err := discover.Browse(ctx, wait, "_ipp._tcp")
	if err != nil {
		return nil, err
	}
	var out []Printer
	for _, svc := range found {
		if p, ok := fromService(svc); ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// fromService is the printer an mDNS answer describes: its TXT records
// say the path jobs go to (rp=) and the kinds it takes (pdl=).
func fromService(svc discover.Service) (Printer, bool) {
	host := svc.Addr // an address works where a .local name may not
	if host == "" {
		host = svc.Host
	}
	if svc.Type != "_ipp._tcp" || host == "" || svc.Port == 0 {
		return Printer{}, false
	}
	rp := "ipp/print"
	var formats []string
	for _, t := range svc.Text {
		k, v, _ := strings.Cut(t, "=")
		switch strings.ToLower(k) {
		case "rp":
			rp = v
		case "pdl":
			for _, f := range strings.Split(v, ",") {
				if f = strings.TrimSpace(f); f != "" {
					formats = append(formats, f)
				}
			}
		}
	}
	return Printer{Name: svc.Name, URL: fmt.Sprintf("http://%s:%d/%s", host, svc.Port, strings.TrimPrefix(rp, "/")), Formats: formats}, true
}

// Print sends one document to a printer: format is its media type, such
// as application/pdf, and name is the job's name the printer may show.
// It is done when the printer has taken it; an error says why it did not,
// in words.
func Print(ctx context.Context, p Printer, doc io.Reader, format, name string) error {
	req := Message{Code: PrintJob, RequestID: 1, Attrs: []Attr{
		{Group: OperationGroup, Tag: charsetTag, Name: "attributes-charset", Value: "utf-8"},
		{Group: OperationGroup, Tag: languageTag, Name: "attributes-natural-language", Value: "en"},
		{Group: OperationGroup, Tag: uriTag, Name: "printer-uri", Value: "ipp" + strings.TrimPrefix(p.URL, "http")},
		{Group: OperationGroup, Tag: nameTag, Name: "requesting-user-name", Value: "Sameway"},
		{Group: OperationGroup, Tag: nameTag, Name: "job-name", Value: name},
		{Group: OperationGroup, Tag: mediaTypeTag, Name: "document-format", Value: format},
	}}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, io.MultiReader(bytes.NewReader(req.Encode()), doc))
	if err != nil {
		return err
	}
	hr.Header.Set("Content-Type", "application/ipp")
	res, err := http.DefaultClient.Do(hr)
	if err != nil {
		// What the network said (dial tcp ..., connectex: ...) is not for a
		// person: an off or sleeping printer is what it nearly always means.
		return fmt.Errorf("%s did not answer. It may be switched off or asleep: switch it on, then print again", p.Name)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s refused it (HTTP %s)", p.Name, res.Status)
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	ans, err := Decode(body)
	if err != nil {
		return fmt.Errorf("%s answered something that is not IPP: %w", p.Name, err)
	}
	if ans.Code < 0x0100 { // successful-ok and its kin
		return nil
	}
	said := statusWords[ans.Code]
	if said == "" {
		said = fmt.Sprintf("it answered status 0x%04x", ans.Code)
	}
	if m := ans.Value("status-message"); m != "" {
		said += " (" + m + ")"
	}
	return fmt.Errorf("%s did not print it: %s", p.Name, said)
}

// statusWords are the printer's refusals a person can act on.
var statusWords = map[uint16]string{
	0x0400: "it did not understand the request",
	0x0401: "it does not print for Sameway",
	0x0402: "it wants a sign-in first",
	0x0403: "it wants a sign-in first",
	0x0404: "it cannot print that now",
	0x0408: "the document is too big for it",
	0x040A: "it does not print this kind of file",
	0x040B: "it does not take one of the settings sent",
	0x0500: "it had a fault of its own",
	0x0501: "it does not take print jobs this way",
	0x0502: "it is not printing now",
	0x0506: "it is not taking jobs now",
	0x0507: "it is busy; try again in a moment",
}

// What a message carries: the operation asked for, the groups attributes
// sit in, and the kinds of value used here.
const (
	PrintJob uint16 = 0x0002

	OperationGroup byte = 0x01
	endTag         byte = 0x03

	nameTag      byte = 0x42
	uriTag       byte = 0x45
	charsetTag   byte = 0x47
	languageTag  byte = 0x48
	mediaTypeTag byte = 0x49
)

// Message is an IPP request or answer: Code is the operation asked for or
// the status answered, and Data is what follows the attributes, the
// document in a Print-Job.
type Message struct {
	Code      uint16
	RequestID uint32
	Attrs     []Attr
	Data      []byte
}

// Attr is one attribute, in the group it was sent in. A value that is a
// number is its bytes as sent.
type Attr struct {
	Group, Tag  byte
	Name, Value string
}

// Value is the first value of an attribute, "" when it is not there.
func (m Message) Value(name string) string {
	for _, a := range m.Attrs {
		if a.Name == name {
			return a.Value
		}
	}
	return ""
}

// Encode is the message as sent, IPP 1.1, the version every printer takes.
func (m Message) Encode() []byte {
	var b bytes.Buffer
	b.Write([]byte{1, 1})
	binary.Write(&b, binary.BigEndian, m.Code)
	binary.Write(&b, binary.BigEndian, m.RequestID)
	var group byte
	for _, a := range m.Attrs {
		if a.Group != group {
			group = a.Group
			b.WriteByte(group)
		}
		b.WriteByte(a.Tag)
		binary.Write(&b, binary.BigEndian, uint16(len(a.Name)))
		b.WriteString(a.Name)
		binary.Write(&b, binary.BigEndian, uint16(len(a.Value)))
		b.WriteString(a.Value)
	}
	b.WriteByte(endTag)
	b.Write(m.Data)
	return b.Bytes()
}

// Decode reads a message. A value after the first of one attribute comes
// with no name of its own and is kept under the name before it.
func Decode(data []byte) (Message, error) {
	short := errors.New("the message ends early")
	if len(data) < 9 {
		return Message{}, short
	}
	m := Message{Code: binary.BigEndian.Uint16(data[2:4]), RequestID: binary.BigEndian.Uint32(data[4:8])}
	var group byte
	name := ""
	for i := 8; i < len(data); {
		tag := data[i]
		i++
		if tag == endTag {
			m.Data = data[i:]
			return m, nil
		}
		if tag < 0x10 { // a group begins
			group = tag
			continue
		}
		if i+2 > len(data) {
			return m, short
		}
		n := int(binary.BigEndian.Uint16(data[i:]))
		if i += 2; i+n+2 > len(data) {
			return m, short
		}
		if n > 0 {
			name = string(data[i : i+n])
		}
		i += n
		v := int(binary.BigEndian.Uint16(data[i:]))
		if i += 2; i+v > len(data) {
			return m, short
		}
		m.Attrs = append(m.Attrs, Attr{Group: group, Tag: tag, Name: name, Value: string(data[i : i+v])})
		i += v
	}
	return m, short
}
