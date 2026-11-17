package mailin

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
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

// Send sends one email from the account to one address, as plain text.
func Send(ctx context.Context, a Account, to, subject, text string) error {
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
	if _, err := w.Write(message(a.User, rcpt.String(), subject, text)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// message is an email as it is sent: one line per header, a subject in
// any language, and the text quoted-printable, so no line is too long.
func message(from, to, subject, text string) []byte {
	id := make([]byte, 12)
	rand.Read(id)
	_, domain, _ := strings.Cut(from, "@")
	subject = strings.Join(strings.Fields(subject), " ")
	var b bytes.Buffer
	for _, h := range [][2]string{
		{"From", from}, {"To", to}, {"Subject", mime.QEncoding.Encode("utf-8", subject)},
		{"Date", time.Now().Format(time.RFC1123Z)}, {"Message-ID", "<" + hex.EncodeToString(id) + "@" + domain + ">"},
		{"MIME-Version", "1.0"}, {"Content-Type", "text/plain; charset=utf-8"}, {"Content-Transfer-Encoding", "quoted-printable"},
	} {
		b.WriteString(h[0] + ": " + h[1] + "\r\n")
	}
	b.WriteString("\r\n")
	q := quotedprintable.NewWriter(&b)
	q.Write([]byte(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")))
	q.Close()
	return b.Bytes()
}
