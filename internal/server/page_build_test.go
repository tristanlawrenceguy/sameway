package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// blockPropsOf reads a block's stored props.
func blockPropsOf(t *testing.T, s *store.Store, id string) map[string]any {
	t.Helper()
	rec, err := s.Get(records.BlockType, id)
	if err != nil {
		t.Fatal(err)
	}
	props, _ := rec.Fields["props"].(map[string]any)
	return props
}

// The agent evaluation's T5: "only the ones not done, soonest first, and
// call it Up next". A person narrows and sorts with Show and sort, keeps
// the choices, and renames the list; each is undone from its outcome.
func TestKeepTheseChoicesMakesThemTheSetup(t *testing.T) {
	a, h := newApp(t)
	seedTasks(t, a)
	id := addCollection(t, h, map[string]any{"type": "task", "label": "Tasks", "controls": true})
	chosen := url.Values{"c-" + id + "-done": {"false"}, "c-" + id + "-sort": {"due"}}
	list := section(t, get(t, h, "/?"+chosen.Encode()).Body.String(), id)
	for _, want := range []string{
		`Showing: not done, due soonest first.`,
		`<form method="post" action="/canvas/` + id + `/keep" class="sw-filters__keep">`,
		`<input type="hidden" name="c-` + id + `-done" value="false">`,
		`<input type="hidden" name="from" value="/#collection-` + id + `">`,
		`Keep these choices<span class="sw-visually-hidden"> for Tasks</span></button>`,
	} {
		if !strings.Contains(list, want) {
			t.Errorf("the list offers to keep its choices with %s: %.3000s", want, list)
		}
	}
	if strings.Contains(section(t, get(t, h, "/").Body.String(), id), "Keep these choices") {
		t.Error("nothing to keep before a choice is made")
	}

	form := url.Values{"from": {"/#collection-" + id}}
	for k, v := range chosen {
		form[k] = v
	}
	rec := postForm(t, h, "/canvas/"+id+"/keep", form)
	wantStatus(t, rec, http.StatusSeeOther)
	if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/#collection-"+id) && !strings.HasPrefix(loc, "/?") {
		t.Errorf("back at the list as it is now set up: %q", loc)
	}
	props := blockPropsOf(t, a.Store, id)
	if w, _ := json.Marshal(props["where"]); string(w) != `["done=false"]` || props["order"] != "due" {
		t.Fatalf("the choices are the block's where and order: %v", props)
	}
	list = section(t, get(t, h, "/").Body.String(), id)
	if !strings.Contains(list, "6 tasks, not done, due soonest first.") || strings.Contains(list, "Showing:") || !inOrder(list, "Dig the pond", "Mend the fence", "Order compost") {
		t.Errorf("the list says what it is now set up to show: %.3000s", list)
	}
	if strings.Contains(list, "Call the dentist") {
		t.Error("done tasks are gone from the kept list")
	}

	// Undone, it is as the assistant set it up.
	_, out := agentPost(t, h, "/canvas/"+id+"/keep", map[string]any{"c-" + id + "-sort": "-created_at"})
	undo, _ := out["undo"].(string)
	if out["ok"] != true || undo == "" {
		t.Fatalf("an agent keeps choices too, with an undo: %v", out)
	}
	_, back := agentPost(t, h, undo, map[string]any{})
	if back["ok"] != true {
		t.Fatalf("undo: %v", back)
	}
	if props := blockPropsOf(t, a.Store, id); props["order"] != "due" {
		t.Errorf("undo takes back the last keep: %v", props)
	}
}

// Keep with no choice made says so, rather than saving nothing.
func TestKeepWithNothingChosenSaysSo(t *testing.T) {
	a, h := newApp(t)
	seedTasks(t, a)
	id := addCollection(t, h, map[string]any{"type": "task", "label": "Tasks", "controls": true})
	rec, out := agentPost(t, h, "/canvas/"+id+"/keep", map[string]any{"c-" + id + "-done": "maybe"})
	if rec.Code != http.StatusBadRequest || out["title"] != "Nothing to keep" {
		t.Errorf("keeping no choice is refused in words: %d %v", rec.Code, out)
	}
}

// A collection, a calendar and a chart each say their name can be edited,
// as Name, for the inline editor; saving it renames the block, logged and
// undone like any edit.
func TestABlockIsRenamedFromThePage(t *testing.T) {
	a, h := newApp(t)
	seedTasks(t, a)
	id := addCollection(t, h, map[string]any{"type": "task", "label": "Tasks"})
	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, `<template data-edit-fields><span data-prop="label" data-label="Name" data-source="Tasks"></span></template>`) {
		t.Fatalf("the list's name is marked for the editor: %.4000s", page)
	}
	if !strings.Contains(page, `id="sw-controls"`) {
		t.Error("the page carries the editor's controls")
	}
	_, out := agentPost(t, h, "/canvas/"+id+"/props", map[string]any{"prop-label": "Up next"})
	if out["ok"] != true {
		t.Fatalf("rename: %v", out)
	}
	if !strings.Contains(section(t, get(t, h, "/").Body.String(), id), ">Up next</h") {
		t.Error("the list is headed Up next")
	}
	agentPost(t, h, out["undo"].(string), map[string]any{})
	if blockPropsOf(t, a.Store, id)["label"] != "Tasks" {
		t.Error("undo gives the old name back")
	}

	rec := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "chart", "props": map[string]any{"type": "task", "by": "done"}})
	wantStatus(t, rec, http.StatusCreated)
	if !strings.Contains(get(t, h, "/").Body.String(), `<span data-prop="caption" data-label="Name"`) {
		t.Error("a chart's caption is its name to edit")
	}
}

// A props post that names no prop is refused in words, whoever sends it;
// a JSON body is read as the form whatever the Accept header says. In the
// agent evaluation (C-t5) both got an empty 303 and nothing done.
func TestAPropsPostIsNeverSilent(t *testing.T) {
	a, h := newApp(t)
	id := addCollection(t, h, map[string]any{"type": "task", "label": "Tasks"})

	rec := postForm(t, h, "/canvas/"+id+"/props", url.Values{"label": {"Up next"}, "order": {"due"}})
	wantStatus(t, rec, http.StatusBadRequest)
	if body := rec.Body.String(); !strings.Contains(body, "prop-<name>") || !strings.Contains(body, "send prop-label, prop-order") {
		t.Errorf("says what to send: %q", body)
	}

	rec = postJSON(t, h, http.MethodPost, "/canvas/"+id+"/props", map[string]any{"label": "Up next"})
	var out map[string]any
	if rec.Code != http.StatusBadRequest || json.Unmarshal(rec.Body.Bytes(), &out) != nil || !strings.Contains(out["text"].(string), "send prop-label") {
		t.Errorf("a JSON body without the JSON Accept is answered in JSON: %d %s", rec.Code, rec.Body.String())
	}

	rec = postJSON(t, h, http.MethodPost, "/canvas/"+id+"/props", map[string]any{"prop-label": "Up next"})
	if rec.Code != http.StatusOK || blockPropsOf(t, a.Store, id)["label"] != "Up next" {
		t.Errorf("a JSON body of prop- fields works without the Accept header: %d %s", rec.Code, rec.Body.String())
	}

	// A browser is sent back with the outcome on its page, as forms are.
	req := httptest.NewRequest(http.MethodPost, "/canvas/"+id+"/props", strings.NewReader("label=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther || !strings.Contains(w.Header().Get("Set-Cookie"), "sw-outcome=") {
		t.Errorf("a browser gets the outcome where it was: %d %v", w.Code, w.Header())
	}
}

// Home offers "Skip to latest message" only when that message is on it:
// with the chat block gone from Home, the link led nowhere (axe
// skip-link, in every in-app run of the agent evaluation).
func TestHomeSkipLinkTargetExists(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{{Text: "Hello."}}}, nil
	get(t, h, "/")
	postForm(t, h, "/chat", url.Values{"message": {"hi"}, "from": {"/"}})
	addCollection(t, h, map[string]any{"type": "task", "label": "Tasks"})

	skipTarget := func() (string, bool) {
		page := get(t, h, "/").Body.String()
		i := strings.Index(page, `">Skip to latest message</a>`)
		if i < 0 {
			return "", false
		}
		start := strings.LastIndex(page[:i], `href="#`) + len(`href="#`)
		target := page[start:i]
		return target, strings.Contains(page, `id="`+target+`"`)
	}
	if target, there := skipTarget(); target == "" || !there {
		t.Fatalf("with the chat on Home, the link leads to its latest message: %q %v", target, there)
	}
	blocks, _ := a.Store.List(records.BlockType, store.ListOptions{})
	for _, b := range blocks {
		if b.Fields["component"] == records.ComponentName {
			wantStatus(t, postForm(t, h, "/canvas/"+b.ID+"/delete", url.Values{}), http.StatusSeeOther)
		}
	}
	if target, _ := skipTarget(); target != "" {
		t.Errorf("no chat on Home, no link to its latest message: %q", target)
	}
}

// Someone who may only look is not offered what only changes things.
func TestALookerIsNotOfferedKeepOrRename(t *testing.T) {
	a, h := newApp(t)
	seedTasks(t, a)
	id := addCollection(t, h, map[string]any{"type": "task", "label": "Tasks", "controls": true})
	v := records.Visitor{Login: "vic@example.com", Name: "Vic", Access: records.View}
	page := as(t, h, v, http.MethodGet, "/?c-"+id+"-done=false", "", "").Body.String()
	if !strings.Contains(page, "Showing: not done") {
		t.Fatalf("a looker still narrows: %.2000s", page)
	}
	if strings.Contains(page, "Keep these choices") || strings.Contains(page, "data-edit-fields") {
		t.Error("a looker is offered Keep these choices or the list's name to edit")
	}
}
