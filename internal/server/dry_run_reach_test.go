package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// dryRun sends a request as a dry run.
func dryRun(h http.Handler, method, path, body, kind string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", kind)
	req.Header.Set(server.DryRun, "1")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func refusedToTry(res *httptest.ResponseRecorder) bool {
	return res.Code == http.StatusUnprocessableEntity && strings.Contains(res.Body.String(), "cannot_try")
}

// Every route that changes something says whether it stays in the
// workspace, so a new one cannot be tried on a copy while it reaches out;
// and a route whose tool reaches outside is outward too.
func TestEveryChangeSaysWhereItReaches(t *testing.T) {
	for _, rt := range server.Routes() {
		if strings.HasPrefix(rt.Pattern, "GET ") {
			continue
		}
		if rt.Reach == "" {
			t.Errorf("%s does not say whether it reaches outside the workspace (reach in the route table)", rt.Pattern)
		}
		if op, ok := chat.OpFor(rt.Tool); ok && op.OpenWorld && rt.Reach != "outward" {
			t.Errorf("%s does %s, which reaches outside the workspace, but says it is %s", rt.Pattern, rt.Tool, rt.Reach)
		}
	}
}

var pathValue = regexp.MustCompile(`\{[^}]+\}`)

// No route that reaches outside the workspace is tried on a copy.
func TestADryRunTriesNothingOutward(t *testing.T) {
	_, h := newApp(t)
	n := 0
	for _, rt := range server.Routes() {
		if rt.Reach != "outward" {
			continue
		}
		n++
		method, path, _ := strings.Cut(rt.Pattern, " ")
		path = pathValue.ReplaceAllString(path, "x")
		if res := dryRun(h, method, path, "", "application/x-www-form-urlencoded"); !refusedToTry(res) {
			t.Errorf("%s was tried on a copy: %d %.200s", rt.Pattern, res.Code, res.Body.String())
		}
	}
	if n < 30 {
		t.Errorf("only %d routes are outward", n)
	}
	for _, tool := range []string{"run_action", "update_sameway", "add_workspace", "open_workspace", "restore_workspace"} {
		if res := dryRun(h, http.MethodPost, "/api/tools/"+tool, "{}", "application/json"); !refusedToTry(res) {
			t.Errorf("the tool %s was tried on a copy: %d %.200s", tool, res.Code, res.Body.String())
		}
	}
}

// Running an action from the API and sending reminders to a phone were
// tried on a copy, which ran the action and reached ntfy for real.
func TestADryRunDoesNotRunAnActionOrReachAPhone(t *testing.T) {
	a, h := newApp(t)
	if res := dryRun(h, http.MethodPost, "/api/act/anything", "", "application/json"); !refusedToTry(res) {
		t.Errorf("POST /api/act/{id} was tried: %d %s", res.Code, res.Body.String())
	}
	form := url.Values{"set": {"on"}}.Encode()
	if res := dryRun(h, http.MethodPost, "/notify/phone", form, "application/x-www-form-urlencoded"); !refusedToTry(res) {
		t.Errorf("POST /notify/phone was tried: %d %s", res.Code, res.Body.String())
	}
	if got := a.Workspace.Get("notify.phone"); got != "" {
		t.Errorf("a dry run set a phone to send to: %q", got)
	}
}

// What stays in the workspace is still tried, from a page as from the API.
func TestADryRunStillTriesWhatStaysInside(t *testing.T) {
	a, h := newApp(t)
	before, _ := a.Store.Count("note")
	res := dryRun(h, http.MethodPost, "/t/note/add", url.Values{"title": {"Only tried"}}.Encode(), "application/x-www-form-urlencoded")
	if refusedToTry(res) || res.Header().Get(server.DryRun) == "" {
		t.Errorf("adding a note was not tried: %d %q %.200s", res.Code, res.Header().Get(server.DryRun), res.Body.String())
	}
	res = dryRun(h, http.MethodPost, "/api/tools/create_record", `{"type":"note","fields":{"title":"Only tried"}}`, "application/json")
	if refusedToTry(res) || res.Header().Get(server.DryRun) == "" {
		t.Errorf("a tool that stays inside was not tried: %d %.200s", res.Code, res.Body.String())
	}
	if now, _ := a.Store.Count("note"); now != before {
		t.Errorf("a dry run made a note: %d, was %d", now, before)
	}
}
