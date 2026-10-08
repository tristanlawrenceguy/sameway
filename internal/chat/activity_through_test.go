package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// ActivitySummariesNeverReferenceTheAPI checks that new activity summaries
// written through WriteAs never contain ", through the API" or "through the
// command line", so headings and body text on /activity read in plain words.
func TestActivitySummariesNeverReferenceTheAPI(t *testing.T) {
	svc := newFullService(t)

	rec, err := svc.Store.Create("note", map[string]any{"title": "Plan"})
	if err != nil {
		t.Fatal(err)
	}

	_, id, _ := records.WriteAs(svc.Store, records.Who{Actor: "human", Via: records.ThroughAPI}, "updated", "note", rec.ID, map[string]any{"title": rec.Fields["title"]})
	if id == "" {
		t.Fatal("expected activity record from WriteAs with ThroughAPI")
	}

	entry, err := svc.Store.Get(records.ActivityType, id)
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

	_, id, _ := records.WriteAs(svc.Store, records.Who{Actor: "human", Via: records.ThroughCLI}, "updated", "note", rec.ID, map[string]any{"title": rec.Fields["title"]})
	if id == "" {
		t.Fatal("expected activity record from WriteAs with ThroughCLI")
	}

	entry, err := svc.Store.Get(records.ActivityType, id)
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

	_, id, _ := records.WriteAs(svc.Store, records.Who{Actor: "human", Via: "pixel-7"}, "updated", "note", rec.ID, map[string]any{"title": rec.Fields["title"]})
	if id == "" {
		t.Fatal("expected activity record from WriteAs")
	}

	got, err := svc.Store.Get(records.ActivityType, id)
	if err != nil {
		t.Fatalf("activity not stored: %v", err)
	}
	summary, _ := got.Fields["summary"].(string)

	want := "You updated note Test, on pixel-7"
	if summary != want {
		t.Errorf("non-through via should say ', on <device>'\nwant: %q\n got: %q", want, summary)
	}
}
