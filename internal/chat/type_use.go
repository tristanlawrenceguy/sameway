package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A lived-in workspace gathers kinds nobody uses: ones an agent made to
// try something, ones the assistant made and the person never filled.
// The assistant acts on what is in front of it, so each type in the index
// says how much it holds, and an empty one who made it, when the log
// still says; then it can offer to tidy them away.

// fewRecords is "nearly none": below it a type also says when it was last
// written, so one left since a test reads differently from one in use.
const fewRecords = 3

// makerDepth is how far back the log is read for who made an empty type:
// one read, and only when a type is empty.
const makerDepth = 500

// typeUse is how much a type holds, in a few words: its count, when it
// was last written if that is nearly nothing, and who made an empty one.
func (s *Service) typeUse(t *schema.Type, makers func() map[string]string) string {
	n, err := s.Store.Count(t.Name)
	if err != nil {
		return ""
	}
	if n == 0 {
		if by := makers()[t.Name]; by != "" {
			return " No records, made by " + by + "."
		}
		return " No records."
	}
	word := "records"
	if n == 1 {
		word = "record"
	}
	if n >= fewRecords {
		return fmt.Sprintf(" %d %s.", n, word)
	}
	last, err := s.Store.List(t.Name, store.ListOptions{OrderBy: "updated_at", Desc: true, Limit: 1})
	if err != nil || len(last) == 0 {
		return fmt.Sprintf(" %d %s.", n, word)
	}
	return fmt.Sprintf(" %d %s, last changed %s.", n, word, last[0].UpdatedAt.In(s.Now().Location()).Format("2 January 2006"))
}

// typeMakers is who made each type the recent log says was added and
// was not the person: an agent, by name when it gave one, or the
// assistant.
func (s *Service) typeMakers() map[string]string {
	out := map[string]string{}
	entries, err := s.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: makerDepth})
	if err != nil {
		return out
	}
	for _, e := range entries {
		action, _ := e.Fields["action"].(string)
		target, _ := e.Fields["target"].(string)
		if action != "added" || target != "type" {
			continue
		}
		detail, _ := e.Fields["detail"].(string)
		// An older entry named the type test_type, a newer one test type.
		name := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(detail)), " ", "_")
		if _, seen := out[name]; seen {
			continue // the newest entry is who made the type there now
		}
		actor, _ := e.Fields["actor"].(string)
		by, _ := e.Fields["by"].(string)
		switch {
		case actor == records.ActorAgent && by != "" && !records.MachineName(by):
			out[name] = "an agent, " + by
		case actor == records.ActorAgent:
			out[name] = "an agent"
		case actor == "assistant":
			out[name] = "the assistant"
		default:
			out[name] = ""
		}
	}
	return out
}

// lazyMakers reads the log for who made the types at most once, and only
// when a type turns out empty.
func (s *Service) lazyMakers() func() map[string]string {
	var got map[string]string
	return func() map[string]string {
		if got == nil {
			got = s.typeMakers()
		}
		return got
	}
}
