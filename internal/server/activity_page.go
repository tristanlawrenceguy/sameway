package server

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// An entry in the log has its own page, and it was a record's page like
// any other: its stored fields, read out raw, under a heading in words.
// Action "updated", target "meeting_notes_template", a target id, what it
// was before as a JSON dump, and the entry it undid by its id. The crew's
// people read that page for a day and filed each line of it. What an entry
// holds is for the log to use; its page says what happened, in words: what
// it was about, what that was before, what it took back and how it came.

// activityFacts is an entry as its page says it.
func (s *Server) activityFacts(e *store.Record) []any {
	str := func(k string) string { v, _ := e.Fields[k].(string); return v }
	var items []any
	target, id := str("target"), str("target_id")
	t, isType := s.app.Types.Get(target)

	// What it was about, by its name, leading to it while it is there.
	switch href := s.hrefFor(e); {
	case isType && href != "":
		if rec, err := s.app.Store.Get(t.Name, id); err == nil {
			items = append(items, map[string]any{"label": capitalize(schema.Words(t.Name)), "value": s.title(t, rec), "href": href})
		}
	case href != "":
		items = append(items, map[string]any{"label": "About", "value": records.Sentence(s.app.Store, map[string]any{"action": "", "summary": str("detail")}), "href": href})
	case isType && id != "":
		items = append(items, map[string]any{"label": capitalize(schema.Words(t.Name)), "value": "no longer here"})
	}

	// What it was before, field by field, where that is different now.
	before, _ := e.Fields["before"].(map[string]any)
	switch {
	case str("action") == "set" && before != nil:
		was := fmt.Sprint(before["value"])
		if was == "" || was == "<nil>" {
			was = "not set"
		} else {
			was = capitalize(was)
		}
		items = append(items, map[string]any{"label": capitalize(strings.ToLower(workspace.SettingLabel(target))) + " was", "value": was})
	case isType && before != nil:
		now := map[string]any{}
		if rec, err := s.app.Store.Get(t.Name, id); err == nil {
			now = rec.Fields
		}
		for _, f := range t.Shown() {
			v, had := before[f.Name]
			said := display(f, v)
			if !had || said == "" || records.Print(v) == records.Print(now[f.Name]) {
				continue
			}
			if f.Type == "ref" {
				said = s.RefTitle(f, said)
			}
			items = append(items, map[string]any{"label": f.Display() + " was", "value": said})
		}
	}

	// What it took back, by that change's own words.
	if undoes := str("undoes"); undoes != "" {
		if u, err := s.app.Store.Get(records.ActivityType, undoes); err == nil {
			items = append(items, map[string]any{"label": "It took back", "value": records.Sentence(s.app.Store, u.Fields), "href": "/t/" + records.ActivityType + "/" + u.ID})
		}
	}

	// How it came, when not on this computer.
	if via := str("via"); via != "" {
		if !strings.HasPrefix(via, "through ") {
			via = "on " + via
		}
		items = append(items, map[string]any{"label": "How", "value": capitalize(via)})
	}
	return items
}
