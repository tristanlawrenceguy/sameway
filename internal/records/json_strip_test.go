package records_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Activity entry sanitisation tests: "said" entries must not display raw
// JSON-like syntax. These exercise Say() and Sentence(), the two rendering
// paths, covering acceptance items 1-3.

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// SaySaidEntryWithJSONDetailIsSanitized checks that Say() applies the
// sanitisation when action is "said". (Acceptance 1)
func TestSaySaidEntryWithJSONDetailIsSanitized(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	fields := map[string]any{
		"action": "said",
		"detail": `{"type": "task", "record": "abc123"}`,
	}
	got := records.Say(st, fields)
	if got.Detail == `{"type": "task", "record": "abc123"}` {
		t.Errorf("Say said detail should replace JSON with [content omitted], got %q", got.Detail)
	}
	if got.Detail != "[content omitted]" {
		t.Errorf("Say said detail: got %q, want %q", got.Detail, "[content omitted]")
	}
}

// SaySaidEntryWithNormalTextIsUnchanged checks that normal human text in a
// "said" entry is left alone. (Acceptance 2)
func TestSaySaidEntryWithNormalTextIsUnchanged(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	fields := map[string]any{
		"action": "said",
		"detail": "remind me to take a break",
	}
	got := records.Say(st, fields)
	if got.Detail != "remind me to take a break" {
		t.Errorf("Say said detail: got %q, want unchanged text", got.Detail)
	}
}

// SaySaidEntryMixedTextAndJSONPreservesSurrounding checks that mixed text +
// JSON has surrounding text preserved with only the JSON block replaced.
// (Acceptance 1 & 3)
func TestSaySaidEntryMixedTextAndJSONPreservesSurrounding(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	fields := map[string]any{
		"action": "said",
		"detail": `said: {"type": "task", "record": "abc123"}`,
	}
	got := records.Say(st, fields)
	want := `said: [content omitted]`
	if got.Detail != want {
		t.Errorf("Say said mixed detail: got %q, want %q", got.Detail, want)
	}
}

// SaySaidEntryBracesWithoutQuotesUnchanged checks that curly braces without
// quotes are left alone. (Acceptance 3)
func TestSaySaidEntryBracesWithoutQuotesUnchanged(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	fields := map[string]any{
		"action": "said",
		"detail": "{hello}",
	}
	got := records.Say(st, fields)
	if got.Detail != "{hello}" {
		t.Errorf("Say said braces without quotes: got %q, want unchanged", got.Detail)
	}
}

// SaySaidEntryMultipleJSONBlocksAllReplaced checks that multiple JSON blocks
// are all replaced. (Acceptance 1 & 3)
func TestSaySaidEntryMultipleJSONBlocksAllReplaced(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	fields := map[string]any{
		"action": "said",
		"detail": `{"type":"task","id":"x"} then {"type":"note","id":"y"}`,
	}
	got := records.Say(st, fields)
	want := `[content omitted] then [content omitted]`
	if got.Detail != want {
		t.Errorf("Say said multiple JSON blocks: got %q, want %q", got.Detail, want)
	}
}

// SentenceSaidEntryWithJSONDetailIsSanitized checks that Sentence() applies
// the sanitisation when action is "said". (Acceptance 1 & 4)
func TestSentenceSaidEntryWithJSONDetailIsSanitized(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	fields := map[string]any{
		"actor":  "assistant",
		"action": "said",
		"detail": `{"type": "task", "record": "abc123"}`,
	}
	got := records.Sentence(st, fields)
	if strings.HasPrefix(got, "{") || strings.Contains(got, `{"`) {
		t.Errorf("Sentence said detail must not contain raw JSON: %q", got)
	}
	if !strings.Contains(got, "[content omitted]") {
		t.Errorf("Sentence said detail must contain [content omitted]: %q", got)
	}
}

// SentenceSaidEntryWithNormalTextIsUnchanged checks that normal text in a
// "said" entry is left alone by Sentence(). (Acceptance 2)
func TestSentenceSaidEntryWithNormalTextIsUnchanged(t *testing.T) {
	t.Parallel()
	st := newTestStore(t)
	fields := map[string]any{
		"actor":  "assistant",
		"action": "said",
		"detail": "remind me to take a break",
	}
	got := records.Sentence(st, fields)
	want := "Assistant said remind me to take a break"
	if got != want {
		t.Errorf("Sentence said detail: got %q, want %q", got, want)
	}
}
