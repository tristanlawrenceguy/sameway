package ingest

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"
)

// mboxColumns are the columns a mailbox is read into: one row per message,
// shaped so it lands as an interaction with the person it was with.
var mboxColumns = []string{"kind", "at", "summary", "from_name", "from_email", "to", "notes"}

// ReadMbox reads a mailbox export (mbox, or one .eml) into one row per
// message: when it was sent, its subject, who it was from and to, and the
// first of its text. Each row is an email interaction; the sender is the
// person it is linked to.
func ReadMbox(data []byte) (*Table, error) {
	t := &Table{Source: "mbox", Columns: mboxColumns}
	for _, raw := range splitMbox(data) {
		msg, err := mail.ReadMessage(bytes.NewReader(raw))
		if err != nil {
			continue
		}
		row := map[string]string{"kind": "email"}
		if d, err := msg.Header.Date(); err == nil {
			row["at"] = d.UTC().Format(time.RFC3339)
		}
		row["summary"] = decodeHeader(msg.Header.Get("Subject"))
		if from, err := mail.ParseAddress(decodeHeader(msg.Header.Get("From"))); err == nil {
			row["from_name"], row["from_email"] = from.Name, strings.ToLower(from.Address)
		} else {
			row["from_email"] = strings.ToLower(strings.TrimSpace(msg.Header.Get("From")))
		}
		row["to"] = decodeHeader(msg.Header.Get("To"))
		row["notes"] = firstText(msg)
		if row["summary"] == "" {
			row["summary"] = "(no subject)"
		}
		t.Rows = append(t.Rows, row)
	}
	if len(t.Rows) == 0 {
		return nil, errors.New("no messages were found in the file")
	}
	return t, nil
}

// splitMbox cuts a mailbox at its From_ lines; a lone message is one part.
func splitMbox(data []byte) [][]byte {
	var parts [][]byte
	var cur bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if bytes.HasPrefix(line, []byte("From ")) && cur.Len() > 0 {
			parts = append(parts, append([]byte(nil), cur.Bytes()...))
			cur.Reset()
			continue
		}
		if bytes.HasPrefix(line, []byte("From ")) {
			continue
		}
		cur.Write(line)
		cur.WriteByte('\n')
	}
	if strings.TrimSpace(cur.String()) != "" {
		parts = append(parts, cur.Bytes())
	}
	return parts
}

func decodeHeader(s string) string {
	dec := new(mime.WordDecoder)
	if out, err := dec.DecodeHeader(s); err == nil {
		return strings.TrimSpace(out)
	}
	return strings.TrimSpace(s)
}

// firstText is the start of the plain text of a message, a few hundred
// characters at most, with quoted replies left out.
func firstText(msg *mail.Message) string {
	body := readPart(msg.Header.Get("Content-Type"), msg.Header.Get("Content-Transfer-Encoding"), msg.Body)
	var lines []string
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimRight(l, "\r")
		if strings.HasPrefix(l, ">") || strings.HasPrefix(l, "On ") && strings.HasSuffix(l, "wrote:") {
			break
		}
		lines = append(lines, l)
	}
	out := strings.TrimSpace(strings.Join(lines, "\n"))
	if len(out) > 600 {
		out = out[:600] + "…"
	}
	return out
}

func readPart(contentType, encoding string, r io.Reader) string {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(r, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err != nil {
				return ""
			}
			if text := readPart(p.Header.Get("Content-Type"), p.Header.Get("Content-Transfer-Encoding"), p); text != "" {
				return text
			}
		}
	}
	if mediaType != "" && !strings.HasPrefix(mediaType, "text/plain") {
		return ""
	}
	if strings.EqualFold(encoding, "quoted-printable") {
		r = quotedprintable.NewReader(r)
	}
	raw, _ := io.ReadAll(io.LimitReader(r, 64<<10))
	return string(raw)
}
