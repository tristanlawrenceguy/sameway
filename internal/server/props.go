package server

import (
	"fmt"
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
		s.failed(w, r, "Not saved", err, "/")
		return
	}
	name, _ := rec.Fields["component"].(string)
	comp, ok := s.app.Registry.Get(name)
	if !ok {
		s.failed(w, r, "Not saved", fmt.Errorf("this block is a %s, which this workspace no longer has", name), "/")
		return
	}
	if err := r.ParseForm(); err != nil {
		s.failed(w, r, "Not saved", err, "/")
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
		s.failed(w, r, "Not saved", err, "/")
		return
	}
	for k, v := range edited {
		props[k] = v
	}
	if len(edited) == 0 {
		http.Redirect(w, r, backOf(r, "/"), http.StatusSeeOther)
		return
	}

	clean, err := comp.Validate(props)
	if err != nil {
		// Said where the person is; what they typed waits in the draft.
		s.failed(w, r, "Not saved", err, "/")
		return
	}
	if _, err := s.app.Store.Update(chat.BlockType, rec.ID,
		s.app.Chat.BlockFields(map[string]any{"props": clean, "actor": "human"})); err != nil {
		s.failed(w, r, "Not saved", err, "/")
		return
	}
	undo := s.record(r, chat.Change{
		Action: "updated", Component: name, ID: rec.ID, Detail: chat.Summarise(name, clean), Before: rec.Fields,
	})
	s.tell(w, r, outcome{Title: "Changes saved", Undo: undo}, "/")
}
