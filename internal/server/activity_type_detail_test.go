package server_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// ActivityDetailIsReadableInAPIAndHTML checks that a type-setting activity
// entry shows a human-readable display name in its detail field on the API,
// not the raw schema identifier. This covers acceptance items 1 and 2: the
// API list must show "Test Type", and the HTML log must also say it — both
// read from the same stored data through Sentence() / Say().
func TestActivityDetailIsReadableInAPIAndHTML(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	// Create a type-setting entry with raw identifiers.
	entry, err := a.Store.Create("activity", map[string]any{
		"actor":   "system",
		"action":  "added",
		"target":  "type",
		"detail":  "test_type",
		"summary": "System added type test_type",
	})
	if err != nil {
		t.Fatal(err)
	}

	list := get(t, h, "/t/activity").Body.String()

	// The HTML log must not contain the raw identifier.
	if strings.Contains(list, "test_type") {
		t.Errorf("the HTML activity log still shows the raw identifier %q\n%s", "test_type", truncate(list))
	}

	// The HTML log must show the resolved name in the event sentence.
	if !strings.Contains(said(list), "added type Test Type") && !anyH3Says(list, "System added type Test Type") {
		t.Errorf("the HTML activity log should say 'added type Test Type'\n%s", truncate(list))
	}

	// The API list must also not contain the raw identifier in the title.
	apiList := get(t, h, "/api/activity").Body.String()
	var apiResp map[string]any
	json.Unmarshal([]byte(apiList), &apiResp)
	recsArr, ok := apiResp["records"].([]any)
	if !ok || len(recsArr) == 0 {
		t.Fatal("no activity records returned from API")
	}

	// Find our entry in the list by its id.
	var myRec map[string]any
	for _, r := range recsArr {
		rec, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if rid, _ := rec["id"].(string); rid == entry.ID {
			myRec = rec
			break
		}
	}
	if myRec == nil {
		t.Fatal("our activity entry not found in API list")
	}

	title := myRec["title"].(string)
	if strings.Contains(title, "test_type") || !strings.Contains(title, "Test Type") {
		t.Errorf("the API list title should show 'Test Type', not raw identifier; got %q", title)
	}

	// The entry's own page must also resolve the identifier.
	entryPage := get(t, h, "/t/activity/"+entry.ID).Body.String()
	if strings.Contains(said(entryPage), "test_type") {
		t.Errorf("the entry's detail page still shows the raw identifier %q\n%s", "test_type", truncate(entryPage))
	}

	// Acceptance 1: detail field must show display-form name, not raw id.
	fields := myRec["fields"].(map[string]any)
	detail, _ := fields["detail"].(string)
	if detail == "test_type" {
		t.Errorf("acceptance 1: API list detail is still the raw identifier %q; want 'Test Type'; full record: %#v", detail, myRec)
	}

	// Acceptance 2: summary field must be cleaned for type-setting entries.
	summary, _ := fields["summary"].(string)
	if strings.Contains(summary, "test_type") {
		t.Errorf("acceptance 2: API list summary still has raw identifier %q; want 'System added type Test Type'; got %q", "test_type", summary)
	}

	// Acceptance 3: non-type-setting entries must be unaffected.
	a.Store.Create("activity", map[string]any{
		"actor":   "assistant",
		"action":  "created",
		"target":  "note",
		"detail":  store.NewID(),
		"summary": "Assistant created note Plan",
	})
	apiList2 := get(t, h, "/api/activity").Body.String()
	var apiResp2 map[string]any
	json.Unmarshal([]byte(apiList2), &apiResp2)
	recsArr2, ok := apiResp2["records"].([]any)
	if !ok {
		t.Fatal("no records in second API list")
	}
	for _, r := range recsArr2 {
		rec, ok := r.(map[string]any)
		if !ok {
			continue
		}
		target, _ := rec["target"].(string)
		if target == "note" {
			sum, _ := rec["summary"].(string)
			if sum != "Assistant created note Plan" {
				t.Errorf("acceptance 3: non-type entry summary was changed to %q; expected 'Assistant created note Plan'", sum)
			}
		}
	}
}

// TestAPIActivityDetailFieldResolvesTypeIdentifier checks that GET
// /api/activity/{id} resolves raw schema identifiers in the detail field
// when the entry targets a content type (target == "type"). This is the
// single-record API endpoint — the list and HTML pages already resolve it.
// Acceptance item 1: the JSON response must show display-form names, not
// raw identifiers like test_type.
func TestAPIActivityDetailFieldResolvesTypeIdentifier(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	// Create a type-setting entry with raw identifiers.
	entry, err := a.Store.Create("activity", map[string]any{
		"actor":   "system",
		"action":  "added",
		"target":  "type",
		"detail":  "test_type",
		"summary": "System added type test_type",
	})
	if err != nil {
		t.Fatal(err)
	}

	var one map[string]any
	json.Unmarshal(get(t, h, "/api/activity/"+entry.ID).Body.Bytes(), &one)
	fields, _ := one["fields"].(map[string]any)
	detail, ok := fields["detail"].(string)
	if !ok || detail != "Test Type" {
		t.Errorf("GET /api/activity/{id} for a type-setting entry should resolve the detail field to display name 'Test Type', got %q (ok=%v)", detail, ok)
	}

	// Non-type entries must keep their raw fields unchanged.
	setEntry, err := a.Store.Create("activity", map[string]any{
		"actor":   "assistant",
		"action":  "set",
		"target":  "ui.text",
		"detail":  "large",
		"summary": "Assistant changed text size to Large",
	})
	if err != nil {
		t.Fatal(err)
	}

	var setOne map[string]any
	json.Unmarshal(get(t, h, "/api/activity/"+setEntry.ID).Body.Bytes(), &setOne)
	setFields, _ := setOne["fields"].(map[string]any)
	setDetail, ok2 := setFields["detail"].(string)
	if !ok2 || setDetail != "large" {
		t.Errorf("non-type entries must keep raw detail; expected 'large', got %q (ok=%v)", setDetail, ok2)
	}
}
