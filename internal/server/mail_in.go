package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/mailin"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Email in: what comes by email (a booking, a bill, a note to self) was
// copied over by hand or forgotten. Now the owner connects their mailbox
// once with an app password, and mail they send or forward to their
// +sameway address, or move to a folder called Sameway, becomes a note
// tagged to sort. Today lists them, each a press from being a task or
// done with; the assistant finds them like any note. The password stays
// in the keys file beside the model's, never in the workspace.

// toSort is the tag a note from email has until it is dealt with.
const toSort = "to sort"

var mailMu sync.Mutex // one read of the mailbox at a time

func mailKey(user string) string { return "SAMEWAY_MAIL_PASSWORD " + strings.ToLower(user) }

// mailAccount is the mailbox connected, if any.
func (s *Server) mailAccount() (mailin.Account, bool) {
	var a mailin.Account
	if json.Unmarshal([]byte(s.app.Store.Meta("mail:account")), &a) != nil || a.User == "" {
		return a, false
	}
	a.Password = llm.Key(mailKey(a.User))
	a.Insecure = mailInsecure
	return a, a.Password != ""
}

// mailInsecure lets a test read from a mail server without TLS.
var mailInsecure = false

var mailHow = []struct{ service, how string }{
	{"Gmail", "Turn on 2-Step Verification, then make one at myaccount.google.com/apppasswords."},
	{"iCloud", "At account.apple.com, Sign-In and Security, App-Specific Passwords. iCloud has no + addresses: move mail to a folder called Sameway instead."},
	{"Yahoo", "At login.yahoo.com/account/security, Generate app password."},
	{"Fastmail", "Settings, Privacy & Security, App passwords, with IMAP access."},
}

func (s *Server) mailPage(w http.ResponseWriter, r *http.Request) {
	esc := template.HTMLEscapeString
	var b strings.Builder
	if a, ok := s.mailAccount(); ok {
		b.WriteString(`<p>Connected to ` + esc(a.User) + `. Send or forward mail to <strong>` + esc(mailin.PlusAddress(a.User)) + `</strong>, or move it to a folder called ` + mailin.Folder + `, and it comes in as a note within five minutes. Nothing in your mailbox is marked read, moved or deleted.</p>`)
		if at := s.app.Store.Meta("mail:checked"); at != "" {
			b.WriteString(`<p class="sw-small sw-muted">Last looked: ` + esc(at) + `</p>`)
		}
		b.WriteString(`<form method="post" action="/mail/off">` + string(s.component("button", map[string]any{"label": "Stop reading my email", "type": "submit", "variant": "secondary"})) + `</form>`)
		s.page(w, r, "Email in", template.HTML(b.String()), pageOptions{})
		return
	}
	b.WriteString(`<p>Connect your mailbox once, and anything you send or forward to your own address with +sameway after the name (you+sameway@gmail.com) comes in as a note, with its attachments. Only that mail is read, and nothing is changed in your mailbox.</p>`)
	b.WriteString(`<p>It needs an app password: one your mail service makes for a program, so your own password is never given.</p><dl class="sw-stack">`)
	for _, h := range mailHow {
		b.WriteString(`<dt><strong>` + esc(h.service) + `</strong></dt><dd>` + esc(h.how) + `</dd>`)
	}
	b.WriteString(`</dl><form method="post" action="/mail/connect" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Your email address", "name": "user", "type": "email", "required": true, "autocomplete": "email", "spellcheck": false})))
	b.WriteString(string(s.component("text-field", map[string]any{"label": "App password", "name": "password", "type": "password", "required": true, "autocomplete": "off", "spellcheck": false, "hint": "Kept on this computer only, outside the workspace."})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Connect", "type": "submit"})) + `</form>`)
	s.page(w, r, "Email in", template.HTML(b.String()), pageOptions{Lede: "Mail you forward becomes notes and tasks."})
}

func (s *Server) mailConnect(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	user := strings.ToLower(strings.TrimSpace(r.PostForm.Get("user")))
	pass := strings.ReplaceAll(strings.TrimSpace(r.PostForm.Get("password")), " ", "") // Gmail shows it in fours
	host := strings.TrimSpace(r.PostForm.Get("host"))
	if host == "" {
		host = mailin.Host(user)
	}
	if host == "" || pass == "" {
		s.failed(w, r, "Not connected", errors.New("give your email address and an app password"), "/mail")
		return
	}
	a := mailin.Account{Host: host, User: user, Password: pass, Insecure: mailInsecure}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := mailin.Check(ctx, a); err != nil {
		s.failed(w, r, "Not connected", err, "/mail")
		return
	}
	if err := llm.SaveKey(mailKey(user), pass); err != nil {
		s.failed(w, r, "Not connected", errors.New("the password could not be kept: "+err.Error()), "/mail")
		return
	}
	raw, _ := json.Marshal(a)
	s.app.Store.SetMeta("mail:account", string(raw))
	s.app.Store.SetMeta("mail:marks", "")
	n, _ := s.readMail(ctx)
	said := "Send or forward mail to " + mailin.PlusAddress(user) + " and it comes in as a note."
	if n > 0 {
		said += " " + schema.Count(n, "email") + " from the last week came in already."
	}
	s.tell(w, r, outcome{Title: "Email connected", Text: said}, "/mail")
}

func (s *Server) mailOff(w http.ResponseWriter, r *http.Request) {
	if a, ok := s.mailAccount(); ok {
		llm.SaveKey(mailKey(a.User), "")
	}
	s.app.Store.SetMeta("mail:account", "")
	s.tell(w, r, outcome{Title: "Email no longer read", Text: "The notes that came in stay."}, "/mail")
}

// KeepMail reads the mailbox every five minutes while Sameway runs.
func (s *Server) KeepMail(ctx context.Context) {
	go func() {
		tick := time.NewTicker(5 * time.Minute)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				c, cancel := context.WithTimeout(ctx, 2*time.Minute)
				if _, err := s.readMail(c); err != nil {
					log.Printf("email: %v", err)
				}
				cancel()
			}
		}
	}()
}

// readMail brings in the new mail as notes, and says how many.
func (s *Server) readMail(ctx context.Context) (int, error) {
	a, ok := s.mailAccount()
	if !ok {
		return 0, nil
	}
	mailMu.Lock()
	defer mailMu.Unlock()
	marks := map[string]mailin.Mark{}
	json.Unmarshal([]byte(s.app.Store.Meta("mail:marks")), &marks)
	mails, marks, err := mailin.New(ctx, a, marks)
	raw, _ := json.Marshal(marks)
	s.app.Store.SetMeta("mail:marks", string(raw))
	s.app.Store.SetMeta("mail:checked", time.Now().Format("2 Jan 15:04"))
	n := 0
	for _, m := range mails {
		key := "mail:got:" + hashOf(m.ID)
		if s.app.Store.Meta(key) != "" {
			continue
		}
		if s.mailNote(m) == nil {
			s.app.Store.SetMeta(key, "1")
			n++
		}
	}
	return n, err
}

// mailNote is one email as a note to sort, its files kept beside it.
func (s *Server) mailNote(m mailin.Mail) error {
	who := records.Who{Actor: "system", Via: "from email"}
	text := m.Text
	if h := m.HTML(); h != "" {
		text, _ = convert.HTMLToMarkdown(h)
	}
	parts := []string{"From " + m.From + ", " + m.Date.Local().Format("Mon 2 Jan 2006 15:04")}
	if t := strings.TrimSpace(text); t != "" {
		parts = append(parts, clipMail(t, 40000))
	}
	for _, f := range m.Files {
		if f.Data == nil {
			parts = append(parts, f.Name+" (too large to keep)")
			continue
		}
		rec, path, err := s.keepFile(who, bytes.NewReader(f.Data), f.Name, "", "")
		if err != nil {
			continue
		}
		s.readKept(rec.ID, f.Name, path, false)
		name, _ := rec.Fields["title"].(string)
		parts = append(parts, "["+name+"](/t/"+FileType+"/"+rec.ID+")")
	}
	title := strings.Join(strings.Fields(m.Subject), " ")
	if title == "" {
		title = "Email from " + m.From
	}
	_, _, err := records.WriteAs(s.app.Store, who, "created", "note", "", map[string]any{
		"title": clipMail(title, 200), "body": strings.Join(parts, "\n\n"), "tags": []any{"email", toSort}})
	return err
}

func clipMail(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return strings.TrimSpace(string(r[:n-1])) + "…"
	}
	return s
}

func (s *Server) mailRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /mail", s.mailPage)
	m.HandleFunc("POST /mail/connect", s.mailConnect)
	m.HandleFunc("POST /mail/off", s.mailOff)
	m.HandleFunc("POST /mail/task", s.mailTask) // mail_sort.go
	m.HandleFunc("POST /mail/sorted", s.mailSorted)
}

// mailLine is Email in on Help.
func (s *Server) mailLine() string {
	if a, ok := s.mailAccount(); ok {
		return `Email: mail sent or forwarded to ` + template.HTMLEscapeString(mailin.PlusAddress(a.User)) + ` comes in as a note to sort, on <a class="sw-link" href="/today">Today</a>. <a class="sw-link" href="/mail">Email in</a>`
	}
	return `Email: forward a booking, a bill or a note to self and have it here, as a note to sort or a task. <a class="sw-link" href="/mail">Connect your email</a>`
}
