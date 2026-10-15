package chat_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// The system prompt must not leak JSON Schema keywords or internal enum
// values into the LLM's context, because the model repeats them verbatim
// in rendered replies.  These assertions cover:
//   - "(required)" — schema keyword that should never reach a person's eyes
//     (acceptance items 1 and 3)
//   - raw enum values like "draft|published" used as field descriptions
//     (acceptance item 2)

func TestSystemPromptHasNoSchemaKeywords(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{}
	svc.Provider = m
	svc.Send(context.Background(), "hi")
	system := m.seen[0].System

	// "(required)" is a JSON Schema keyword; the prompt should list fields
	// by name, with its label when that is another word: "at (Rings)",
	// never "title (text, required)".
	if strings.Contains(system, ", required)") {
		i := strings.Index(system, ", required)")
		t.Errorf("system prompt leaks the schema keyword 'required': …%s…",
			system[max(i-40, 0):min(i+50, len(system))])
	}

	// Enum values used as kind descriptions — "draft|published" in place of
	// a plain field name — must not appear either.
	if strings.Contains(system, "draft|published") {
		i := strings.Index(system, "draft|published")
		t.Errorf("system prompt leaks raw enum values 'draft|published': …%s…",
			system[max(i-40, 0):min(i+50, len(system))])
	}

	// The fields should still be present as plain names.
	for _, want := range []string{"Content types", "note"} {
		if !strings.Contains(system, want) {
			t.Errorf("prompt lost %q while stripping keywords", want)
		}
	}
}

func TestRefusalMessageHasNoSchemaKeywords(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{}
	svc.Provider = m

	raw, _ := json.Marshal(map[string]any{"type": "note", "fields": map[string]any{"body": "no title"}})
	text, isErr := svc.Call("create_record", raw)

	if !isErr {
		t.Fatal("expected a refusal when required field 'title' is missing")
	}

	// The refusal error tells the model what fields the type takes; it must
	// not contain ", required)" or enum values like "draft|published".
	for _, bad := range []string{", required)", "draft|published"} {
		if strings.Contains(text, bad) {
			i := strings.Index(text, bad)
			t.Errorf("refusal message leaks %q: …%s…", bad, text[max(i-40, 0):min(i+80, len(text))])
		}
	}

	// The field display names should still appear so the model knows what to ask for.
	if !strings.Contains(text, "title") {
		t.Errorf("refusal should still mention 'title': %.400s", text)
	}
}

// A field is listed by the name a model writes it by, with its label when
// that is another word: told only Rings, a model wrote rings for at, and
// was refused.
func TestFieldsAreListedByTheNameTheyAreWrittenBy(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{}
	svc.Provider = m
	svc.Send(context.Background(), "hi")
	system := m.seen[0].System
	for _, want := range []string{"at (Rings)", "event (Came up at)", "title", "status (one of todo, doing, done)"} {
		if !strings.Contains(system, want) {
			t.Errorf("the type list should say %q", want)
		}
	}
}
