package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// What became of a request, at its own address, for whoever was given
// it: a visitor from the internet, or another person's agent checking
// back. Its key is in the address and is long enough that nobody finds
// one by guessing. It says what was asked and the answer, in Sameway's
// words; after a yes, what was made, as it stands now, and only what they
// asked to set in it, with when it last changed, so whoever asked can keep
// up with it. It goes into their own things the ways Sameway gives any
// record (a calendar, a contact), at the same address with that ending.

const (
	requestPrefix = "/request/"
	requestGet    = "GET /request/{token}"
)

func (s *Server) requestPage(w http.ResponseWriter, r *http.Request) {
	token, ext, _ := strings.Cut(r.PathValue("token"), ".")
	p := s.app.Chat.RequestByToken(token)
	if p == nil {
		http.NotFound(w, r)
		return
	}
	t, made, changed := s.madeFor(p)
	if ext != "" {
		if made == nil && changed == chat.RequestGone && ext == export.ICS.Ext {
			// A calendar keeping it in step takes it away too.
			w.Header().Set("Content-Type", export.ICS.Type)
			export.Calendar(w, s.app.Workspace.Config.Name, nil)
			return
		}
		if made == nil {
			http.NotFound(w, r)
			return
		}
		f, ok := export.ByExt(t, ext)
		if !ok || !takeable(f) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", f.Type)
		web.Attachment(w, "request."+f.Ext)
		export.Write(w, f, t, []*store.Record{made}, s.RefTitle, s.now().Location())
		return
	}
	asked, _ := p.Fields["asked"].(string)
	state := chat.RequestState(p)
	if wantsJSON(r) {
		out := map[string]any{"asked": asked, "state": state, "said": requestWords(state)}
		if words, _ := p.Fields["answer"].(string); words != "" {
			out["words"] = words
		}
		if changed != "" {
			out["changed"] = changed
		}
		if made != nil {
			out["made"] = s.madeWords(t, made)
			for _, f := range export.For(t) {
				if takeable(f) {
					out[f.Ext] = seenFrom(r, chat.RequestPath(token)+"."+f.Ext)
				}
			}
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	esc := template.HTMLEscapeString
	body := template.HTML(`<p>You asked to `+esc(asked)+`.</p>`) + s.part(ui.Status{Message: requestWords(state)})
	if words, _ := p.Fields["answer"].(string); words != "" && state != "waiting" {
		body += template.HTML(`<p>` + esc(words) + `</p>`)
	}
	if made != nil {
		var b strings.Builder
		b.WriteString(`<h2>As it is now</h2><dl>`)
		for _, kv := range s.madeWords(t, made) {
			b.WriteString(`<dt>` + esc(kv[0]) + `</dt><dd>` + esc(kv[1]) + `</dd>`)
		}
		b.WriteString(`</dl>`)
		for _, f := range export.For(t) {
			if takeable(f) {
				b.WriteString(`<p><a href="` + esc(chat.RequestPath(token)+"."+f.Ext) + `" download>Add it to your ` + esc(takeWhere(f)) + `</a></p>`)
			}
		}
		body += template.HTML(b.String())
	} else if changed == chat.RequestGone {
		body += template.HTML(`<p>It has since been taken away.</p>`)
	}
	s.page(w, r, "Your request", body, pageOptions{})
}

// madeFor is what a yes made, as it stands now, holding only the fields
// the request asked to set: what the owner added since stays theirs.
// changed is when it last changed, chat.RequestGone once it is taken
// away, or "" before a yes.
func (s *Server) madeFor(p *store.Record) (t *schema.Type, made *store.Record, changed string) {
	typ, id, ok := strings.Cut(fmt.Sprint(p.Fields["made"]), "/")
	t, known := s.app.Types.Get(typ)
	if chat.RequestState(p) != "yes" || !ok || !known {
		return t, nil, ""
	}
	rec, err := s.app.Store.Get(typ, id)
	if err != nil {
		return t, nil, chat.RequestGone
	}
	action, _ := p.Fields["action"].(map[string]any)
	asked, _ := action["fields"].(map[string]any)
	only := *rec
	only.Fields = map[string]any{}
	for k := range asked {
		if v, has := rec.Fields[k]; has {
			only.Fields[k] = v
		}
	}
	return t, &only, rec.UpdatedAt.UTC().Format(time.RFC3339Nano)
}

// madeWords are the fields of what was made, each its label and value as
// a person reads it, in the order of the type.
func (s *Server) madeWords(t *schema.Type, made *store.Record) [][2]string {
	var out [][2]string
	for _, f := range t.Shown() {
		if v, has := made.Fields[f.Name]; has {
			if words := export.Value(f, v, s.RefTitle, s.now().Location()); words != "" {
				out = append(out, [2]string{f.Display(), words})
			}
		}
	}
	return out
}

// takeable are the ways one thing goes into a person's own apps: a
// spreadsheet is for many.
func takeable(f export.Format) bool { return f == export.ICS || f == export.VCard }

func takeWhere(f export.Format) string {
	if f == export.VCard {
		return "contacts"
	}
	return "calendar"
}

// requestWords is the answer in a sentence.
func requestWords(state string) string {
	switch state {
	case "yes":
		return "They said yes: it is made."
	case "no":
		return "They said no: nothing was made."
	}
	return "It waits for their answer. Come back to this page to see it."
}

// requestsSeen are the requests one question put to the owner, each with
// what was asked and its address: what a question page tells them.
func (s *Server) requestsSeen(r *http.Request, made []*store.Record) []map[string]any {
	var out []map[string]any
	for _, p := range made {
		out = append(out, s.requestSeen(r, p))
	}
	return out
}

// requestSeen is a request as the one who asked has it: what they asked,
// and the address that says what became of it.
func (s *Server) requestSeen(r *http.Request, p *store.Record) map[string]any {
	token, _ := p.Fields["token"].(string)
	return map[string]any{"asked": p.Fields["asked"], "status": seenFrom(r, chat.RequestPath(token))}
}

// seenFrom is an address as the one asking reaches it: at the host they
// came by, over https, which is how the internet reaches a workspace.
func seenFrom(r *http.Request, path string) string {
	if r.Host == "" {
		return path
	}
	return "https://" + r.Host + path
}
