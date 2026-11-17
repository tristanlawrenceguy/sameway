package mailin

import (
	"context"
	"io"
	"mime/quotedprintable"
	"net"
	"net/textproto"
	"strings"
	"testing"
)

// fakeSMTP answers one session on the other end of a pipe, nothing
// listening on the network, and gives what it was sent.
func fakeSMTP(t *testing.T) (got chan string) {
	got = make(chan string, 1)
	old := dialSMTP
	t.Cleanup(func() { dialSMTP = old })
	dialSMTP = func(context.Context, string) (net.Conn, error) {
		here, there := net.Pipe()
		go func() {
			defer there.Close()
			c := textproto.NewConn(there)
			c.PrintfLine("220 localhost ready")
			var b strings.Builder
			for {
				line, err := c.ReadLine()
				if err != nil {
					return
				}
				b.WriteString(line + "\n")
				switch verb := strings.ToUpper(strings.Fields(line + " x")[0]); verb {
				case "EHLO":
					c.PrintfLine("250-localhost")
					c.PrintfLine("250 AUTH PLAIN")
				case "AUTH":
					c.PrintfLine("235 ok")
				case "DATA":
					c.PrintfLine("354 go on")
					data, _ := c.ReadDotBytes()
					b.Write(data)
					c.PrintfLine("250 sent")
				case "QUIT":
					c.PrintfLine("221 bye")
					got <- b.String()
					return
				default:
					c.PrintfLine("250 ok")
				}
			}
		}()
		return here, nil
	}
	return got
}

// An email goes from the person's own mailbox, signed in with the app
// password that reads it, to one address, with a subject in any language
// and nothing a line break could add to its headers.
func TestSendGoesFromTheMailboxToOneAddress(t *testing.T) {
	got := fakeSMTP(t)
	a := Account{Host: "localhost:993", User: "tris@example.com", Password: "app", Insecure: true}
	err := Send(context.Background(), a, "Pat Lee <pat@example.com>", "Your visit\r\nBcc: everyone@example.com, Thursday é", "Yes, Thursday at 3pm.\nParking is round the back.")
	if err != nil {
		t.Fatal(err)
	}
	s := <-got
	for _, want := range []string{"MAIL FROM:<tris@example.com>", "RCPT TO:<pat@example.com>", "From: tris@example.com", `To: "Pat Lee" <pat@example.com>`, "Subject: =?utf-8?q?Your_visit_Bcc:_everyone@example.com,_Thursday_=C3=A9?="} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in\n%s", want, s)
		}
	}
	if strings.Contains(s, "\nBcc:") {
		t.Errorf("a line break in the subject adds no header:\n%s", s)
	}
	body, _ := io.ReadAll(quotedprintable.NewReader(strings.NewReader(s[strings.Index(s, "\r\n\r\n")+4:])))
	if !strings.Contains(string(body), "Parking is round the back.") {
		t.Errorf("the text is sent: %s", body)
	}
	if err := Send(context.Background(), a, "not an address", "x", "y"); err == nil {
		t.Error("only an address is sent to")
	}
	if SMTPHost("imap.gmail.com:993") != "smtp.gmail.com:587" || SMTPHost("imap.mail.me.com:993") != "smtp.mail.me.com:587" || SMTPHost("imap.gmx.net:993") != "mail.gmx.net:587" {
		t.Error("each service sends from beside where it is read")
	}
}
