package chat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/automate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// "When a task tagged client is marked done, send its title to ...": a
// small model made the webhook and left out when it runs, so it ran only
// when pressed, and sent {{name}}, which a task does not have. An action
// made to run only when pressed says how to make it run on its own,
// for the model to judge against what the person said; one whose words fill a name the record it watches does not have
// says which names it has.

// actionSaid is that note on a new action, or "".
func (s *Service) actionSaid(t *schema.Type, rec *store.Record) string {
	if t.Name != "action" {
		return ""
	}
	out := ""
	_, said := s.latestAsk()
	lower := strings.ToLower(" " + said)
	when, _ := rec.Fields["when"].(string)
	every, _ := rec.Fields["every"].(string)
	if when == "" && (every == "" || every == "never") {
		out += " It runs only when pressed. If the person said when it should run, set when (added, changed or removed), what (the kind it watches, such as task) and only (the conditions, such as status=done or tags=client) with update_record, or every for a time."
	}
	what, _ := rec.Fields["what"].(string)
	if what == "" {
		what = s.kindSaid(lower)
	}
	wt, ok := s.Store.Types().Get(what)
	if !ok {
		return out
	}
	have := map[string]bool{"title": true, "type": true, "id": true, "page": true}
	for _, f := range wt.Fields {
		have[f.Name] = true
	}
	var missing []string
	for _, field := range []string{"url", "body", "payload", "message", "command"} {
		v, _ := rec.Fields[field].(string)
		for _, m := range automate.Placeholder.FindAllStringSubmatch(v, -1) {
			if !have[m[1]] {
				missing = append(missing, "{{"+m[1]+"}}")
			}
		}
	}
	if len(missing) > 0 {
		var names []string
		for n := range have {
			names = append(names, "{{"+n+"}}")
		}
		sort.Strings(names)
		out += fmt.Sprintf(" A %s has nothing called %s: what it can send is %s.", schema.Words(what), strings.Join(missing, ", "), schema.Names(names))
	}
	return out
}

// kindSaid is the kind of record the person's words name, or "".
func (s *Service) kindSaid(lower string) string {
	for _, t := range s.Store.Types().Types {
		if t.Internal {
			continue
		}
		if strings.Contains(lower, " "+schema.Words(t.Name)+" ") || strings.Contains(lower, " "+schema.Words(t.Name)+"'s ") {
			return t.Name
		}
	}
	return ""
}
