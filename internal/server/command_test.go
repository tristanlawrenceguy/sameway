package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Pressing a command button before it is accepted takes the person to the
// question, with the command line and "Yes, run it"; Yes runs it and every
// press after that just runs. Something outside presses a button through
// its trigger word, and an unaccepted command stays a question even then.
func TestACommandButtonAsksOnceThenRuns(t *testing.T) {
	_, h := newApp(t)
	created := postJSON(t, h, http.MethodPost, "/api/action", map[string]any{"title": "Say hello", "kind": "command", "command": "echo hello there", "trigger": "long-secret-word"})
	wantStatus(t, created, http.StatusCreated)
	var action struct{ ID string }
	decode(t, created, &action)

	// The press becomes a question the person is taken to.
	rec := postForm(t, h, "/act/"+action.ID, url.Values{"from": {"/t/action/" + action.ID}})
	wantStatus(t, rec, http.StatusSeeOther)
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/t/proposal/") {
		t.Fatalf("an unaccepted command should lead to its question, got %q", loc)
	}
	question := get(t, h, loc).Body.String()
	if !strings.Contains(question, "echo hello there") || !strings.Contains(question, "Yes, run it") || !strings.Contains(question, "runs on your machine") {
		t.Fatalf("the question should show the command line and what Yes means: %.500s", question)
	}
	if logged(t, h, "You ran action Say hello (exit 0)") {
		t.Fatal("nothing should have run yet")
	}

	// From outside, the same command is still a question, not a run.
	rec = postJSON(t, h, http.MethodPost, "/hook/long-secret-word", nil)
	wantStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), "waiting_for") || logged(t, h, "System ran action Say hello (exit 0)") {
		t.Errorf("a trigger cannot run an unaccepted command, got %s", rec.Body.String())
	}

	// Yes accepts and runs; the log says the person did it.
	pid := strings.TrimPrefix(loc, "/t/proposal/")
	wantStatus(t, postForm(t, h, "/proposal/"+pid+"/accept", url.Values{"from": {loc}}), http.StatusSeeOther)
	if !logged(t, h, "You ran action Say hello (exit 0)") {
		t.Error("accepting should run the command under the person's name")
	}
	// Now every press just runs, including from outside.
	rec = postForm(t, h, "/act/"+action.ID, url.Values{"from": {"/"}})
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("an accepted command runs and returns to the page, got %q", loc)
	}
	rec = postJSON(t, h, http.MethodPost, "/hook/long-secret-word", nil)
	wantStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), "hello there") || !logged(t, h, "System ran action Say hello (exit 0)") {
		t.Errorf("the trigger should run the accepted command and answer with what it printed, got %s", rec.Body.String())
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/hook/wrong-word", nil), http.StatusNotFound)

	// A webhook action with a trigger is pressed from outside the same way.
	calls := 0
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; io.WriteString(w, "on") }))
	defer remote.Close()
	chat.HTTPClient = remote.Client()
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{"title": "Alarm", "url": remote.URL, "trigger": "alarm-word"}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/hook/alarm-word", nil), http.StatusOK)
	if calls != 1 {
		t.Errorf("the trigger should call the webhook once, got %d", calls)
	}
}
