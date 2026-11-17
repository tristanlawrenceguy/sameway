package mailin

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// The same app password that reads a mailbox sends from it: every mail
// service takes it on its submission port, beside where its mail is read.
// What is sent is plain text, from the person's own address, and goes to
// their Sent folder the way their service keeps what is sent.

// SMTPHost is where a mailbox's mail is sent from, by where it is read.
func SMTPHost(imapHost string) string {
	host, _, err := net.SplitHostPort(imapHost)
	if err != nil {
		host = imapHost
	}
	switch {
	case host == "imap.gmx.net":
		return "mail.gmx.net:587"
	case strings.HasPrefix(host, "imap."):
		return "smtp." + strings.TrimPrefix(host, "imap.") + ":587"
	}
	return host + ":587"
}

// dialSMTP reaches a mail server; a test gives one of its own.
var dialSMTP = func(ctx context.Context, addr string) (net.Conn, error) {
	return (&net.Dialer{Timeout: 20 * time.Second}).DialContext(ctx, "tcp", addr)
}

// Outgoing is one email to send, to one address.
type Outgoing struct {
	To, Subject, Text string
	// Calendar is an invitation it carries (export.Invite), or nil.
	Calendar []byte
	// ReplyTo is where an answer goes, when not the mailbox itself: the
	// address Sameway reads, so a conversation comes back to it.
	ReplyTo string
	// Refs are the emails it answers, oldest first: the thread it is in.
	Refs []string
}

// Send sends one email from the account, as plain text, and gives the
// Message-ID it went with, by which an answer names it.
func Send(ctx context.Context, a Account, m Outgoing) (string, error) {
	rcpt, err := mail.ParseAddress(m.To)
	if err != nil {
		return "", fmt.Errorf("%q is not an email address", m.To)
	}
	addr := SMTPHost(a.Host)
	conn, err := dialSMTP(ctx, addr)
	if err != nil {
		return "", fmt.Errorf("%s could not be reached", addr)
	}
	if dl, ok := ctx.Deadline(); ok {
		conn.SetDeadline(dl)
	}
	host, _, _ := net.SplitHostPort(addr)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return "", fmt.Errorf("%s did not answer as a mail server", addr)
	}
	defer c.Close()
	if !a.Insecure {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return "", fmt.Errorf("%s did not answer securely", addr)
		}
	}
	if err := c.Auth(smtp.PlainAuth("", a.User, a.Password, host)); err != nil {
		return "", errors.New("the mailbox did not take its app password for sending: connect it again on Email in")
	}
	if err := c.Mail(a.User); err != nil {
		return "", err
	}
	if err := c.Rcpt(rcpt.Address); err != nil {
		return "", fmt.Errorf("%s would not send to %s", addr, rcpt.Address)
	}
	w, err := c.Data()
	if err != nil {
		return "", err
	}
	id := messageID(a.User)
	m.To = rcpt.String()
	if _, err := w.Write(message(a.User, id, m)); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return id, c.Quit()
}

// messageID is a new email's own name, on the sender's domain.
func messageID(from string) string {
	b := make([]byte, 12)
	rand.Read(b)
	_, domain, _ := strings.Cut(from, "@")
	return "<" + hex.EncodeToString(b) + "@" + domain + ">"
}

// message is an email as it is sent: one line per header, a subject in
// any language, and the text quoted-printable, so no line is too long.
// With a calendar it is the text and the calendar as two ways of saying
// one thing, which is how a mail app knows an invitation and offers its
// own Yes, No and Maybe.
func message(from, id string, m Outgoing) []byte {
	subject := strings.Join(strings.Fields(m.Subject), " ")
	text, calendar := m.Text, m.Calendar
	headers := [][2]string{
		{"From", from}, {"To", m.To}, {"Subject", mime.QEncoding.Encode("utf-8", subject)},
		{"Date", time.Now().Format(time.RFC1123Z)}, {"Message-ID", id},
	}
	if r, err := mail.ParseAddress(m.ReplyTo); err == nil {
		headers = append(headers, [2]string{"Reply-To", r.String()})
	}
	var refs []string
	for _, r := range m.Refs {
		if r = strings.Trim(strings.TrimSpace(r), "<>"); r != "" && !strings.ContainsAny(r, "<>\r\n \t") {
			refs = append(refs, "<"+r+">")
		}
	}
	if len(refs) > 0 {
		headers = append(headers, [2]string{"In-Reply-To", refs[len(refs)-1]}, [2]string{"References", strings.Join(refs, " ")})
	}
	var b bytes.Buffer
	for _, h := range append(headers, [2]string{"MIME-Version", "1.0"}) {
		b.WriteString(h[0] + ": " + h[1] + "\r\n")
	}
	words := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
	if len(calendar) == 0 {
		b.WriteString("Content-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n")
		q := quotedprintable.NewWriter(&b)
		q.Write([]byte(words))
		q.Close()
		return b.Bytes()
	}
	mw := multipart.NewWriter(&b)
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + mw.Boundary() + "\"\r\n\r\n")
	part, _ := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=utf-8"}, "Content-Transfer-Encoding": {"quoted-printable"}})
	q := quotedprintable.NewWriter(part)
	q.Write([]byte(words))
	q.Close()
	method := "REQUEST"
	for _, m := range []string{"CANCEL", "REPLY"} {
		if bytes.Contains(calendar, []byte("METHOD:"+m)) {
			method = m
		}
	}
	part, _ = mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/calendar; charset=utf-8; method=" + method}, "Content-Transfer-Encoding": {"base64"}})
	enc := base64.StdEncoding.EncodeToString(calendar)
	for len(enc) > 76 {
		part.Write([]byte(enc[:76] + "\r\n"))
		enc = enc[76:]
	}
	part.Write([]byte(enc + "\r\n"))
	mw.Close()
	return b.Bytes()
}
