package mailin

import (
	"io"
	"slices"
	"strings"

	_ "github.com/emersion/go-message/charset" // mail in any charset
	"github.com/emersion/go-message/mail"
)

// Parse reads an email: who from, the subject, the words (plain text, or
// the HTML's text when there is no plain), and the files attached.
func Parse(r io.Reader) (Mail, error) {
	mr, err := mail.CreateReader(r)
	if err != nil && mr == nil {
		return Mail{}, err
	}
	h := mr.Header
	var m Mail
	m.Subject, _ = h.Subject()
	m.Date, _ = h.Date()
	m.ID, _ = h.MessageID()
	m.Refs, _ = h.MsgIDList("References")
	if irt, _ := h.MsgIDList("In-Reply-To"); len(irt) > 0 && !slices.Contains(m.Refs, irt[0]) {
		m.Refs = append(m.Refs, irt[0])
	}
	if from, err := h.AddressList("From"); err == nil && len(from) > 0 {
		m.From = from[0].Address
		if from[0].Name != "" {
			m.From = from[0].Name + " <" + from[0].Address + ">"
		}
	}
	var plain, html string
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch ph := p.Header.(type) {
		case *mail.InlineHeader:
			kind, _, _ := ph.ContentType()
			b, _ := io.ReadAll(io.LimitReader(p.Body, 2<<20))
			switch {
			case kind == "text/plain" && plain == "":
				plain = string(b)
			case kind == "text/html" && html == "":
				html = string(b)
			}
		case *mail.AttachmentHeader:
			name, _ := ph.Filename()
			if name == "" {
				name = "attachment"
			}
			b, _ := io.ReadAll(io.LimitReader(p.Body, filesMost+1))
			if len(b) > filesMost {
				b = nil
			}
			m.Files = append(m.Files, File{Name: name, Data: b})
		}
	}
	m.Text = strings.TrimSpace(plain)
	if m.Text == "" && html != "" {
		m.Text = "\x00html\x00" + html // the caller turns HTML into words
	}
	return m, nil
}

// HTML is the HTML of a mail with no plain words, or "".
func (m Mail) HTML() string {
	if h, ok := strings.CutPrefix(m.Text, "\x00html\x00"); ok {
		return h
	}
	return ""
}
