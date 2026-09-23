package server_test

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// TestNoConversationalFillerInWorkspaceError checks that when workspace
// creation fails (sibling folder exists), the error page does not use the old
// conversational filler "That did not go through." — it starts directly with
// what went wrong. Covers acceptance item 1.
func TestNoConversationalFillerInWorkspaceError(t *testing.T) {
	_, h := newApp(t)

	// Create one workspace first so we can trigger the sibling-folder error.
	postForm(t, h, "/workspaces/new", url.Values{"name": {"existing"}})

	// Try to create another with the same name — this hits showWorkspaces
	// which previously rendered "That did not go through." as an alert title.
	res := postForm(t, h, "/workspaces/new", url.Values{"name": {"existing"}})
	body := res.Body.String()

	if strings.Contains(body, "That did not go through") {
		t.Errorf("workspace error page must not say \"That did not go through.\",\n"+
			"messages should start directly with what went wrong.\n"+
			"Body: %s", truncate(body))
	}

	_ = h
	_ = res
}

// TestNoConversationalFillerInBlockEditError checks that when a block props
// edit fails validation, the error message in the conversation does not use
// filler like "That edit did not save." — it starts directly with the problem.
// Covers acceptance item 2.
func TestNoFillerInBlockEditError(t *testing.T) {
	h, id := canvasWithABlock(t)

	// Post an empty title to trigger validation failure on a required field.
	postForm(t, h, "/canvas/"+id+"/props", url.Values{"prop-title": {""}})

	var msgs struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/message"), &msgs)

	for _, m := range msgs.Records {
		if m.Fields["role"] == "error" {
			content, _ := m.Fields["content"].(string)
			if strings.Contains(content, "That edit did not save") ||
				strings.Contains(content, "did not save") {
				t.Errorf("block edit error must not use filler phrases like \"That edit did not save.\",\n"+
					"it should start directly with the problem description.\n"+
					"Got: %s", truncate(content))
			}
		}
	}

	_ = h
}

// TestNoTechnicalPrefixInValidationError checks that when validation errors
// reach surfaces that use ValidationError.Error(), they do not begin with
// "invalid record:" — messages start with plain language. Covers acceptance
// item 2.
func TestValidationErrorNoTechnicalPrefix(t *testing.T) {
	newApp(t) // app loaded only for the schema it brings; handler not needed here

	// Create a type in a temp workspace schema dir and trigger validation.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "testtype.yaml"), []byte("name: testtype\nfields:\n  name: {type: string, required: true}\n"), 0o644)

	ws, err := schema.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	builtin, err := schema.LoadFS(fstest.MapFS{
		"schema/block.yaml":    {Data: []byte("name: block\ninternal: true\nfields:\n  component: {type: string, required: true}\n")},
		"schema/canvas.yaml":   {Data: []byte("name: canvas\ninternal: true\nfields:\n  name: {type: string, required: true}\n")},
		"schema/note.yaml":     {Data: []byte("name: note\nfields:\n  title: {type: string}\n  body: {type: text}\n")},
		"schema/activity.yaml": {Data: []byte("name: activity\ninternal: true\ntitle: summary\nfields:\n  summary: {type: string}\n  action: {type: string, required: true}\n")},
	}, "schema")
	if err != nil {
		t.Fatal(err)
	}
	ws.Complete(builtin)

	typ, ok := ws.Get("testtype")
	if !ok {
		t.Fatal("testtype not loaded")
	}

	_, verr := typ.Normalize(map[string]any{}) // missing required "name"
	var ve *schema.ValidationError
	if !errors.As(verr, &ve) {
		t.Fatalf("expected ValidationError, got %v", verr)
	}

	errMsg := ve.Error()
	if strings.HasPrefix(errMsg, "invalid record:") {
		t.Errorf("ValidationError.Error() must not begin with \"invalid record:\",\n"+
			"it should start directly with the plain-language problem description.\n"+
			"Got: %s", truncate(errMsg))
	}

	if !strings.Contains(errMsg, "is required") {
		t.Errorf("ValidationError.Error() must contain \"name is required\",\n"+
			"but got: %s", truncate(errMsg))
	}
}
