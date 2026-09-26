package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// ActivitySummariesNeverReferenceTheAPI checks that new activity summaries
// produced by summarise() never contain ", through the API" or "through the
// command line", so headings and body text on /activity read in plain words.
func TestActivitySummariesNeverReferenceTheAPI(t *testing.T) {
	svc := newFullService(t)

	rec, err := svc.Store.Create("note", map[string]any{"title": "Plan"})
	if err != nil {
		t.Fatal(err)
	}

	id := chat.RecordWrite(svc.Store, chat.ThroughAPI, "updated", rec, map[string]any{})
	if id == "" {
		t.Fatal("expected activity record from RecordWrite with ThroughAPI")
	}

	entry, err := svc.Store.Get(chat.ActivityType, id)
	if err != nil {
		t.Fatalf("activity not stored: %v", err)
	}
	summary, _ := entry.Fields["summary"].(string)

	if strings.Contains(summary, "through the API") {
		t.Errorf("new summary must not contain 'through the API': %q", summary)
	}
	if strings.Contains(summary, "through the command line") {
		t.Errorf("new summary must not contain 'through the command line': %q", summary)
	}

	wantPrefix := "You updated note Plan"
	if !strings.HasPrefix(strings.TrimSpace(summary), wantPrefix) {
		t.Errorf("summary should start with %q, got %q", wantPrefix, summary)
	}
}

// ActivitySummariesNeverReferenceTheCLI checks that CLI writes also drop the
// machine-language suffix from new summaries.
func TestActivitySummariesNeverReferenceTheCLI(t *testing.T) {
	svc := newFullService(t)

	rec, err := svc.Store.Create("note", map[string]any{"title": "Seed"})
	if err != nil {
		t.Fatal(err)
	}

	id := chat.RecordWrite(svc.Store, chat.ThroughCLI, "updated", rec, map[string]any{})
	if id == "" {
		t.Fatal("expected activity record from RecordWrite with ThroughCLI")
	}

	entry, err := svc.Store.Get(chat.ActivityType, id)
	if err != nil {
		t.Fatalf("activity not stored: %v", err)
	}
	summary, _ := entry.Fields["summary"].(string)

	if strings.Contains(summary, "through the command line") {
		t.Errorf("new summary must not contain 'through the command line': %q", summary)
	}
}

// NonThroughViaStillSaysOn checks that a non-"through" via value still gets
// rendered as ", on <device>".
func TestNonThroughViaStillSaysOn(t *testing.T) {
	svc := newFullService(t)

	rec, err := svc.Store.Create("note", map[string]any{"title": "Test"})
	if err != nil {
		t.Fatal(err)
	}

	id := chat.RecordWrite(svc.Store, "pixel-7", "updated", rec, map[string]any{})
	if id == "" {
		t.Fatal("expected activity record from RecordWrite")
	}

	got, err := svc.Store.Get(chat.ActivityType, id)
	if err != nil {
		t.Fatalf("activity not stored: %v", err)
	}
	summary, _ := got.Fields["summary"].(string)

	want := "You updated note Test, on pixel-7"
	if summary != want {
		t.Errorf("non-through via should say ', on <device>'\nwant: %q\n got: %q", want, summary)
	}
}
