package server

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/query"
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

// keepForm is what Keep these choices sends: the picks in the address
// that belong to this block, and the page to come back to.
func keepForm(q url.Values, block, back string) map[string]any {
	prefix := "c-" + block + "-"
	fields := []any{}
	for _, name := range slices.Sorted(maps.Keys(q)) {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		for _, v := range q[name] {
			fields = append(fields, map[string]any{"name": name, "value": v})
		}
	}
	fields = append(fields, map[string]any{"name": "from", "value": back})
	return map[string]any{"action": "/canvas/" + block + "/keep", "fields": fields}
}

// setupWords is what a collection is set up to show, said after how many
// match when no choice is made: "9 tasks, not done, due soonest first".
// Once choices are kept this is where they are read back.
func setupWords(t *schema.Type, where []string, order string, choices []choice) string {
	var said []string
	if w := query.Words(t, where); w != "" {
		said = append(said, w)
	}
	if order != "" && order != baseOrder("") {
		words := strings.TrimPrefix(orderWords(t, order), ", ")
		if len(choices) > 0 {
			for _, o := range choices[0].options {
				if o.value == order && o.label != "As set up" {
					words = lowerFirst(o.label)
				}
			}
		}
		said = append(said, words)
	}
	return strings.Join(said, ", ")
}

// countSaying is how many match, with what the list is set up to show.
func countSaying(t *schema.Type, n int, setup string) string {
	if n == 0 {
		return ""
	}
	if setup == "" {
		return schema.Count(n, t.Name)
	}
	return schema.Count(n, t.Name) + ", " + setup
}

// canvasKeep makes the choices a person made on a collection its setup.
func (s *Server) canvasKeep(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(chat.BlockType, r.PathValue("id"))
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
	where := strs(props["where"])
	order, _ := props["order"].(string)
	by := ""
	if f, err := boardField(t, props["by"]); err == nil && props["as"] == "board" {
		by = f.Name
	}
	// The same choices the page offered, read the same way, so only what
	// they offer can be kept.
	out := map[string]any{"id": "collection-" + rec.ID}
	kept, sorted, active := applyChoices(out, collectionChoices(t, where, order, by), &collectionPlace{Path: "/", Query: r.PostForm}, rec.ID, where, order)
	if !active {
		s.failed(w, r, "Nothing to keep", errors.New("no choice differs from how the list is set up: choose with Show and sort, Apply, then Keep these choices"), "/")
		return
	}
	said, _ := out["choices"].(map[string]any)["showing"].(string)
	list := make([]any, 0, len(kept))
	for _, c := range kept {
		list = append(list, c)
	}
	if len(list) > 0 {
		props["where"] = list
	}
	if sorted != baseOrder(order) {
		props["order"] = sorted
	}
	clean, err := comp.Validate(props)
	if err != nil {
		s.failed(w, r, "Not kept", err, "/")
		return
	}
	if _, err := s.app.Store.Update(chat.BlockType, rec.ID,
		s.app.Chat.BlockFields(map[string]any{"props": clean, "actor": "human"})); err != nil {
		s.failed(w, r, "Not kept", err, "/")
		return
	}
	label := str(clean["label"], schema.Plural(t.Name))
	undo := s.record(r, chat.Change{
		Action: "updated", Component: name, ID: rec.ID, Detail: label + " kept as " + said, Before: rec.Fields,
	})
	s.tell(w, r, outcome{Title: "Choices kept", Text: capitalize(label) + " now shows " + said + ".", Undo: undo, Of: "keeping " + said}, "/")
}
