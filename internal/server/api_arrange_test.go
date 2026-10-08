package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Over REST, a block write answers with the layout line, and POST
// /api/arrange lays out the tab in one logged, undoable change.
func TestTheAPIArrangesATabInOneChange(t *testing.T) {
	a, h := newApp(t)
	agent := map[string]string{"X-Sameway-Agent": "layout-test"}
	var ids []string
	for _, body := range []string{
		`{"component":"heading","props":{"text":"Week"},"span":6}`,
		`{"component":"collection","props":{"type":"task","label":"Overdue","where":["due<today"]},"span":6}`,
	} {
		rec := apiAs(t, h, http.MethodPost, "/api/block", body, agent)
		wantStatus(t, rec, http.StatusCreated)
		var out struct {
			ID     string `json:"id"`
			Layout string `json:"layout"`
		}
		json.Unmarshal(rec.Body.Bytes(), &out)
		if !strings.HasPrefix(out.Layout, "Layout now: row 1:") {
			t.Errorf("a block write should answer with the layout, got %q", rec.Body.String())
		}
		ids = append(ids, out.ID)
	}
	rec := apiAs(t, h, http.MethodPost, "/api/arrange", `{"blocks":[{"id":"`+ids[0]+`"}]}`, agent)
	wantStatus(t, rec, http.StatusUnprocessableEntity)
	if !strings.Contains(rec.Body.String(), "is missing") {
		t.Errorf("leaving a block out is refused: %s", rec.Body.String())
	}
	rec = apiAs(t, h, http.MethodPost, "/api/arrange", `{"canvas":"","blocks":[{"id":"`+ids[0]+`","span":12,"frame":"bare"},{"id":"`+ids[1]+`","span":12}]}`, agent)
	wantStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), "It reads in order with full rows.") {
		t.Errorf("the answer should say how the page reads now: %s", rec.Body.String())
	}
	e := newestEntry(t, a.Store)
	if e.Fields["action"] != "arranged" || e.Fields["summary"] != "layout-test (through the API) arranged Home, 2 blocks" {
		t.Errorf("one entry, the agent's: %v", e.Fields)
	}
	if err := a.Chat.UndoAs("human", e.ID); err != nil {
		t.Fatal(err)
	}
	blk, _ := a.Store.Get(records.BlockType, ids[0])
	if blk.Fields["span"] != int64(6) {
		t.Errorf("undo should put the span back: %v", blk.Fields["span"])
	}
	idx := apiAs(t, h, http.MethodGet, "/api/describe", "", nil).Body.String()
	if !strings.Contains(idx, "block_arrange: POST /api/arrange") {
		t.Error("the describe index should say how to arrange a tab")
	}
}
