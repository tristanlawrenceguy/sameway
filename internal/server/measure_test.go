package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// versionOn is the version a page of Home gives a block, read from the page
// as the browser would.
func versionOn(t *testing.T, h http.Handler, id string) string {
	t.Helper()
	body := get(t, h, "/").Body.String()
	m := regexp.MustCompile(`data-block-id="` + id + `"[^>]*data-measure-v="([0-9a-f]{16})"`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("the owner's Home should mark block %s for measuring:\n%s", id, body)
	}
	return m[1]
}

func measureOf(id, v string, height int) string {
	raw, _ := json.Marshal(map[string]any{"view": "tab", "canvas": "", "vw": 390, "dpr": 3, "font": 16, "cols": 1,
		"blocks": []map[string]any{{"id": id, "v": v, "w": 358, "h": height, "top": 0, "sh": 1240, "sb": 480, "sx": 0, "cut": 0, "d": 0, "past": 0, "pane": 0}}})
	return string(raw)
}

// A page the owner or an editor opens is measured; one someone who may
// only look opens, or the internet reads, is not, and the route refuses
// them, an agent's key, and anything but numbers and ids.
func TestTheMeasuringRouteTakesOnlyNumbersFromThoseWhoMayChange(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	srv := h.(*server.Server)
	id := storedBlock(t, a, "collection", map[string]any{"type": "note", "label": "Reading list"})
	v := versionOn(t, h, id)
	if !strings.Contains(get(t, h, "/").Body.String(), `data-measure="tab"`) {
		t.Error("the owner's Home should be marked for measuring")
	}
	for _, who := range []records.Visitor{{Name: "Vi", Access: records.View}, {Access: records.Public}} {
		if body := as(t, h, who, http.MethodGet, "/", "", "").Body.String(); strings.Contains(body, "data-measure") {
			t.Errorf("a page for %s access should not be measured", who.Access)
		}
		if r := as(t, h, who, http.MethodPost, "/canvas/measure", measureOf(id, v, 600), "application/json"); r.Code != http.StatusForbidden {
			t.Errorf("%s access should be refused a measurement: %d", who.Access, r.Code)
		}
	}
	if r := as(t, h, records.Visitor{Name: "bot", Access: records.Edit, Agent: true}, http.MethodPost, "/canvas/measure", measureOf(id, v, 600), "application/json"); r.Code != http.StatusForbidden {
		t.Errorf("an agent's key measures nothing: %d", r.Code)
	}
	if r := public(t, srv.Public(nil), http.MethodPost, "/canvas/measure", measureOf(id, v, 600)); r.Code != http.StatusMethodNotAllowed {
		t.Errorf("the internet sends nothing: %d", r.Code)
	}

	good := measureOf(id, v, 600)
	for name, body := range map[string]string{
		"words for a number":  strings.Replace(good, `"h":600`, `"h":"tall"`, 1),
		"a fraction of pixel": strings.Replace(good, `"h":600`, `"h":600.5`, 1),
		"words besides":       strings.Replace(good, `"pane":0`, `"pane":0,"label":"Reading list"`, 1),
		"too tall":            strings.Replace(good, `"h":600`, `"h":900000`, 1),
		"below nothing":       strings.Replace(good, `"h":600`, `"h":-5`, 1),
		"a made up id":        strings.Replace(good, id, "<b>hi</b>", 1),
		"another view":        strings.Replace(good, `"view":"tab"`, `"view":"print"`, 1),
		"a tiny screen":       strings.Replace(good, `"vw":390`, `"vw":10`, 1),
		"not json":            "h=600",
	} {
		if r := postJSONRaw(t, h, "/canvas/measure", body); r.Code != http.StatusBadRequest {
			t.Errorf("%s should be refused, got %d", name, r.Code)
		}
	}
	var many []string
	for range 201 {
		many = append(many, `{"id":"`+id+`","v":"`+v+`","w":1,"h":1,"top":0}`)
	}
	if r := postJSONRaw(t, h, "/canvas/measure", `{"view":"tab","canvas":"","vw":390,"blocks":[`+strings.Join(many, ",")+`]}`); r.Code != http.StatusBadRequest {
		t.Errorf("more blocks than a page holds should be refused: %d", r.Code)
	}
	if n := len(a.Chat.MeasuredOn("", "")); n != 0 {
		t.Fatalf("nothing refused should be kept, got %d readings", n)
	}

	if r := postJSONRaw(t, h, "/canvas/measure", measureOf(id, "0123456789abcdef", 600)); r.Code != http.StatusNoContent || len(a.Chat.MeasuredOn("", "")) != 0 {
		t.Errorf("a block drawn before it changed is taken and left out: %d", r.Code)
	}
	if r := postJSONRaw(t, h, "/canvas/measure", good); r.Code != http.StatusNoContent {
		t.Fatalf("the owner's measurement should be kept: %d %s", r.Code, r.Body.String())
	}
	if r := as(t, h, records.Visitor{Name: "Ed", Access: records.Edit}, http.MethodPost, "/canvas/measure", good, "application/json"); r.Code != http.StatusNoContent {
		t.Errorf("an editor's page is measured too: %d", r.Code)
	}
	got := a.Chat.MeasuredOn("", "")
	if len(got) != 1 || got[0].Device != "phone" || got[0].Height != 600 || !got[0].Scrolls() {
		t.Fatalf("one phone reading, scrolling inside, should be kept: %+v", got)
	}

	// /api/look says it, and the layout line uses it.
	var looked struct {
		Measured struct {
			Note   string
			Person []chat.Reading
		}
	}
	decode(t, get(t, h, "/api/look?path=/"), &looked)
	if len(looked.Measured.Person) != 1 || looked.Measured.Person[0].Content != 1240 || !strings.Contains(looked.Measured.Note, "own browser") {
		t.Errorf("a look at Home should give the measurements: %+v", looked.Measured)
	}
	if line := a.Chat.LayoutNow(""); !strings.Contains(line, `"Reading list" scrolls inside on your phone (1,240px of content in a 480px box)`) {
		t.Errorf("the layout line should name the inner scroll:\n%s", line)
	}
}

// With nothing measured, a look says so rather than giving nothing.
func TestALookSaysWhenNothingIsMeasured(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	storedBlock(t, a, "text", map[string]any{"content": "hello"})
	var looked struct {
		Measured struct {
			Note   string
			Person []chat.Reading
		}
	}
	decode(t, get(t, h, "/api/look?path=/"), &looked)
	if !strings.Contains(looked.Measured.Note, "Not measured yet") || looked.Measured.Person == nil {
		t.Errorf("an unmeasured page should say so: %+v", looked.Measured)
	}
	var elsewhere map[string]any
	decode(t, get(t, h, "/api/look?path=/t/note"), &elsewhere)
	if _, ok := elsewhere["measured"]; ok {
		t.Error("a page that is not of blocks has nothing measured")
	}
}

// That the measuring script sends numbers and ids, never words, and nothing
// when a robot drives the page or the browser saves data, is checked in a
// browser: tools/a11y-runner/behave-live.mjs.

func mustHandler(t *testing.T) http.Handler {
	_, h := newApp(t)
	return h
}

func postJSONRaw(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return as(t, h, records.Visitor{Access: records.Owner}, http.MethodPost, path, body, "application/json")
}
