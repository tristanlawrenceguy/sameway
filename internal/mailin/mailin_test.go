package mailin

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/mailin/mailintest"
)

func email(to, subject, body string) string {
	return "From: Ana Silva <ana@example.com>\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nMessage-Id: <" + strings.ReplaceAll(subject, " ", ".") + "@example.com>\r\nDate: Mon, 5 Oct 2026 09:00:00 +0000\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body + "\r\n"
}

// Only mail to the +sameway address is read from the inbox, all of the
// Sameway folder's from when it was first seen, and nothing twice; a
// wrong password is said plainly.
func TestOnlyMailForTheWorkspaceIsRead(t *testing.T) {
	t.Parallel()
	srv := mailintest.Start(t, "me@example.com", "app-pass")
	srv.Folder(Folder)
	srv.Put(t, "INBOX", email("me+sameway@example.com", "Dentist on Friday", "At 10."))
	srv.Put(t, "INBOX", email("me@example.com", "Private", "Not for the workspace."))
	srv.Put(t, Folder, email("me@example.com", "Old in folder", "Before connecting."))
	a := Account{Host: srv.Addr, User: "me@example.com", Password: "app-pass", Insecure: true}
	ctx := context.Background()

	if err := Check(ctx, Account{Host: srv.Addr, User: "me@example.com", Password: "wrong", Insecure: true}); err == nil || !strings.Contains(err.Error(), "app password") {
		t.Errorf("a wrong password is said plainly: %v", err)
	}
	plain := mailintest.Start(t, "you@example.com", "pw")
	if err := Check(ctx, Account{Host: plain.Addr, User: "you@example.com", Password: "pw", Insecure: true}); err != nil {
		t.Fatal(err)
	}
	if !plain.Has(Folder) {
		t.Error("connecting makes the Sameway folder, so there is somewhere to move mail")
	}
	mails, marks, err := New(ctx, a, nil, nil)
	if err != nil || len(mails) != 1 || mails[0].Subject != "Dentist on Friday" || mails[0].Text != "At 10." || !strings.Contains(mails[0].From, "Ana Silva") {
		t.Fatalf("the +sameway mail, alone: %+v %v", mails, err)
	}
	srv.Put(t, Folder, email("me@example.com", "Moved here", "Keep this."))
	srv.Put(t, "INBOX", email("Me <me+sameway@example.com>", "Bill", "Pay by the 20th."))
	mails, marks, err = New(ctx, a, marks, nil)
	var got []string
	for _, m := range mails {
		got = append(got, m.Subject)
	}
	if err != nil || strings.Join(got, ",") != "Bill,Moved here" {
		t.Fatalf("what came since, in both: %v %v", got, err)
	}
	if mails, _, _ = New(ctx, a, marks, nil); len(mails) != 0 {
		t.Errorf("nothing twice: %+v", mails)
	}
}

// An email's words are its plain text, or its HTML's; files come along.
func TestAnEmailIsReadIntoWordsAndFiles(t *testing.T) {
	t.Parallel()
	raw := "From: shop@example.com\r\nSubject: Your order\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=b\r\n\r\n" +
		"--b\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<p>Order <b>42</b> ships Monday.</p>\r\n" +
		"--b\r\nContent-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"invoice.pdf\"\r\nContent-Transfer-Encoding: base64\r\n\r\nJVBERi0xLjQ=\r\n--b--\r\n"
	m, err := Parse(strings.NewReader(raw))
	if err != nil || m.Subject != "Your order" || !strings.Contains(m.HTML(), "<b>42</b>") || len(m.Files) != 1 || m.Files[0].Name != "invoice.pdf" || string(m.Files[0].Data) != "%PDF-1.4" {
		t.Fatalf("%+v %v", m, err)
	}
	if PlusAddress("me@gmail.com") != "me+sameway@gmail.com" || Host("Me@Gmail.com") != "imap.gmail.com:993" {
		t.Error("the address and the server are known from the email address")
	}
}

// A picture from the sender's server is not fetched when the email is
// read (a spy pixel says when and where it was opened): it is said by what
// it shows, or left out when it shows nothing; links stay.
func TestNoPictureIsFetchedFromTheSender(t *testing.T) {
	t.Parallel()
	md := "Your order ships Monday.\n\n![](https://track.example/open.gif?u=7)\n\n" +
		"[![Track parcel](https://cdn.example/btn.png)](https://shop.example/track)\n\n" +
		"[![](https://cdn.example/logo.png)](https://shop.example)\n\n![Logo](cid:logo@x) ![](data:image/png;base64,AAAA)"
	got := NoRemotePictures(md)
	for _, gone := range []string{"track.example", "cdn.example", "cid:", "[](https://shop.example)"} {
		if strings.Contains(got, gone) {
			t.Errorf("no %s left: %s", gone, got)
		}
	}
	for _, kept := range []string{"ships Monday", "[(picture: Track parcel)](https://shop.example/track)", "(picture: Logo)", "data:image/png"} {
		if !strings.Contains(got, kept) {
			t.Errorf("%s kept: %s", kept, got)
		}
	}
}
