package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/mailin/mailintest"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Connecting a mailbox brings the last week's +sameway mail in as notes
// to sort, files and all; Today lists them, each a press from a task or
// done with; the password stays out of the workspace.
func TestEmailComesInAsNotesToSort(t *testing.T) {
	server.MailInsecure(true)
	defer server.MailInsecure(false)
	mail := mailintest.Start(t, "me@example.com", "abcdefghijklmnop")
	mail.Put(t, "INBOX", "From: Clinic <desk@clinic.example>\r\nTo: me+sameway@example.com\r\nSubject: Dentist on Friday\r\nMIME-Version: 1.0\r\n"+
		"Content-Type: multipart/mixed; boundary=b\r\n\r\n--b\r\nContent-Type: text/plain\r\n\r\nFriday at 10, bring the form.\r\n"+
		"--b\r\nContent-Type: text/plain\r\nContent-Disposition: attachment; filename=\"form.txt\"\r\n\r\nName: ...\r\n--b--\r\n")
	a, h := newApp(t)
	if page := get(t, h, "/help").Body.String(); !strings.Contains(page, `href="/mail"`) {
		t.Errorf("Help offers it: %s", truncate(page))
	}
	if rec := postForm(t, h, "/mail/connect", url.Values{"user": {"me@example.com"}, "password": {"wrong"}, "host": {mail.Addr}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("%d", rec.Code)
	}
	if notes := emailNotes(t, a.Store); len(notes) != 0 {
		t.Fatal("a wrong password connects nothing")
	}
	// Gmail shows an app password in fours, spaces between.
	postForm(t, h, "/mail/connect", url.Values{"user": {"me@example.com"}, "password": {"abcd efgh ijkl mnop"}, "host": {mail.Addr}})
	notes := emailNotes(t, a.Store)
	if len(notes) != 1 {
		t.Fatalf("the mail came in: %d", len(notes))
	}
	n := notes[0]
	body, _ := n.Fields["body"].(string)
	if n.Fields["subject"] != "Dentist on Friday" || n.Fields["from"] != "Clinic <desk@clinic.example>" || n.Fields["received"] == nil || !strings.Contains(body, "bring the form") || !strings.Contains(body, "[form](/t/file/") {
		t.Errorf("subject, sender, words and file: %v", n.Fields)
	}
	if strings.Contains(a.Store.Meta("mail:account"), "abcdefghijklmnop") {
		t.Error("the password is not in the workspace")
	}
	if card := get(t, h, "/mail/contact.vcf").Body.String(); !strings.Contains(card, "EMAIL;TYPE=INTERNET:me+sameway@example.com") {
		t.Errorf("the address is a contact to add, nothing to remember: %s", card)
	}
	if page := get(t, h, "/mail").Body.String(); !strings.Contains(page, "me+sameway@example.com") || !strings.Contains(page, "contact.vcf") {
		t.Errorf("the page says where to send: %s", truncate(page))
	}
	today := get(t, h, "/today").Body.String()
	if !strings.Contains(today, "To sort") || !strings.Contains(today, "Make it a task") {
		t.Fatalf("Today lists it: %s", truncate(today))
	}
	rec := postForm(t, h, "/mail/task", url.Values{"id": {n.ID}})
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/t/task/") {
		t.Fatalf("made a task: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	tasks, _ := a.Store.List("task", store.ListOptions{})
	made := false
	for _, task := range tasks {
		made = made || task.Fields["title"] == "Dentist on Friday"
	}
	if !made || strings.Contains(get(t, h, "/today").Body.String(), "To sort") {
		t.Error("the task is made and the email is sorted")
	}
}

func emailNotes(t *testing.T, st *store.Store) []*store.Record {
	recs, _ := st.List("email", store.ListOptions{})
	return recs
}
