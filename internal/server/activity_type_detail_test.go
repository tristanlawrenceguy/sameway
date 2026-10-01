package server_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// ActivityDetailIsReadableInAPIAndHTML checks that a type-setting activity
// entry shows a human-readable display name in its detail field on the API,
// not the raw schema identifier. This covers acceptance items 1 and 2: the
// API list must show "Test Type", and the HTML log must also say it — both
// read from the same stored data through Sentence() / Say().
func TestActivityDetailIsReadableInAPIAndHTML(t *testing.T) {
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
}
