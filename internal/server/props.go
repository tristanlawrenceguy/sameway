package server

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// blockProps receives an inline edit: one or more props of a block, sent as
// prop-<name> form values. It is the only way a person changes a block's
// content, and it exists as a plain form post so an agent can use it too.
//
// Anything beyond editing text a person asks the assistant for, which is why
// there is no field here for layout, ids, or anything the component does not
// itself show.
func (s *Server) blockProps(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(chat.BlockType, r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	name, _ := rec.Fields["component"].(string)
	comp, ok := s.app.Registry.Get(name)
	if !ok {
		s.fail(w, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	props := map[string]any{}
	if current, ok := rec.Fields["props"].(map[string]any); ok {
		for k, v := range current {
			props[k] = v
		}
	}
	edited, err := editedFields(r.PostForm)
	if err != nil {
		s.app.Chat.Notice("That edit did not save. " + err.Error())
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	for k, v := range edited {
		props[k] = v
	}
	if len(edited) == 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	clean, err := comp.Validate(props)
	if err != nil {
		// The person is looking at the canvas, so the complaint belongs
		// there, in the conversation, where every other problem is reported.
		s.app.Chat.Notice("That edit did not save. " + err.Error())
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if _, err := s.app.Store.Update(chat.BlockType, rec.ID,
		s.app.Chat.BlockFields(map[string]any{"props": clean, "actor": "human"})); err != nil {
		s.fail(w, err)
		return
	}
	chat.Record(s.app.Store, "human", chat.Change{
		Action: "updated", Component: name, ID: rec.ID, Detail: chat.Summarise(name, clean), Before: rec.Fields,
	})
	http.Redirect(w, r, "/?saved", http.StatusSeeOther)
}
