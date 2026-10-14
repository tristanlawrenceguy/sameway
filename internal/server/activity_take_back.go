package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Taking back what an agent did. Agents a person lets in try things: in
// the crew's workspace they left test kinds of thing, notes called Test
// Note, a list of 106 copied titles and twenty rows on Home, among some
// 890 changes. The log could show one agent's changes, and each had its
// own Undo, one press at a time. Shown one agent's changes, the person can
// take all of them back that still can be, newest first, in one press.

// takeBackOffer is the press for the changes shown, when the log is
// narrowed to an agent and some of them can still be undone.
func (s *Server) takeBackOffer(r *http.Request, recs []*store.Record, f logFilter) string {
	if !strings.HasPrefix(f.who, "a-") || !records.VisitorOf(r.Context()).Owner() || len(recs) == 0 {
		return ""
	}
	n := 0
	for _, e := range recs {
		if s.app.Records.Undoable(e) {
			n++
		}
	}
	if n == 0 {
		return ""
	}
	who := s.whoOf(recs[0]).label
	return string(s.form(ui.Form{Action: "/activity/take-back",
		Hidden: append(append(ui.Hidden("who", f.who), ui.Hidden("kind", f.kind)...), ui.Hidden("when", f.when)...),
		Button: &ui.Button{Label: fmt.Sprintf("Take back %s by %s", changesWords(n), who), Variant: ui.Secondary}}))
}

// takeBack undoes, newest first, every change the posted narrowing shows
// that can still be undone, as the person's, and says how many.
func (s *Server) takeBack(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	all, err := s.app.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
	if err != nil {
		s.fail(w, err)
		return
	}
	recs, _, f := s.narrowLog(all, r.PostForm, s.now())
	back := "/activity?" + url.Values{"who": {f.who}, "kind": {f.kind}, "when": {f.when}}.Encode()
	if !strings.HasPrefix(f.who, "a-") || len(recs) == 0 {
		s.failed(w, r, "Nothing taken back", fmt.Errorf("choose an agent in Who to take back what it did"), "/activity")
		return
	}
	who := s.whoOf(recs[0]).label
	done, stuck := 0, 0
	for _, e := range recs {
		if !s.app.Records.Undoable(e) {
			continue
		}
		if err := s.app.Records.UndoAs("human", e.ID); err != nil {
			stuck++
			continue
		}
		done++
	}
	text := fmt.Sprintf("Took back %s by %s.", changesWords(done), who)
	if done == 0 {
		text = "Nothing could be taken back."
	}
	if stuck > 0 {
		text += fmt.Sprintf(" %s could not be: what came after them changed the same things.", changesWords(stuck))
	}
	s.tellAt(w, r, outcome{Title: "Taken back", Text: text}, back)
}
