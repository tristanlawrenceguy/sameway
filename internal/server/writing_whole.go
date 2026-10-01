package server

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/prose"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A piece read as one: its own words, then each part under its title, in
// order, parts of parts a level down; and the same taken away as Markdown
// or Word. Moving a part is one change to the piece's order, undone like
// any other.

// mostDepth is how far down parts of parts are read.
const mostDepth = 4

// wholeMarkdown is a piece and its parts as one piece of Markdown.
func (s *Server) wholeMarkdown(t *schema.Type, piece *store.Record, level int, seen map[string]bool) string {
	seen[piece.ID] = true
	var b strings.Builder
	if text := strings.TrimSpace(s.textOf(t, piece)); text != "" {
		b.WriteString(shiftHeadings(text, level) + "\n\n")
	}
	if level > mostDepth {
		return b.String()
	}
	for _, p := range chat.Parts(s.app.Store, t, piece) {
		if seen[p.ID] {
			continue
		}
		fmt.Fprintf(&b, "%s %s\n\n", strings.Repeat("#", level), s.title(t, p))
		b.WriteString(s.wholeMarkdown(t, p, level+1, seen))
	}
	return b.String()
}

// shiftHeadings puts a part's own headings under its title.
func shiftHeadings(md string, level int) string {
	lines := strings.Split(md, "\n")
	for i, l := range lines {
		if n := len(l) - len(strings.TrimLeft(l, "#")); n > 0 && n < len(l) && l[n] == ' ' {
			lines[i] = strings.Repeat("#", min(n+level-1, 6)) + l[n:]
		}
	}
	return strings.Join(lines, "\n")
}

// wholePage reads a piece as one, or gives it as a file (?as=md or docx).
func (s *Server) wholePage(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok || !chat.Organised(t) {
		http.NotFound(w, r)
		return
	}
	piece, err := s.app.Store.Get(t.Name, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	title := s.title(t, piece)
	md := s.wholeMarkdown(t, piece, 2, map[string]bool{})
	switch r.URL.Query().Get("as") {
	case "md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		attachment(w, title+".md")
		fmt.Fprintf(w, "# %s\n\n%s", title, md)
		return
	case "docx":
		var buf bytes.Buffer
		if err := export.DOCX(&buf, title, s.lang(), md); err != nil {
			s.fail(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		attachment(w, title+".docx")
		w.Write(buf.Bytes())
		return
	}
	href := "/t/" + t.Name + "/" + piece.ID
	var b strings.Builder
	fmt.Fprintf(&b, `<p><a class="sw-link" href="%s">Back to %s</a></p>`, href, template.HTMLEscapeString(title))
	fmt.Fprintf(&b, `<p class="sw-muted">%s.</p>`, inWords(chat.WordCount(md), "word"))
	fmt.Fprintf(&b, `<article class="sw-prose">%s</article>`, prose.Render(md, 2))
	b.WriteString(string(s.component("export", map[string]any{"what": "all of " + title, "items": []any{
		map[string]any{"href": href + "/whole?as=docx", "format": "docx"},
		map[string]any{"href": href + "/whole?as=md", "format": "md"},
	}})))
	s.page(w, r, title, template.HTML(b.String()), pageOptions{})
}

// moveParts moves one part up or down its piece, as one change.
func (s *Server) moveParts(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	t, ok := s.app.Types.Get(r.PathValue("type"))
	back := "/t/" + r.PathValue("type") + "/" + r.PathValue("id")
	if !ok || !chat.Organised(t) {
		http.NotFound(w, r)
		return
	}
	piece, err := s.app.Store.Get(t.Name, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	parts := chat.Parts(s.app.Store, t, piece)
	at := slices.IndexFunc(parts, func(p *store.Record) bool { return p.ID == r.FormValue("part") })
	to := at - 1
	if r.FormValue("dir") == "down" {
		to = at + 1
	}
	if at < 0 || to < 0 || to >= len(parts) {
		s.tell(w, r, outcome{Failed: true, Title: "Not moved", Text: "That part cannot go further that way."}, back)
		return
	}
	parts[at], parts[to] = parts[to], parts[at]
	order := make([]any, len(parts))
	for i, p := range parts {
		order[i] = p.ID
	}
	_, entry, err := chat.WriteAs(s.app.Store, s.who(r), "updated", t.Name, piece.ID, map[string]any{chat.PartsOrder: order})
	if err != nil {
		s.tell(w, r, outcome{Failed: true, Title: "Not moved", Text: err.Error()}, back)
		return
	}
	name := s.title(t, parts[to])
	s.tell(w, r, outcome{Title: fmt.Sprintf("Moved %s to part %d of %d", name, to+1, len(parts)), Undo: entry, Of: "the move"}, back)
}
