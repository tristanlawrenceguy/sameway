package chat_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// run calls a tool the way MCP does and fails the test on an error.
func run(t *testing.T, svc *chat.Service, name string, args map[string]any) string {
	t.Helper()
	raw, _ := json.Marshal(args)
	text, isErr := svc.Call(name, raw)
	if isErr {
		t.Fatalf("%s refused: %s", name, text)
	}
	return text
}

func refused(t *testing.T, svc *chat.Service, name string, args map[string]any, want string) {
	t.Helper()
	raw, _ := json.Marshal(args)
	text, isErr := svc.Call(name, raw)
	if !isErr || !strings.Contains(text, want) {
		t.Fatalf("%s should be refused saying %q, got %v: %s", name, want, isErr, text)
	}
}

func idOf(text string) string {
	_, after, _ := strings.Cut(text, " as block ")
	id, _, _ := strings.Cut(after, " ")
	return id
}

// weekPage builds the page the haiku runs built for "this week": a
// heading, a list of what is due, a tracker, and what is overdue last.
func weekPage(t *testing.T, svc *chat.Service) (heading, due, tracker, overdue string) {
	heading = idOf(run(t, svc, "add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "This week"}, "span": 6}))
	due = idOf(run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Due this week", "where": []string{"done=false", "due<=+7d"}}, "span": 8}))
	tracker = idOf(run(t, svc, "add_component", map[string]any{"component": "tracker", "props": map[string]any{}, "span": 12}))
	overdue = idOf(run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Overdue", "where": []string{"done=false", "due<today"}}, "span": 6}))
	return
}

func layout(t *testing.T, svc *chat.Service) map[string][2]int64 {
	blocks, _ := svc.Store.List(records.BlockType, store.ListOptions{})
	out := map[string][2]int64{}
	for _, b := range blocks {
		p, _ := b.Fields["position"].(int64)
		s, _ := b.Fields["span"].(int64)
		out[b.ID] = [2]int64{p, s}
	}
	return out
}

func entries(t *testing.T, svc *chat.Service, action string) []*store.Record {
	all, _ := svc.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at"})
	var out []*store.Record
	for _, a := range all {
		if a.Fields["action"] == action {
			out = append(out, a)
		}
	}
	return out
}

// The layout line names what the evaluations found wrong, each with a fix.
func TestEveryBlockWriteSaysHowThePageReads(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	weekPage(t, svc)
	text := run(t, svc, "add_component", map[string]any{"component": "calendar", "props": map[string]any{"type": "task", "detail": "brief"}, "span": 4})
	for _, want := range []string{
		"Layout now: row 1:",
		`"This week" heading 6 (6 of 12 empty)`,
		`row 2: "Due this week" collection 8 (4 of 12 empty), row 3: tracker 12, row 4: "Overdue" collection 6 + calendar of task 4 (2 of 12 empty)`,
		"the section heading \"This week\" is span 6, frame card",
		`"Overdue" is below "Due this week": what is late is the most urgent, put it first`,
		`"Due this week" and "Overdue" both show task, with tracker between them`,
		"calendar of task is brief, a pane size, in the main region",
		"arrange_canvas {\"blocks\":",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("layout line should say %q:\n%s", want, text)
		}
	}
	// Tall beside short, and two headings that say the same.
	svc2 := newFullService(t)
	run(t, svc2, "add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Garden notes"}, "span": 12, "frame": "bare"})
	run(t, svc2, "add_component", map[string]any{"component": "calendar", "props": map[string]any{"type": "task"}, "span": 8})
	run(t, svc2, "add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Seeds"}, "span": 4})
	text = run(t, svc2, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "note", "label": "Garden notes", "where": []string{"tags=garden"}}, "span": 12})
	for _, want := range []string{`calendar of task is tall beside "Seeds"`, `two headings say "Garden notes"`} {
		if !strings.Contains(text, want) {
			t.Errorf("layout line should say %q:\n%s", want, text)
		}
	}
}

// The suggested call is one the tool takes, and it leaves the page with
// full rows, what is late first and related things together.
func TestTheSuggestedArrangementFixesThePage(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	heading, due, tracker, overdue := weekPage(t, svc)
	text := svc.LayoutNow("")
	_, call, ok := strings.Cut(text, "arrange_canvas ")
	if !ok {
		t.Fatalf("no suggestion: %s", text)
	}
	call = call[:strings.Index(call, "}]}")+3]
	var args map[string]any
	if err := json.Unmarshal([]byte(call), &args); err != nil {
		t.Fatalf("the suggestion is not a call: %v: %s", err, call)
	}
	after := run(t, svc, "arrange_canvas", args)
	if !strings.Contains(after, "It reads in order with full rows.") {
		t.Errorf("the suggested arrangement should leave nothing to look at:\n%s", after)
	}
	got := layout(t, svc)
	order := []string{heading, overdue, due, tracker}
	for i, id := range order {
		if got[id][0] != int64(i) {
			t.Errorf("block %d should be at position %d, got %v", i, i, got[id])
		}
	}
}

// One arrangement is one entry and one Undo; undoing the undo puts it back.
func TestArrangeIsOneChangeAndOneUndo(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	heading, due, tracker, overdue := weekPage(t, svc)
	was := layout(t, svc)
	run(t, svc, "arrange_canvas", map[string]any{"blocks": []map[string]any{
		{"id": heading, "span": 12, "frame": "bare"}, {"id": overdue, "span": 12}, {"id": due, "span": 12}, {"id": tracker},
	}})
	now := layout(t, svc)
	if now[overdue] != [2]int64{1, 12} || now[due] != [2]int64{2, 12} || now[heading] != [2]int64{0, 12} {
		t.Fatalf("not arranged: %v", now)
	}
	arranged := entries(t, svc, "arranged")
	if len(arranged) != 1 {
		t.Fatalf("an arrangement should be one entry, got %d", len(arranged))
	}
	if s, _ := arranged[0].Fields["summary"].(string); s != "Assistant arranged Home, 4 blocks" {
		t.Errorf("summary: %q", s)
	}
	run(t, svc, "undo_change", map[string]any{"id": arranged[0].ID})
	back := layout(t, svc)
	for id, v := range was {
		if back[id] != v {
			t.Errorf("undo should put %s back to %v, got %v", id, v, back[id])
		}
	}
	run(t, svc, "undo_change", map[string]any{})
	if again := layout(t, svc); again[overdue] != now[overdue] {
		t.Errorf("undoing the undo should arrange it again: %v", again)
	}
}

// What would lose a block or break the outline is refused whole.
func TestArrangeRefusesWhatLosesABlockOrSkipsAHeading(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	heading, due, tracker, overdue := weekPage(t, svc)
	was := layout(t, svc)
	refused(t, svc, "arrange_canvas", map[string]any{"blocks": []map[string]any{{"id": heading}, {"id": due}, {"id": tracker}}}, "is missing")
	refused(t, svc, "arrange_canvas", map[string]any{"blocks": []map[string]any{{"id": heading}, {"id": due}, {"id": tracker}, {"id": overdue}, {"id": due}}}, "listed twice")
	refused(t, svc, "arrange_canvas", map[string]any{"blocks": []map[string]any{{"id": heading}, {"id": due}, {"id": tracker}, {"id": overdue}, {"id": "nope"}}}, "not a block on this tab")
	refused(t, svc, "arrange_canvas", map[string]any{"blocks": []map[string]any{{"id": heading, "span": 13}, {"id": due}, {"id": tracker}, {"id": overdue}}}, "span must be between 1 and 12")
	// A level 3 list under the heading is fine; moved above it, it skips.
	sub := idOf(run(t, svc, "add_component", map[string]any{"component": "list", "props": map[string]any{"label": "Packing", "items": []string{"a"}, "level": 3}}))
	refused(t, svc, "arrange_canvas", map[string]any{"blocks": []map[string]any{{"id": sub}, {"id": heading}, {"id": due}, {"id": tracker}, {"id": overdue}}}, "skipping level 2")
	delete(layout(t, svc), sub)
	for id, v := range was {
		if layout(t, svc)[id] != v {
			t.Errorf("a refused arrangement must change nothing: %s", id)
		}
	}
	if n := len(entries(t, svc, "arranged")); n != 0 {
		t.Errorf("a refused arrangement must not be logged, got %d", n)
	}
}

// A skipped heading level is refused when a block is written, too.
func TestAHeadingThatSkipsALevelIsRefusedWithTheFix(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	refused(t, svc, "add_component", map[string]any{"component": "list", "props": map[string]any{"label": "Packing", "items": []string{"a"}, "level": 3}}, "Give it level 2, or put a level 2 heading before it")
	id := idOf(run(t, svc, "add_component", map[string]any{"component": "list", "props": map[string]any{"label": "Packing", "items": []string{"a"}, "level": 2}}))
	refused(t, svc, "update_component", map[string]any{"id": id, "props": map[string]any{"label": "Packing", "items": []string{"a"}, "level": 4}}, "skipping level 2")
	// Removing is never refused; the line says what it left.
	h := idOf(run(t, svc, "add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Trip"}, "span": 12, "frame": "bare", "position": -1}))
	run(t, svc, "update_component", map[string]any{"id": id, "props": map[string]any{"label": "Packing", "items": []string{"a"}, "level": 3}})
	if text := run(t, svc, "remove_component", map[string]any{"id": h}); !strings.Contains(text, "skipping level 2") {
		t.Errorf("removing the heading over a level 3 list should say it now skips: %s", text)
	}
}
