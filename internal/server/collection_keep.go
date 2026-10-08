package server

import (
	"errors"
	"fmt"
	"maps"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Keeping a person's choices on a collection. Show and sort are the
// address's alone (collection_choices.go), which is right for a look, but
// a person who narrows Tasks to what is not done, soonest first, and wants
// it to stay that way had no way to say so short of asking the assistant:
// in the agent evaluation (T5, "only the ones not done, soonest first")
// a person-shaped agent set both choices, read "Showing: not done, due
// soonest first" and took it as done, and nothing was kept.
//
// Keep these choices is a plain POST beside Reset, offered to those who
// may change the block. The choices become the block's own where and
// order: its where grows by them, so what the assistant set still holds
// and the list only narrows, as the choices do; its order is replaced.
// It is logged and undone like any other change to a block, and the page
// comes back at Reset, which is now the kept setup.

// canvasKeep makes the choices a person made on a collection its setup.
func (s *Server) canvasKeep(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(records.BlockType, r.PathValue("id"))
	if err != nil {
		s.failed(w, r, "Not kept", err, "/")
		return
	}
	name, _ := rec.Fields["component"].(string)
	comp, ok := s.app.Registry.Get(name)
	if name != collectionComponent || !ok {
		s.failed(w, r, "Not kept", fmt.Errorf("only a collection keeps its choices, and this block is a %s", name), "/")
		return
	}
	if err := r.ParseForm(); err != nil {
		s.failed(w, r, "Not kept", err, "/")
		return
	}
	props := map[string]any{}
	if current, ok := rec.Fields["props"].(map[string]any); ok {
		maps.Copy(props, current)
	}
	typeName, _ := props["type"].(string)
	t, ok := s.app.Types.Get(typeName)
	if !ok {
		s.failed(w, r, "Not kept", errors.New(s.noType(typeName)), "/")
		return
	}
	// The same choices the page offered, read the same way, so only what
	// they offer can be kept.
	where, order, said, active := blocks.Kept(t, props, rec.ID, r.PostForm)
	if !active {
		s.failed(w, r, "Nothing to keep", errors.New("no choice differs from how the list is set up: choose with Show and sort, Apply, then Keep these choices"), "/")
		return
	}
	list := make([]any, 0, len(where))
	for _, c := range where {
		list = append(list, c)
	}
	if len(list) > 0 {
		props["where"] = list
	}
	if order != "" {
		props["order"] = order
	}
	clean, err := comp.Validate(props)
	if err != nil {
		s.failed(w, r, "Not kept", err, "/")
		return
	}
	if _, err := s.app.Store.Update(records.BlockType, rec.ID,
		s.app.Chat.BlockFields(map[string]any{"props": clean, "actor": "human"})); err != nil {
		s.failed(w, r, "Not kept", err, "/")
		return
	}
	label := str(clean["label"], schema.Plural(t.Name))
	undo := s.record(r, records.Change{
		Action: "updated", Component: name, ID: rec.ID, Detail: label + " kept as " + said, Before: rec.Fields,
	})
	s.tell(w, r, outcome{Title: "Choices kept", Text: capitalize(label) + " now shows " + said + ".", Undo: undo, Of: "keeping " + said}, "/")
}
