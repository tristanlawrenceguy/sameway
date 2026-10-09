package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/mailin"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/when"
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
	a.Password = s.keys().Get(mailKey(a.User))
	a.Insecure = mailInsecure
	return a, a.Password != ""
}

// mailInsecure lets a test read from a mail server without TLS.
var mailInsecure = false

func (s *Server) mailPage(w http.ResponseWriter, r *http.Request) {
	esc := template.HTMLEscapeString
	var b strings.Builder
	if a, ok := s.mailAccount(); ok {
		b.WriteString(`<p>Connected to ` + esc(a.User) + `. ` + sendTo(a.User) + ` It comes in within five minutes and is sorted for you on Today. Nothing in your mailbox is marked read, moved or deleted.</p>`) // mail_stories.go
		if at := s.app.Store.Meta("mail:checked"); at != "" {
			b.WriteString(`<p class="sw-small sw-muted">Last looked: ` + esc(at) + `</p>`)
		}
		b.WriteString(string(s.form(ui.Form{Action: "/mail/off", Button: &ui.Button{Label: "Stop reading my email", Variant: ui.Secondary}})))
		s.page(w, r, "Email in", template.HTML(b.String()), pageOptions{})
		return
	}
	b.WriteString(s.mailStoryPage(r)) // mail_stories.go
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
	if err := s.keys().Save(mailKey(user), pass); err != nil {
		s.failed(w, r, "Not connected", errors.New("the password could not be kept: "+err.Error()), "/mail")
		return
	}
	raw, _ := json.Marshal(a)
	s.app.Store.SetMeta("mail:account", string(raw))
	s.setUpSorting() // classify_today.go
	s.app.Store.SetMeta("mail:marks", "")
	n, _ := s.readMail(ctx)
	said := "Move mail to the Sameway folder, made in your mailbox just now, and it comes in as a note."
	if storyFor(user).Plus { // mail_stories.go
		said = "Send or forward mail to " + mailin.PlusAddress(user) + " and it comes in as a note."
	}
	if n > 0 {
		said += " " + schema.Count(n, "email") + " from the last week came in already."
	}
	s.tell(w, r, outcome{Title: "Email connected", Text: said}, "/mail")
}

func (s *Server) mailOff(w http.ResponseWriter, r *http.Request) {
	if a, ok := s.mailAccount(); ok {
		s.keys().Save(mailKey(a.User), "")
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
	mails, marks, err := mailin.New(ctx, a, marks, func(refs []string) bool { return s.threadOf(refs) != "" })
	sort.SliceStable(mails, func(i, j int) bool { return mails[i].Date.Before(mails[j].Date) }) // a reply after what it answers
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
		rec, path, err := s.media.KeepFile(who, bytes.NewReader(f.Data), f.Name, "", "")
		if err != nil {
			continue
		}
		s.media.ReadKept(rec.ID, f.Name, path, false)
		name, _ := rec.Fields["title"].(string)
		parts = append(parts, "["+name+"](/t/"+FileType+"/"+rec.ID+")")
	}
	title := strings.Join(strings.Fields(m.Subject), " ")
	if title == "" {
		title = "Email from " + m.From
	}
	// An email is a record of its own, so one coming in can set off any
	// action (when added, what email); a workspace from before has notes.
	if t, ok := s.app.Types.Get("email"); ok {
		sent := m.Date
		if sent.IsZero() {
			sent = time.Now() // no Date header: when it came
		}
		fields := map[string]any{"subject": clipMail(title, 300), "from": m.From, "received": when.Store(sent, false), "body": strings.Join(parts[1:], "\n\n")}
		if !m.Sent {
			fields["tags"] = []any{toSort} // what the person sent is theirs, not to sort
		}
		if _, ok := t.Field("thread"); ok { // mail_threads.go
			fields["message_id"], fields["from_me"] = m.ID, m.Sent
			if root := s.threadOf(m.Refs); root != "" {
				fields["thread"] = root
			}
		}
		rec, _, err := records.WriteKept(s.app.Store, who, "created", "email", "", fields)
		if root, _ := fields["thread"].(string); err == nil && root != "" {
			s.takeTurn(root, rec) // mail_threads.go
		}
		return err
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

// mailLine is Email in on Help.
func (s *Server) mailLine() string {
	if a, ok := s.mailAccount(); ok {
		return `Email: ` + sendTo(a.User) + ` It is sorted for you on <a class="sw-link" href="/today">Today</a>.` // mail_stories.go
	}
	return `Email: forward a booking, a bill or a note to self and have it here, as a note to sort or a task. <a class="sw-link" href="/mail">Connect your email</a>`
}
