package chat

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// One name for a thing, and one sentence for a change, whoever asks: the
// page's heading, the API's title, the assistant, MCP and the command
// line all call these. There used to be two ways to name a record, one
// here and one in the pages, which disagreed on a record with no title;
// and the log's sentences were stored once and then cleaned on some
// surfaces and not others, so an entry the list said in words the entry's
// own page and the API said in the keys and descriptions it was written
// in. Worked out from the fields each time, an old entry reads like a new
// one wherever it is read.

// Name is what a record is called: an entry by its habit and how much, a
// change in the log by its sentence, anything else by its title field, or
// else the first thing it says, or else its kind and id. It is the whole
// name; a window title or a line of a list trims it.
func Name(st *store.Store, t *schema.Type, rec *store.Record) string {
	switch t.Name {
	case EntryType:
		habit, unit := "", ""
		if id, _ := rec.Fields["habit"].(string); id != "" {
			ht, ok := st.Types().Get("habit")
			if h, err := st.Get("habit", id); ok && err == nil {
				habit = Name(st, ht, h)
				unit, _ = h.Fields["unit"].(string)
			}
		}
		return track.EntryName(habit, unit, rec.Fields)
	case ActivityType:
		if said := Sentence(st, rec.Fields); said != "" {
			return said
		}
	}
	return t.Called(rec.ID, rec.Fields)
}

// recordTitle is a record's name short enough for a sentence: a title
// field trimmed; an entry's name, which keeps its amount, as it is.
func recordTitle(st *store.Store, t *schema.Type, rec *store.Record) string {
	if t.Name == EntryType || t.Name == ActivityType {
		return Name(st, t, rec)
	}
	return trim.Title(Name(st, t, rec))
}

// Sentence is a change in the log said in words, from what the entry
// holds: who, what they did, to what. An undo names the change it took
// back by that change's own sentence, so it reads as well as the change
// did, however long ago either was written.
func Sentence(st *store.Store, f map[string]any) string {
	str := func(k string) string { s, _ := f[k].(string); return s }
	actor := str("actor")
	who := whoDid(actor, str("by"), str("via"))
	if undoes := str("undoes"); undoes != "" {
		if o, err := st.Get(ActivityType, undoes); err == nil {
			if again, _ := o.Fields["undoes"].(string); again != "" {
				if p, err := st.Get(ActivityType, again); err == nil {
					return who + " put back: " + Sentence(st, p.Fields)
				}
			} else {
				return who + " undid: " + Sentence(st, o.Fields)
			}
		}
		// The change it took back is gone: its sentence is what is left.
		return CleanSummary(str("summary"))
	}
	c := Change{Action: str("action"), Component: str("target"), Detail: str("detail"), Via: str("via")}
	// A type or a field is said in words, as it is everywhere else: an
	// entry written before that said test_type.
	switch c.Component {
	case "type":
		c.Detail = schema.DisplayName(c.Detail)
	case "field":
		name, of, _ := strings.Cut(c.Detail, " on ")
		c.Detail = schema.DisplayName(name)
		if of != "" {
			c.Detail += " on " + schema.DisplayName(of)
		}
	}
	if c.Action == "" {
		return CleanSummary(str("summary"))
	}
	if isSettingChange(c) {
		return settingSummary(who, c.Component, c.Detail)
	}
	parts := []string{who, c.Action}
	if c.Component != "" {
		parts = append(parts, schema.Words(personWord(c.Component)))
	}
	if c.Detail != "" {
		parts = append(parts, c.Detail)
	}
	said := strings.Join(parts, " ")
	if actor != ActorAgent && c.Via != "" && !strings.HasPrefix(c.Via, "through ") {
		said += ", on " + c.Via
	}
	return said
}

// whoDid is who made a change, as its sentence starts: You, Assistant,
// System, a person by name, or an agent by its name and way in.
func whoDid(actor, by, via string) string {
	switch {
	case actor == ActorAgent:
		return AgentWho(by, via)
	case actor == "human" && by != "":
		return by
	}
	if who := map[string]string{"human": "You", "assistant": "Assistant", "system": "System"}[actor]; who != "" {
		return who
	}
	return actor
}

// Resay rewrites any stored sentence that is not what Sentence says of
// its entry now, so what reads the stored words (the API's fields, an
// export, another tool) reads what every page says. Entries written
// before settings had names are the ones it finds; it returns how many.
func Resay(st *store.Store) int {
	if _, ok := st.Types().Get(ActivityType); !ok {
		return 0
	}
	all, err := st.List(ActivityType, store.ListOptions{OrderBy: "created_at"})
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range all {
		said := Sentence(st, e.Fields)
		if was, _ := e.Fields["summary"].(string); said != "" && said != was {
			if _, err := st.Update(ActivityType, e.ID, map[string]any{"summary": said}); err == nil {
				n++
			}
		}
	}
	return n
}
