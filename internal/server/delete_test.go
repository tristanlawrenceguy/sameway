package server_test

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// Deleting is one step, because it can be taken back: no page asks "are
// you sure"; the listing the person lands on says what happened and
// offers to put it back.
func TestDeletingIsOneStepAndReversible(t *testing.T) {
	_, h := newApp(t)
	created := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Water the plants", "body": "Sunday."})
	var note struct{ ID string }
	decode(t, created, &note)

	detail := get(t, h, "/t/note/"+note.ID).Body.String()
	if !strings.Contains(detail, `action="/t/note/`+note.ID+`/delete"`) || strings.Contains(detail, "confirm-delete") {
		t.Fatal("the detail page should delete in one step, with no confirmation page")
	}
	wantStatus(t, get(t, h, "/t/note/"+note.ID+"/confirm-delete"), http.StatusNotFound)

	rec := postForm(t, h, "/t/note/"+note.ID+"/delete", url.Values{})
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, "deleted") {
		t.Fatalf("the confirmation page should say deleted; body: %s", truncate(body))
	}
	listing := get(t, h, "/t/note").Body.String()
	undo := regexp.MustCompile(`action="(/activity/[^/"]+/undo)"`).FindStringSubmatch(listing)
	if undo == nil || !strings.Contains(listing, `name="from" value="/t/note"`) {
		t.Fatal("the listing should offer to put the deleted note back")
	}
	wantStatus(t, postForm(t, h, undo[1], url.Values{"from": {"/t/note"}}), http.StatusSeeOther)
	if rec := get(t, h, "/api/note/"+note.ID); rec.Code != http.StatusOK {
		t.Error("undo should bring the note back")
	}
}
