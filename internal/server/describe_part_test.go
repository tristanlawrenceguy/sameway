package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// An agent about to create a note wants the fields of a note, not fifty
// thousand characters of everything. The description is served by part, cut
// by the same Part the MCP server and the command line use, and every wrong
// turn under /api is answered in the same JSON shape as a right one.
func TestDescribeIsReadByPart(t *testing.T) {
	_, h := newApp(t)

	var types []struct {
		Name   string
		Fields []struct{ Name string }
	}
	decode(t, get(t, h, "/api/describe/types"), &types)
	if len(types) == 0 {
		t.Fatal("the types part should list the content types")
	}

	var note struct {
		Name   string
		Fields []struct{ Name string }
	}
	decode(t, get(t, h, "/api/describe/types/note"), &note)
	fields := map[string]bool{}
	for _, f := range note.Fields {
		fields[f.Name] = true
	}
	if note.Name != "note" || !fields["body"] {
		t.Errorf("one type by name should carry its fields, got %+v", note)
	}

	var routes map[string]string
	decode(t, get(t, h, "/api/describe/routes"), &routes)
	for _, want := range []string{"describe", "html_props", "errors"} {
		if routes[want] == "" {
			t.Errorf("the routes part should document %s", want)
		}
	}
	if !strings.Contains(routes["describe"], "/api/describe/types/{name}") {
		t.Errorf("the describe route should say how to read a part: %q", routes["describe"])
	}

	// A wrong part or name says what the right ones are.
	rec := get(t, h, "/api/describe/nope")
	wantStatus(t, rec, http.StatusNotFound)
	var problem struct {
		Error struct{ Code, Message string }
	}
	decode(t, rec, &problem)
	if problem.Error.Code != "not_found" || !strings.Contains(problem.Error.Message, "types, components, arrangements, tools, routes, llm") {
		t.Errorf("an unknown part should name the parts, got %+v", problem)
	}
	decode(t, get(t, h, "/api/describe/types/nope"), &problem)
	if !strings.Contains(problem.Error.Message, "note") {
		t.Errorf("an unknown type should name the types, got %+v", problem)
	}

	// A path nothing serves is still JSON, and points at the routes.
	rec = get(t, h, "/api/note/abc/props")
	wantStatus(t, rec, http.StatusNotFound)
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("an unknown /api path should answer in JSON, got %s", ct)
	}
	decode(t, rec, &problem)
	if !strings.Contains(problem.Error.Message, "/api/describe/routes") {
		t.Errorf("an unknown /api path should say where the routes are, got %+v", problem)
	}
}

// Arrangements are published like components, with their use notes, and
// the assistant's tool for them is on the list an agent reads.
func TestArrangementsAreDescribed(t *testing.T) {
	_, h := newApp(t)
	var week struct {
		Name   string
		Use    struct{ When, Not string }
		Blocks []struct{ Key, Component, Region string }
	}
	decode(t, get(t, h, "/api/describe/arrangements/week"), &week)
	if week.Name != "week" || week.Use.When == "" || len(week.Blocks) != 4 || week.Blocks[1].Region != "right" {
		t.Errorf("the week arrangement should be described whole, got %+v", week)
	}
	var comp struct {
		Use struct{ When string }
	}
	decode(t, get(t, h, "/api/describe/components/card"), &comp)
	if comp.Use.When == "" {
		t.Error("a component's use note should be published")
	}
	var d struct {
		Tools []struct{ Name string }
	}
	decode(t, get(t, h, "/api/describe"), &d)
	found := false
	for _, tool := range d.Tools {
		found = found || tool.Name == "add_arrangement"
	}
	if !found {
		t.Error("add_arrangement should be among the tools")
	}
	if page := get(t, h, "/design").Body.String(); !strings.Contains(page, `id="arrangement-week"`) {
		t.Error("the styleguide should show the arrangements")
	}
}
