package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// A template is an arrangement a person applies with one press, from the
// Templates page: a reading list, a job search, a house move. The
// assistant could apply one, if the person knew to ask and it made the
// kinds of record first; a newcomer knows neither. UseTemplate does the
// whole of it in order: the kinds the arrangement needs that the
// workspace lacks (add_type), a tab named for it, and its blocks there,
// each logged as a change of its own, so Activity takes any of them back.

// Template is an arrangement as the Templates page offers it.
type Template struct {
	Name, Title, Description string
	// Makes are the kinds it would add, by name; empty when all are here.
	Makes []string
}

// Templates are the arrangements a person can apply, with what each
// would add to this workspace.
func (s *Service) Templates() []Template {
	var out []Template
	for _, a := range s.Registry.Arrangements() {
		t := Template{Name: a.Name, Title: titleOf(a), Description: a.Description}
		for _, n := range a.Needs {
			if _, ok := s.Store.Types().Get(n.Type); !ok {
				t.Makes = append(t.Makes, n.Type)
			}
		}
		out = append(out, t)
	}
	return out
}

// titleOf is an arrangement's own heading, or its name capitalised.
func titleOf(a *render.Arrangement) string {
	for _, b := range a.Blocks {
		if b.Component == "heading" {
			if t, _ := b.Props["text"].(string); t != "" {
				return t
			}
		}
	}
	return strings.ToUpper(a.Name[:1]) + a.Name[1:]
}

// UseTemplate applies one: the kinds it needs, a tab of its own, its
// blocks there. It answers the new tab's id.
func (s *Service) UseTemplate(name, actor string) (string, error) {
	a, ok := s.Registry.Arrangement(name)
	if !ok {
		return "", fmt.Errorf("there is no template called %q", name)
	}
	log := func(r toolResult) error {
		if r.isErr {
			return fmt.Errorf("%s", r.text)
		}
		if r.change != nil {
			records.Record(s.Store, actor, *r.change)
		}
		for _, c := range r.changes {
			records.Record(s.Store, actor, c)
		}
		return nil
	}
	for _, n := range a.Needs {
		if _, ok := s.Store.Types().Get(n.Type); ok {
			continue
		}
		raw, _ := json.Marshal(n.Fields)
		var defs []fieldDef
		if err := json.Unmarshal(raw, &defs); err != nil {
			return "", err
		}
		if err := log(s.addType(n.Type, n.Description, defs)); err != nil {
			return "", err
		}
	}
	tab := titleOf(a)
	for _, c := range s.Canvases() {
		if strings.EqualFold(c.Name, tab) {
			tab = tab + " " + fmt.Sprint(len(s.Canvases())+1)
		}
	}
	made := s.createCanvas(tab)
	if err := log(made); err != nil {
		return "", err
	}
	on := *s
	on.current = made.change.ID
	if err := log(on.addArrangement(name, nil)); err != nil {
		return "", err
	}
	return made.change.ID, nil
}
