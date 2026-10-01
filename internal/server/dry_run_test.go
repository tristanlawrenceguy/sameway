package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// A change sent as a dry run is answered as it would be, refusal and all,
// and the workspace is as it was.
func TestADryRunAnswersAndChangesNothing(t *testing.T) {
	a, h := newApp(t)
	workdir := chat.Workdir
	try := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Sameway-Dry-Run", "1")
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		return res
	}
	before, _ := a.Store.Count("note")
	logged, _ := a.Store.Count(chat.ActivityType)

	res := try(http.MethodPost, "/api/note", `{"title":"Only tried"}`)
	if res.Code != http.StatusCreated || !strings.Contains(res.Body.String(), "Only tried") || res.Header().Get("Sameway-Dry-Run") == "" {
		t.Errorf("a dry run answers as the change would, and says it was tried: %d %q %.200s", res.Code, res.Header().Get("Sameway-Dry-Run"), res.Body.String())
	}
	if res := try(http.MethodPost, "/api/note", `{"colour":"red"}`); res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "colour") {
		t.Errorf("a dry run is refused as the change would be: %d %s", res.Code, res.Body.String())
	}
	if res := try(http.MethodPost, "/act/anything", ""); res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "cannot_try") {
		t.Errorf("what reaches outside is not tried: %d %s", res.Code, res.Body.String())
	}
	if now, _ := a.Store.Count("note"); now != before {
		t.Errorf("a dry run made a note: %d, was %d", now, before)
	}
	if now, _ := a.Store.Count(chat.ActivityType); now != logged {
		t.Errorf("a dry run was logged: %d, was %d", now, logged)
	}
	if chat.Workdir != workdir {
		t.Errorf("trying on a copy moved where commands run to %q", chat.Workdir)
	}
}
