package chat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// A template is an arrangement a person applies with one press, from the
// Templates page: a reading list, a job search, a house move. The
// assistant could apply one, if the person knew to ask and it made the
// kinds of record first; a newcomer knows neither. UseTemplate does the
// whole of it in order: the kinds the arrangement needs that the
// workspace lacks (add_type), then a tab named for it with its blocks
// there. The tab and its blocks are one entry in the log, so one Undo
// takes the whole page back (COGA 4.5.2, let users go back in one step);
// a kind it made stays, empty and harmless, with any records made since.

// Template is an arrangement as the Templates page offers it.
type Template struct {
	Name, Title, Description string
	// Makes are the kinds it would add, by name; empty when all are here.
	Makes []string
	// Shows is what its tab shows, by each block's own heading: the
	// page described in words before it is made, the same for a screen
	// reader as for the eye.
	Shows []string
	// Have is the id of a tab already named for it, when there is one.
	Have string
}

// Used is what UseTemplate made: the tab, by id and name, the kinds it
// added, and the log entry that takes the tab back.
type Used struct {
	Tab, Name, Activity string
	Makes               []string
}

// Templates are the arrangements a person can apply, with what each
// would add to this workspace, in the order of their titles.
func (s *Service) Templates() []Template {
	var out []Template
	for _, a := range s.Registry.Arrangements() {
		t := Template{Name: a.Name, Title: titleOf(a), Description: a.Description, Shows: showsOf(a)}
		for _, n := range a.Needs {
			if _, ok := s.Store.Types().Get(n.Type); !ok {
				t.Makes = append(t.Makes, n.Type)
			}
		}
		for _, c := range s.Canvases() {
			if strings.EqualFold(c.Name, t.Title) {
				t.Have = c.ID
			}
		}
		out = append(out, t)
	}
	sort.SliceStable(out, func(i, j int) bool { return sortKey(out[i].Title) < sortKey(out[j].Title) })
	return out
}

// sortKey is a title as an index would file it: The move under M.
func sortKey(title string) string {
	return strings.TrimPrefix(strings.ToLower(title), "the ")
}

// showsOf is each block's heading, in reading order, but the page's own.
func showsOf(a *render.Arrangement) []string {
	var out []string
	for _, b := range a.Blocks {
		if b.Component == "heading" {
			continue
		}
		for _, k := range []string{"label", "caption", "title"} {
			if t, _ := b.Props[k].(string); t != "" {
				out = append(out, t)
				break
			}
		}
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
// blocks there.
func (s *Service) UseTemplate(name, actor string) (Used, error) {
	a, ok := s.Registry.Arrangement(name)
	if !ok {
		return Used{}, fmt.Errorf("there is no template called %q", name)
	}
	var used Used
	for _, n := range a.Needs {
		if _, ok := s.Store.Types().Get(n.Type); ok {
			continue
		}
		raw, _ := json.Marshal(n.Fields)
		var defs []fieldDef
		if err := json.Unmarshal(raw, &defs); err != nil {
			return Used{}, err
		}
		r := s.addType(n.Type, n.Description, defs)
		if r.isErr {
			return Used{}, fmt.Errorf("%s", r.text)
		}
		records.Record(s.Store, actor, *r.change)
		used.Makes = append(used.Makes, n.Type)
	}
	used.Name = s.freeTabName(titleOf(a))
	made := s.createCanvas(used.Name)
	if made.isErr {
		return Used{}, fmt.Errorf("%s", made.text)
	}
	used.Tab = made.change.ID
	on := *s
	on.current = used.Tab
	blocks := on.addArrangement(name, nil)
	if blocks.isErr {
		records.ApplyOps(s.Store, records.Op{Type: records.CanvasType, ID: used.Tab})
		return Used{}, fmt.Errorf("%s", blocks.text)
	}
	whole := *made.change
	for _, c := range blocks.changes {
		whole.Ops = append(whole.Ops, c.Ops...)
	}
	used.Activity = records.Record(s.Store, actor, whole)
	return used, nil
}

// freeTabName is the title, or the title with the first number that
// makes it a tab name of its own: Job search 2.
func (s *Service) freeTabName(title string) string {
	taken := map[string]bool{}
	for _, c := range s.Canvases() {
		taken[strings.ToLower(c.Name)] = true
	}
	name := title
	for n := 2; taken[strings.ToLower(name)]; n++ {
		name = fmt.Sprintf("%s %d", title, n)
	}
	return name
}
