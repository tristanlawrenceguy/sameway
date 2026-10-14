package schema

import (
	"regexp"
	"testing"
)

// A value Sameway gives a name is shown for as long as the record keeps
// it, so its name is true at any age. "Ringing" on a reminder due eight
// days before was the moment named and shown long after it (crew finding
// 0727); the name says what happened ("Rang") and the clock says what is
// happening now. A name that says something is under way is only for a
// value something here ends: the person, or Sameway itself, each listed
// with how, so the next one is a decision and not an accident.
var underWay = regexp.MustCompile(`^(?:[A-Z][a-z]+ing|Being \w+)$`)

var endedBy = map[string]string{
	"task.status.doing":        "the person: it is their word for what they are doing, until they say otherwise",
	"reminder.state.set":       "the clock: it waits until its time, when it rings and becomes rang (server/ring.go)",
	"suggestion.state.pending": "the person: it waits until they take it or turn it down",
	"proposal.state.pending":   "the person: it waits until they answer it",
	"file.status.converting":   "Sameway: when it opens, a file left being read is put right (server/media/left_over.go)",
}

func TestEveryStateNameIsTrueAtAnyAge(t *testing.T) {
	t.Parallel()
	set, err := Load("../../examples/workspaces/starter/schema")
	if err != nil {
		t.Fatal(err)
	}
	for _, ty := range set.Types {
		for _, f := range ty.Fields {
			if f.Name != "status" && f.Name != "state" { // a kind, a tone: not a state
				continue
			}
			for _, v := range f.Values {
				key := ty.Name + "." + f.Name + "." + v
				if underWay.MatchString(f.ValueLabel(v)) && endedBy[key] == "" {
					t.Errorf("%s is named %q, which says it is under way and will read wrong days later: name what happened, or say in endedBy what ends it", key, f.ValueLabel(v))
				}
			}
		}
	}
}
