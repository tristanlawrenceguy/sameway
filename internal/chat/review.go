package chat

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A small model on the person's computer stopped half way more often than
// it failed: it set two of four priorities and said all were set, made
// the people and said they were on the meeting, moved the tasks and left
// the lunch, wrote a time in UTC and told the person the hour they asked
// for. Each tool had said what it did; the model did not add it up. So a
// turn on a model on this computer that changed something is shown, once, what the
// turn changed in the person's own time, beside what they asked, and
// finishes or answers again. A turn that changed nothing (an answer, or a
// question to the person) is left alone. The check is of this message
// only: told to finish "what the person asked", the model went back to
// an earlier message still waiting for the person's yes and removed a
// block nobody asked it to.

// reviewNote is that word to the model, or "" when the reply stands.
func (s *Service) reviewNote(asked string, changes []records.Change) string {
	if len(changes) == 0 {
		return ""
	}
	var lines []string
	for _, c := range changes {
		for _, op := range c.Ops {
			if line := s.opSaid(op); line != "" {
				lines = append(lines, "- "+line)
			}
		}
	}
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > 20 {
		lines = append(lines[:20], fmt.Sprintf("- and %d more", len(lines)-20))
	}
	return fromSameway("before the person reads your reply, check it against what they asked in this message: " + strconv.Quote(trim.Clip(asked, 300)) + ". This turn changed, in the person's time:\n" + strings.Join(lines, "\n") +
		"\nCompare that with every part of this message and with what your reply says. If something in it is not done, or not as they said it, do it now with the tools. Leave anything asked in earlier messages as it is. Then give your reply again, saying only what was done.")
}

// opSaid is one op in a line: made, changed or removed, the record's
// name, and the fields it now has that it did not have before.
func (s *Service) opSaid(op records.Op) string {
	t, ok := s.Store.Types().Get(op.Type)
	if !ok || op.Type == records.MessageType || op.Type == records.ActivityType {
		return ""
	}
	verb := "changed"
	switch {
	case op.After == nil:
		return fmt.Sprintf("removed %s %q", op.Type, t.Called(op.ID, op.Before))
	case op.Before == nil:
		verb = "made"
	}
	var set []string
	for name, v := range op.After {
		if name == "version" || name == "created_at" || name == "updated_at" || fmt.Sprint(op.Before[name]) == fmt.Sprint(v) {
			continue
		}
		if blankValue(v) {
			if op.Before != nil && !blankValue(op.Before[name]) {
				set = append(set, name+": (emptied)")
			}
			continue
		}
		set = append(set, name+": "+s.valueSaid(t, name, v))
	}
	slices.Sort(set)
	return fmt.Sprintf("%s %s %q: %s", verb, op.Type, t.Called(op.ID, op.After), strings.Join(set, "; "))
}

// valueSaid is a field's value as the person reads it: a day or a time
// in their own words and zone, anything else as written, short.
func (s *Service) valueSaid(t *schema.Type, name string, v any) string {
	if f, ok := t.Field(name); ok && f.Type == "datetime" {
		if str, ok := v.(string); ok && str != "" {
			return when.Text(str, s.Now().Location(), true)
		}
	}
	if str, ok := v.(string); ok {
		return trim.Clip(str, 80)
	}
	b, _ := json.Marshal(v)
	return trim.Clip(string(b), 80)
}

// blankValue is a field with nothing in it: as made, it says nothing.
func blankValue(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case bool:
		return !x
	case []any:
		return len(x) == 0
	case []string:
		return len(x) == 0
	}
	return false
}
