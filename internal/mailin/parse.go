package mailin

import (
	"io"
	"regexp"
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
	auto := strings.ToLower(strings.TrimSpace(h.Get("Auto-Submitted")))
	prec := strings.ToLower(strings.TrimSpace(h.Get("Precedence")))
	m.Auto = auto != "" && auto != "no" || prec == "bulk" || prec == "auto_reply" || prec == "list" || h.Get("X-Autoreply") != ""
	if from, err := h.AddressList("From"); err == nil && len(from) > 0 {
		m.Verified = verified(h.Get("Authentication-Results"), from[0].Address)
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
			case kind == "text/calendar" && m.Calendar == nil:
				m.Calendar = b
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

// A picture in an email is fetched from the sender's server when the
// email is opened, and that fetch is how a sender learns it was read, when
// and from where: a spy pixel, in most mail sent by a business. HEY strips
// them, Apple Mail fetches through a proxy. Sameway keeps the words: a
// picture from elsewhere is said by what it shows, and one that says
// nothing (the pixel) is left out. Links stay; following one is a choice.
var (
	remotePicture = regexp.MustCompile(`!\[([^\]]*)\]\(\s*<?(?:https?:|cid:|//)[^)]*\)`)
	emptyLink     = regexp.MustCompile(`(^|[^!])\[\s*\]\([^)]*\)`)
	blankLines    = regexp.MustCompile(`\n{3,}`)
)

// NoRemotePictures is an email's Markdown with each picture from another
// server turned into what it shows, or taken out when it says nothing.
func NoRemotePictures(markdown string) string {
	out := remotePicture.ReplaceAllStringFunc(markdown, func(img string) string {
		alt := strings.TrimSpace(remotePicture.FindStringSubmatch(img)[1])
		if alt == "" {
			return ""
		}
		return "(picture: " + alt + ")"
	})
	return blankLines.ReplaceAllString(emptyLink.ReplaceAllString(out, "$1"), "\n\n")
}

// verified reads what the receiving server found of who sent a mail, from
// the topmost Authentication-Results, the one it added itself (RFC 8601):
// DMARC passed for the sender's domain, or a DKIM signature of the
// sender's own domain did. The server copies in words the sender chose,
// such as the envelope address, in comments and quoted strings; those
// are dropped first, so "dmarc=pass" there vouches for nothing.
func verified(results, from string) bool {
	_, domain, _ := strings.Cut(strings.ToLower(from), "@")
	if domain == "" {
		return false
	}
	for _, part := range strings.Split(resultsWords(strings.ToLower(results)), ";") {
		f := strings.Fields(part)
		if len(f) == 0 {
			continue
		}
		for _, kv := range f[1:] {
			if d, ok := strings.CutPrefix(kv, "header.from="); ok && f[0] == "dmarc=pass" && d == domain {
				return true
			}
			if d, ok := strings.CutPrefix(kv, "header.d="); ok && f[0] == "dkim=pass" && (d == domain || strings.HasSuffix(domain, "."+d)) {
				return true
			}
		}
	}
	return false
}

// resultsWords is an Authentication-Results without its comments, (like
// this), and quoted strings, which may hold a semicolon of their own.
func resultsWords(s string) string {
	var b strings.Builder
	depth, quoted := 0, false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '\\' && (quoted || depth > 0):
			i++
		case quoted:
			quoted = c != '"'
		case c == '"' && depth == 0:
			quoted = true
			b.WriteByte(' ')
		case c == '(':
			depth++
		case c == ')' && depth > 0:
			depth--
			b.WriteByte(' ')
		case depth == 0:
			b.WriteByte(c)
		}
	}
	return b.String()
}
