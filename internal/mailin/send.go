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

// Send sends one email from the account to one address, as plain text,
// with a calendar when it carries an invitation (export.Invite).
func Send(ctx context.Context, a Account, to, subject, text string, calendar []byte) error {
	rcpt, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("%q is not an email address", to)
	}
	addr := SMTPHost(a.Host)
	conn, err := dialSMTP(ctx, addr)
	if err != nil {
		return fmt.Errorf("%s could not be reached", addr)
	}
	if dl, ok := ctx.Deadline(); ok {
		conn.SetDeadline(dl)
	}
	host, _, _ := net.SplitHostPort(addr)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("%s did not answer as a mail server", addr)
	}
	defer c.Close()
	if !a.Insecure {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return fmt.Errorf("%s did not answer securely", addr)
		}
	}
	if err := c.Auth(smtp.PlainAuth("", a.User, a.Password, host)); err != nil {
		return errors.New("the mailbox did not take its app password for sending: connect it again on Email in")
	}
	if err := c.Mail(a.User); err != nil {
		return err
	}
	if err := c.Rcpt(rcpt.Address); err != nil {
		return fmt.Errorf("%s would not send to %s", addr, rcpt.Address)
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(message(a.User, rcpt.String(), subject, text, calendar)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// message is an email as it is sent: one line per header, a subject in
// any language, and the text quoted-printable, so no line is too long.
// With a calendar it is the text and the calendar as two ways of saying
// one thing, which is how a mail app knows an invitation and offers its
// own Yes, No and Maybe.
func message(from, to, subject, text string, calendar []byte) []byte {
	id := make([]byte, 12)
	rand.Read(id)
	_, domain, _ := strings.Cut(from, "@")
	subject = strings.Join(strings.Fields(subject), " ")
	var b bytes.Buffer
	for _, h := range [][2]string{
		{"From", from}, {"To", to}, {"Subject", mime.QEncoding.Encode("utf-8", subject)},
		{"Date", time.Now().Format(time.RFC1123Z)}, {"Message-ID", "<" + hex.EncodeToString(id) + "@" + domain + ">"},
		{"MIME-Version", "1.0"},
	} {
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
	if bytes.Contains(calendar, []byte("METHOD:CANCEL")) {
		method = "CANCEL"
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
