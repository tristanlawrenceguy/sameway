// Package mailin reads the email a person sends to their workspace: mail
// to their own address with +sameway after the name (you+sameway@gmail.com),
// and anything they move to a folder called Sameway. It signs in over IMAP
// with an app password, the kind every mail service makes for a program,
// and only looks: nothing is marked read, moved or deleted.
package mailin

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	_ "github.com/emersion/go-message/charset" // mail in any charset
	"github.com/emersion/go-message/mail"
)

// Folder is the folder whose mail is all for the workspace.
const Folder = "Sameway"

// Tag is what goes after the + in the address mail is sent to.
const Tag = "sameway"

// Account is a mailbox to read: where, who, and the app password.
type Account struct {
	Host     string `json:"host"` // host:port, IMAP over TLS
	User     string `json:"user"`
	Password string `json:"-"`
	// Insecure skips TLS, for a test's server on this computer.
	Insecure bool `json:"-"`
}

// Mark is how far each folder has been read: the folder's UIDVALIDITY and
// the last UID seen, so nothing is read twice.
type Mark struct {
	Validity uint32 `json:"validity"`
	Last     uint32 `json:"last"`
}

// Mail is one email, as words and files.
type Mail struct {
	ID      string // the Message-Id, or folder and UID
	From    string
	Subject string
	Date    time.Time
	Text    string
	Files   []File
}

// File is a file attached to an email.
type File struct {
	Name string
	Data []byte
}

// filesMost is the largest attachment kept; a bigger one is named only.
const filesMost = 25 << 20

// PlusAddress is the address to send to: the person's own, with +sameway.
func PlusAddress(user string) string {
	name, domain, ok := strings.Cut(user, "@")
	if !ok {
		return ""
	}
	return name + "+" + Tag + "@" + domain
}

// Host is the IMAP server of a well-known mail service, by the address.
func Host(user string) string {
	_, domain, _ := strings.Cut(strings.ToLower(strings.TrimSpace(user)), "@")
	switch domain {
	case "gmail.com", "googlemail.com":
		return "imap.gmail.com:993"
	case "icloud.com", "me.com", "mac.com":
		return "imap.mail.me.com:993"
	case "yahoo.com", "yahoo.co.uk", "ymail.com":
		return "imap.mail.yahoo.com:993"
	case "fastmail.com", "fastmail.fm":
		return "imap.fastmail.com:993"
	case "aol.com":
		return "imap.aol.com:993"
	case "gmx.com", "gmx.net", "gmx.de":
		return "imap.gmx.net:993"
	case "zoho.com":
		return "imap.zoho.com:993"
	case "":
		return ""
	}
	return "imap." + domain + ":993"
}

func dial(ctx context.Context, a Account) (*imapclient.Client, error) {
	d := &net.Dialer{Timeout: 20 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", a.Host)
	if err != nil {
		return nil, fmt.Errorf("%s could not be reached", a.Host)
	}
	if !a.Insecure {
		host, _, _ := net.SplitHostPort(a.Host)
		tc := tls.Client(conn, &tls.Config{ServerName: host})
		if err := tc.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, fmt.Errorf("%s did not answer securely", a.Host)
		}
		conn = tc
	}
	if dl, ok := ctx.Deadline(); ok {
		conn.SetDeadline(dl)
	}
	c := imapclient.New(conn, nil)
	if err := c.Login(a.User, a.Password).Wait(); err != nil {
		c.Close()
		return nil, errors.New("the address and app password were not accepted: check both, and that the password is an app password, not the one you sign in with")
	}
	return c, nil
}

// Check signs in and out, to say at once whether the account works, and
// makes the Sameway folder when there is none, so a service with no +
// addresses has somewhere to put mail without the person making it.
func Check(ctx context.Context, a Account) error {
	c, err := dial(ctx, a)
	if err != nil {
		return err
	}
	c.Create(Folder, nil).Wait() // there already, it says so; either way it is there
	c.Logout().Wait()
	c.Close()
	return nil
}

// New reads the mail that came since marks: in the inbox, only what was
// sent to the +sameway address; in the Sameway folder, everything. A
// folder read for the first time starts from now, apart from the last
// week's +sameway mail, which was sent to be kept.
func New(ctx context.Context, a Account, marks map[string]Mark) ([]Mail, map[string]Mark, error) {
	c, err := dial(ctx, a)
	if err != nil {
		return nil, marks, err
	}
	defer c.Close()
	defer c.Logout()
	out := map[string]Mark{}
	for k, v := range marks {
		out[k] = v
	}
	var mails []Mail
	for _, box := range []string{"INBOX", Folder} {
		sel, err := c.Select(box, &imap.SelectOptions{ReadOnly: true}).Wait()
		if err != nil {
			continue // no Sameway folder yet
		}
		mark, seen := out[box]
		criteria := &imap.SearchCriteria{}
		switch {
		case !seen || mark.Validity != sel.UIDValidity:
			if box != "INBOX" {
				out[box] = Mark{Validity: sel.UIDValidity, Last: uint32(sel.UIDNext) - 1}
				continue
			}
			criteria.Since = time.Now().AddDate(0, 0, -7)
		default:
			criteria.UID = []imap.UIDSet{{imap.UIDRange{Start: imap.UID(mark.Last + 1), Stop: 0}}}
		}
		if box == "INBOX" {
			plus := PlusAddress(a.User)
			criteria.Or = [][2]imap.SearchCriteria{{
				{Header: []imap.SearchCriteriaHeaderField{{Key: "To", Value: plus}}},
				{Header: []imap.SearchCriteriaHeaderField{{Key: "Cc", Value: plus}}},
			}}
		}
		found, err := c.UIDSearch(criteria, nil).Wait()
		if err != nil {
			return mails, out, fmt.Errorf("%s could not be searched", box)
		}
		last := mark.Last
		if !seen || mark.Validity != sel.UIDValidity {
			last = 0
		}
		var uids []imap.UID
		for _, u := range found.AllUIDs() {
			if uint32(u) > last {
				uids = append(uids, u)
			}
		}
		next := Mark{Validity: sel.UIDValidity, Last: last}
		if len(uids) > 0 {
			got, err := fetch(c, box, uids)
			if err != nil {
				return mails, out, err
			}
			mails = append(mails, got...)
			for _, u := range uids {
				if uint32(u) > next.Last {
					next.Last = uint32(u)
				}
			}
		}
		out[box] = next
	}
	return mails, out, nil
}

func fetch(c *imapclient.Client, box string, uids []imap.UID) ([]Mail, error) {
	section := &imap.FetchItemBodySection{Peek: true}
	msgs, err := c.Fetch(imap.UIDSetNum(uids...), &imap.FetchOptions{UID: true, BodySection: []*imap.FetchItemBodySection{section}}).Collect()
	if err != nil {
		return nil, fmt.Errorf("the mail in %s could not be read", box)
	}
	var out []Mail
	for _, m := range msgs {
		raw := m.FindBodySection(section)
		if raw == nil {
			continue
		}
		mm, err := Parse(strings.NewReader(string(raw)))
		if err != nil {
			continue
		}
		if mm.ID == "" {
			mm.ID = fmt.Sprintf("%s/%d", box, m.UID)
		}
		out = append(out, mm)
	}
	return out, nil
}

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
