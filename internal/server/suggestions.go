package server

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/prose"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The changes suggested to a record's words wait on its page, above its
// fields, for whoever may change it (chat/suggest.go). Each is shown as
// writing: the sentence as it is and as it would read, formatted, never
// Markdown. They are grouped the way an editor would, small fixes first,
// and only fixes and formatting can be accepted all at once: a change to
// how something is said is weighed on its own. Someone who may only look
// sees none of them.

// kinds is the order the groups come in, with their names.
var suggestionKinds = []struct{ key, name, all string }{
	{"fix", "Fixes", "fixes"},
	{"format", "Formatting", "formatting changes"},
	{"clarity", "Clarity", ""},
	{"style", "Style", ""},
	{"structure", "Structure", ""},
}

// suggestionsOn is the section of what waits on a record.
func (s *Server) suggestionsOn(r *http.Request, t *schema.Type, rec *store.Record) string {
	waiting := chat.Suggestions(s.app.Store, t.Name, rec.ID)
	if len(waiting) == 0 || !changes(r) {
		return ""
	}
	from := "/t/" + t.Name + "/" + rec.ID
	var b strings.Builder
	fmt.Fprintf(&b, `<section class="sw-stack" aria-labelledby="suggested"><h2 id="suggested">%s</h2>`, inWords(len(waiting), "suggested change"))
	b.WriteString(`<p class="sw-muted">Nothing changes until you accept one, and each accepted change can be undone.</p>`)
	n := 0
	for _, k := range suggestionKinds {
		var group []*store.Record
		for _, sg := range waiting {
			if kind := str(sg.Fields["kind"], "clarity"); kind == k.key || k.key == "clarity" && !knownKind(kind) {
				group = append(group, sg)
			}
		}
		if len(group) == 0 {
			continue
		}
		fmt.Fprintf(&b, `<h3>%s</h3>`, k.name)
		if k.all != "" && len(group) > 1 {
			fmt.Fprintf(&b, `<form method="post" action="/suggestions/accept-all"><input type="hidden" name="about" value="%s"><input type="hidden" name="kind" value="%s"><input type="hidden" name="from" value="%s">%s</form>`,
				t.Name+"/"+rec.ID, k.key, from, s.component("button", map[string]any{"label": fmt.Sprintf("Accept all %d %s", len(group), k.all), "type": "submit", "variant": "secondary"}))
		}
		for _, sg := range group {
			n++
			b.WriteString(string(s.component("suggestion", s.suggestionProps(rec, sg, fmt.Sprintf("%d of %d", n, len(waiting)), from))))
		}
	}
	b.WriteString(`</section>`)
	return b.String()
}

func knownKind(kind string) bool {
	for _, k := range suggestionKinds {
		if k.key == kind {
			return true
		}
	}
	return false
}

// suggestionProps shows one suggestion in the writing around it.
func (s *Server) suggestionProps(rec, sg *store.Record, label, from string) map[string]any {
	text, _ := rec.Fields[str(sg.Fields["field"], "")].(string)
	passage, replacement := str(sg.Fields["passage"], ""), str(sg.Fields["replacement"], "")
	props := map[string]any{"id": "suggestion-" + sg.ID, "label": label, "why": str(sg.Fields["why"], ""),
		"meaning": sg.Fields["meaning"] == true, "now": "", "nowMark": passage, "from": from, "actor": s.actorOf(sg),
		"accept": "/suggestions/" + sg.ID + "/accept", "decline": "/suggestions/" + sg.ID + "/decline"}
	at := strings.Index(text, passage)
	if at < 0 || strings.Count(text, passage) != 1 {
		props["outdated"] = true
		return props
	}
	start, end := paragraphOf(text, at, at+len(passage))
	props["now"] = text[start:end]
	props["after"] = text[start:at] + replacement + text[at+len(passage):end]
	props["afterMark"] = replacement
	if words, only := prose.FormatChange(passage, replacement); only {
		props["describe"], props["afterMark"] = words, ""
	} else if replacement == "" {
		props["describe"] = "Take these words out"
	}
	if strings.TrimSpace(props["after"].(string)) == "" {
		props["after"] = ""
	}
	return props
}

// paragraphOf is the stretch of text around a passage that reads as its
// own: the sentence it is in, within its paragraph; a passage that is
// more than a sentence, such as lines becoming a list, has its paragraph.
func paragraphOf(text string, from, to int) (int, int) {
	start, end := 0, len(text)
	if i := strings.LastIndex(text[:from], "\n\n"); i >= 0 {
		start = i + 2
	}
	if i := strings.Index(text[to:], "\n\n"); i >= 0 {
		end = to + i
	}
	if i := sentenceEnd.FindAllStringIndex(text[start:from], -1); len(i) > 0 {
		start += i[len(i)-1][1]
	}
	if i := sentenceEnd.FindStringIndex(text[to:end]); i != nil {
		end = to + i[0] + 1
	}
	return start, end
}

// sentenceEnd is where a sentence ends and the next begins.
var sentenceEnd = regexp.MustCompile(`[.!?]["'”’)]?\s+`)

// actorOf is who suggested it, for its colour: an agent, or the assistant.
func (s *Server) actorOf(sg *store.Record) string {
	if w := s.app.Chat.Writers().Of(chat.SuggestionType, sg); strings.Contains(w.Words, "agent") {
		return "agent"
	}
	return "assistant"
}

// suggestionAnswer accepts or declines one suggestion.
func (s *Server) suggestionAnswer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sg, err := s.app.Store.Get(chat.SuggestionType, id)
	if err != nil {
		s.tell(w, r, outcome{Failed: true, Title: "No such suggestion", Text: "It may have been answered already."}, "/")
		return
	}
	about := str(sg.Fields["about"], "")
	if !s.mayAnswer(w, r, about) {
		return
	}
	if r.PathValue("answer") == "decline" {
		if err := chat.DeclineSuggestion(s.app.Store, id); err != nil {
			s.tell(w, r, outcome{Failed: true, Title: "Not declined", Text: err.Error()}, "/t/"+about)
			return
		}
		s.tell(w, r, outcome{Title: "Declined", Text: "Nothing changed. " + s.left(about)}, "/t/"+about)
		return
	}
	s.accept(w, r, about, []string{id})
}

// suggestionsAcceptAll accepts every waiting suggestion of one kind on a
// record, as one change.
func (s *Server) suggestionsAcceptAll(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	about, kind := r.FormValue("about"), r.FormValue("kind")
	if !s.mayAnswer(w, r, about) {
		return
	}
	typ, id, _ := strings.Cut(about, "/")
	var ids []string
	for _, sg := range chat.Suggestions(s.app.Store, typ, id) {
		if sg.Fields["kind"] == kind {
			ids = append(ids, sg.ID)
		}
	}
	s.accept(w, r, about, ids)
}

func (s *Server) accept(w http.ResponseWriter, r *http.Request, about string, ids []string) {
	back := "/t/" + about
	rec, entry, made, outdated, err := chat.AcceptSuggestions(s.app.Store, s.who(r), ids)
	switch {
	case errors.Is(err, chat.ErrOutdated):
		s.tell(w, r, outcome{Failed: true, Title: "Not changed", Text: "The words it would change have changed since, so it no longer fits; it is set aside. " + s.left(about)}, back)
	case err != nil:
		s.tell(w, r, outcome{Failed: true, Title: "Not changed", Text: err.Error()}, back)
	default:
		t, _ := s.app.Types.Get(strings.SplitN(about, "/", 2)[0])
		text := "Accepted. "
		if made > 1 {
			text = fmt.Sprintf("Accepted %d. ", made)
		}
		if outdated > 0 {
			text += fmt.Sprintf("%s no longer fitted and set aside. ", inWords(outdated, "suggestion"))
		}
		s.tell(w, r, outcome{Title: "Changed " + s.title(t, rec), Text: text + s.left(about), Undo: entry, Of: s.title(t, rec)}, back)
	}
}

// left is how many still wait on a record, said after each answer.
func (s *Server) left(about string) string {
	typ, id, _ := strings.Cut(about, "/")
	if n := len(chat.Suggestions(s.app.Store, typ, id)); n > 0 {
		return fmt.Sprintf("%d left.", n)
	}
	return "None left."
}

// mayAnswer refuses a suggestion about the owner's own kinds of record to
// anyone else.
func (s *Server) mayAnswer(w http.ResponseWriter, r *http.Request, about string) bool {
	typ, _, _ := strings.Cut(about, "/")
	t, ok := s.app.Types.Get(typ)
	if !ok || t.Owners && !chat.VisitorOf(r.Context()).Owner() {
		s.tell(w, r, outcome{Failed: true, Title: "Not yours to change"}, "/")
		return false
	}
	return true
}

// inWords is a count in words: 1 suggestion, 3 suggestions.
func inWords(n int, one string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %ss", n, one)
}
