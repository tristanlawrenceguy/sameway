package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// A task an agent makes over the API that is called as an open one is
// says so in the answer, with the one already there and what to do; the
// first of its name says nothing.
func TestAPICreateSaysWhatIsCalledTheSame(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	old, err := a.Store.Create("task", map[string]any{"title": "Pick up dry cleaning"})
	if err != nil {
		t.Fatal(err)
	}
	type answer struct {
		ID    string `json:"id"`
		Alike *struct {
			Records []struct {
				ID   string `json:"id"`
				Page string `json:"page"`
			} `json:"records"`
			Note string `json:"note"`
		} `json:"alike"`
	}
	var first answer
	json.Unmarshal(postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Call the bank"}).Body.Bytes(), &first)
	if first.ID == "" || first.Alike != nil {
		t.Errorf("the first of its name says nothing alike: %+v", first)
	}
	var second answer
	made := postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Pick up dry cleaning"})
	json.Unmarshal(made.Body.Bytes(), &second)
	if second.Alike == nil || len(second.Alike.Records) != 1 || second.Alike.Records[0].ID != old.ID || second.Alike.Records[0].Page != "/t/task/"+old.ID {
		t.Fatalf("the answer names the one already open: %s", made.Body)
	}
	if !strings.Contains(second.Alike.Note, "keep_one") {
		t.Errorf("and says what to do: %q", second.Alike.Note)
	}
}
