package server

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A suggestion on Today, kept as it is or opened to change: either way it
// is a task, linked to what it came from, and that is sorted. Undo in the
// message takes the task back.

// suggestionWords is a suggestion as Today says it: "Pay the water bill,
// due Fri 16 Oct, important, for Ana. Bill due, late fee after."
func (s *Server) suggestionWords(sug *chat.Suggestion) string {
	parts := []string{"Suggested task: " + sug.Title}
	if sug.Due != "" {
		parts = append(parts, "due "+when.Relative(sug.Due, s.now(), s.h24()))
	}
	if sug.Important {
		parts = append(parts, "important")
	}
	if sug.For != "" {
		parts = append(parts, "for "+sug.For)
	}
	out := strings.Join(parts, ", ") + "."
	if why := strings.TrimSpace(sug.Why); why != "" {
		out += " " + strings.TrimSuffix(why, ".") + "."
	}
	return out
}

// keepSuggestion makes the task a suggestion says, and sorts its note.
func (s *Server) keepSuggestion(r *http.Request) (id, title, act string, err error) {
	r.ParseForm()
	note, _, err := s.sortedNote(r, r.PostForm.Get("id"))
	if err != nil {
		return "", "", "", err
	}
	sug := s.suggestionFor(note.ID)
	if sug == nil || !sug.Task {
		sug = &chat.Suggestion{Title: s.nameOf(note)}
	}
	fields := map[string]any{"title": sug.Title}
	if sug.Due != "" {
		fields["due"] = sug.Due
	}
	if sug.Important {
		fields["tags"] = []any{"important"}
	}
	if sug.For != "" {
		if p := s.app.Chat.PersonByName(sug.For); p != nil {
			fields["for"] = p.ID
		}
	}
	if t, ok := s.app.Types.Get("task"); ok {
		if _, ok := t.Field("notes"); ok {
			fields["notes"] = "From [" + s.nameOf(note) + "](/t/" + note.Type + "/" + note.ID + ")."
		}
	}
	rec, act, err := records.WriteAs(s.app.Store, s.who(r), "created", "task", "", fields)
	if err != nil {
		return "", "", "", err
	}
	return rec.ID, sug.Title, act, nil
}

func (s *Server) sortKeep(w http.ResponseWriter, r *http.Request) {
	_, title, act, err := s.keepSuggestion(r)
	if err != nil {
		s.failed(w, r, "Not made", err, "/today")
		return
	}
	s.tellAt(w, r, outcome{Title: "Task made", Text: title + " is a task now.", Undo: act, Of: title}, "/today")
}

func (s *Server) sortChange(w http.ResponseWriter, r *http.Request) {
	id, title, act, err := s.keepSuggestion(r)
	if err != nil {
		s.failed(w, r, "Not made", err, "/today")
		return
	}
	s.tellAt(w, r, outcome{Title: "Task made", Text: title + " is a task; change what you want and save.", Undo: act, Of: title}, "/t/task/"+id+"#edit")
}
