package automate

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// What a picture's reading could not make out is written [unclear]
// (pictures_read.go). An action set off on its own that would send such a
// part (an amount, a date, a name) is held and put to the owner, as the
// one who can check it: put right on the record's page, Run it runs the
// action with the record as it is then. A guess is never sent on its own.

var unclearMark = regexp.MustCompile(`\[unclear[^\]]*\]`)

// unclearIn is the line of what an action would send that has a part
// read unsure, or "".
func unclearIn(f *store.Record) string {
	for _, k := range []string{"url", "body", "payload", "message"} {
		text, _ := f.Fields[k].(string)
		for _, line := range strings.Split(text, "\n") {
			if unclearMark.MatchString(line) {
				return trim.Clip(strings.TrimSpace(line), 160)
			}
		}
	}
	return ""
}

// held asks the owner before an action set off on its own runs with
// something unclear in it, once for each record; held is false when it
// may run.
func (s *Service) held(rec, f *store.Record, vars map[string]string) (r Result, held bool) {
	what := unclearIn(f)
	if what == "" {
		return Result{}, false
	}
	title, _ := rec.Fields["title"].(string)
	if s.asking("run_held", rec.ID, vars["id"]) {
		return Result{Text: title + " is held until " + vars["title"] + " is checked; the question is waiting for the owner"}, true
	}
	q := Question{"Check “" + vars["title"] + "” before “" + title + "” runs?",
		fmt.Sprintf("Part of what it would send could not be read for sure: %s. Nothing was sent. Put it right on %s, then press Run it: it runs with that as it is then.", what, vars["page"]),
		"Run it", "Not now"}
	r = s.Chat.Ask(q, map[string]any{"tool": "run_held", "id": rec.ID, "type": vars["type"], "record": vars["id"], "shown": Shown(rec)})
	if !r.IsErr {
		r.Text = title + " was held, as part of what it would send could not be read for sure; " + r.Text
	}
	return r, true
}

// RunHeld is a held action's Run it: run with the record as it is now,
// unless what it would send is still unclear.
func (s *Service) RunHeld(ctx context.Context, id, typ, recID string) Result {
	rec, err := s.Store.Get(records.ActionType, id)
	if err != nil {
		return fail("no action with id %s", id)
	}
	if r, asked := s.unaccepted(rec); asked {
		return r
	}
	t, ok := s.Store.Types().Get(typ)
	if !ok {
		return fail("nothing ran: there is no kind of record %s", typ)
	}
	r, err := s.Store.Get(typ, recID)
	if err != nil {
		return fail("nothing ran: the record it was for is gone")
	}
	vars := s.recordVars(t, r)
	f := filled(rec, vars)
	if what := unclearIn(f); what != "" {
		return fail("nothing ran: %s still has a part not read for sure (%s); put it right on %s first", vars["title"], what, vars["page"])
	}
	return s.runRecord(context.WithValue(ctx, triggerKey{}, vars), f, "")
}
