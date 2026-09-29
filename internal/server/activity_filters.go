package server

import (
	stdcmp "cmp"
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The activity log narrowed: who made a change, what it was to, and when,
// chosen together and applied at once (the filters component as a form).
// The choices are made from the log itself, only people and kinds that
// are in it, and live in the address (?who=assistant&kind=task&when=week),
// so the page needs no script and a narrowed log is a link.

// activityParams are the log's own address fields; page is left out when
// the choices change, since page 3 of one log is not page 3 of another.
var activityParams = []string{"who", "kind", "when"}

// activityPageSize is how many changes one page of the log shows.
const activityPageSize = 200

// logChoice is one thing a change can be narrowed by, and what it is
// called: you, the assistant, Sam; tasks, cards; today.
type logChoice struct{ value, label string }

// whoOf is who made a change: you, the assistant, the system, an agent
// by its name, or another person by the name the person component shows them by. Another person's
// value is a short fingerprint of their login, so the address says who
// without carrying an email.
func (s *Server) whoOf(e *store.Record) logChoice {
	switch actor, _ := e.Fields["actor"].(string); actor {
	case "assistant":
		return logChoice{"assistant", "Assistant"}
	case "system":
		return logChoice{"system", "System"}
	case chat.ActorAgent:
		// An agent outside Sameway, by the name it gave or its program's.
		name, _ := e.Fields["by"].(string)
		sum := sha256.Sum256([]byte(name))
		return logChoice{"a-" + hex.EncodeToString(sum[:4]), stdcmp.Or(name, "An agent")}
	case "human", "":
		name, _ := s.whoDid(e)
		if name == "" {
			return logChoice{"you", "You"}
		}
		login, _ := e.Fields["by_login"].(string)
		by, _ := e.Fields["by"].(string)
		sum := sha256.Sum256([]byte(strings.ToLower(stdcmp.Or(login, by))))
		return logChoice{"p-" + hex.EncodeToString(sum[:4]), name}
	default:
		return logChoice{actor, capitalize(actor)}
	}
}

// kindOf is what a change was to: a kind of record or block by its plural,
// a setting, or something else.
func kindOf(e *store.Record) logChoice {
	target, _ := e.Fields["target"].(string)
	switch {
	case strings.HasPrefix(target, "ui.") || strings.HasPrefix(target, "llm."):
		return logChoice{"settings", "Settings"}
	case target == "":
		return logChoice{"other", "Other changes"}
	case target == chat.CanvasType:
		return logChoice{target, "Tabs"}
	}
	return logChoice{target, capitalize(plural(target))}
}

// whenSince is how far back a time a log is narrowed to goes: the start
// of today, or of the day six days before it, for the last 7 days.
func whenSince(when string, now time.Time) (time.Time, bool) {
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	switch when {
	case "today":
		return today, true
	case "week":
		return today.AddDate(0, 0, -6), true
	}
	return time.Time{}, false
}

// logFilter is what the address picked, checked against what the log
// offers: a value it does not offer counts as not picked.
type logFilter struct{ who, kind, when string }

func (f logFilter) active() bool { return f.who != "" || f.kind != "" || f.when != "" }

// narrowLog reads the choices from the address, keeps the changes that
// meet all of them, and fills the filters component's props: the selects
// made from what is in the log, how many match in words, what is shown
// and the way back to all of it.
func (s *Server) narrowLog(all []*store.Record, q url.Values, now time.Time) ([]*store.Record, map[string]any, logFilter) {
	var whos, kinds []logChoice
	seen := map[string]bool{}
	for _, e := range all {
		if w := s.whoOf(e); !seen["who:"+w.value] {
			seen["who:"+w.value] = true
			whos = append(whos, w)
		}
		if k := kindOf(e); !seen["kind:"+k.value] {
			seen["kind:"+k.value] = true
			kinds = append(kinds, k)
		}
	}
	// You first, then the assistant and the system, then others by name.
	rank := map[string]int{"you": 0, "assistant": 1, "system": 2}
	slices.SortStableFunc(whos, func(a, b logChoice) int {
		ra, oka := rank[a.value]
		rb, okb := rank[b.value]
		if !oka {
			ra = 3
		}
		if !okb {
			rb = 3
		}
		return stdcmp.Or(stdcmp.Compare(ra, rb), strings.Compare(a.label, b.label))
	})
	slices.SortStableFunc(kinds, func(a, b logChoice) int { return strings.Compare(a.label, b.label) })
	whens := []logChoice{{"today", "Today"}, {"week", "Last 7 days"}}

	pick := func(name string, opts []logChoice) string {
		v := q.Get(name)
		for _, o := range opts {
			if o.value == v {
				return v
			}
		}
		return ""
	}
	f := logFilter{who: pick("who", whos), kind: pick("kind", kinds), when: pick("when", whens)}
	since, dated := whenSince(f.when, now)
	var out []*store.Record
	for _, e := range all {
		if f.who != "" && s.whoOf(e).value != f.who || f.kind != "" && kindOf(e).value != f.kind || dated && e.CreatedAt.Before(since) {
			continue
		}
		out = append(out, e)
	}

	var said []string
	sel := func(id, name, label, anyLabel, picked string, opts []logChoice, say func(logChoice) string) map[string]any {
		options := []any{map[string]any{"value": "", "label": anyLabel, "selected": picked == ""}}
		for _, o := range opts {
			options = append(options, map[string]any{"value": o.value, "label": o.label, "selected": o.value == picked})
			if o.value == picked {
				said = append(said, say(o))
			}
		}
		return map[string]any{"id": id, "name": name, "label": label, "options": options}
	}
	choices := []any{
		sel("activity-who", "who", "Who", "Anyone", f.who, whos, func(o logChoice) string {
			if o.value == "you" {
				return "by you"
			}
			if o.value == "assistant" || o.value == "system" {
				return "by the " + strings.ToLower(o.label)
			}
			return "by " + o.label
		}),
		sel("activity-kind", "kind", "What", "Anything", f.kind, kinds, func(o logChoice) string {
			if o.value == "other" || o.value == "settings" {
				return strings.ToLower(o.label)
			}
			return "to " + strings.ToLower(o.label)
		}),
		sel("activity-when", "when", "When", "Any time", f.when, whens, func(o logChoice) string {
			if o.value == "today" {
				return "today"
			}
			return "in the last 7 days"
		}),
	}
	// The page's other address fields, but not the page number: a new
	// narrowing starts at its first page.
	keep := url.Values{}
	for name, vals := range q {
		if name != "page" && !slices.Contains(activityParams, name) {
			keep[name] = vals
		}
	}
	var hidden []any
	for _, name := range slices.Sorted(maps.Keys(keep)) {
		for _, v := range keep[name] {
			hidden = append(hidden, map[string]any{"name": name, "value": v})
		}
	}
	props := map[string]any{
		"shape": "form", "label": "Show", "action": "/activity", "choices": choices,
		"count": changesWords(len(out)),
	}
	if hidden != nil {
		props["keep"] = hidden
	}
	if f.active() {
		reset := "/activity"
		if len(keep) > 0 {
			reset += "?" + keep.Encode()
		}
		props["showing"], props["reset"] = strings.Join(said, ", "), reset
	}
	return out, props, f
}

// changesWords says how many changes: 1 change, 12 changes, none.
func changesWords(n int) string {
	switch n {
	case 0:
		return ""
	case 1:
		return "1 change"
	}
	return strconv.Itoa(n) + " changes"
}
