package server_test

import (
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

func withPhase(t *testing.T, a *app.App) {
	t.Helper()
	if _, err := a.AddField("note", schema.Field{Name: "phase", Type: "enum", Values: []string{"draft", "done"}}); err != nil {
		t.Fatal(err)
	}
}

func change(t *testing.T, a *app.App, args map[string]any) string {
	t.Helper()
	raw, _ := json.Marshal(args)
	text, isErr := a.Chat.Call("change_field", raw)
	if isErr {
		t.Fatalf("change_field %v: %s", args, text)
	}
	return text
}

// A pick-list gets a choice, a field and a choice get new labels, and a
// field is hidden: off the pages, out of the assistant's hands, no longer
// asked for, with everything it held kept and back when it is shown.
func TestAFieldChangesWithoutLosingAnything(t *testing.T) {
	a, h := newApp(t)
	withPhase(t, a)
	change(t, a, map[string]any{"type": "note", "field": "phase", "change": "add_choice", "value": "blocked", "label": "Waiting on someone"})
	change(t, a, map[string]any{"type": "note", "field": "phase", "change": "label", "label": "Stage"})
	n, err := a.Store.Create("note", map[string]any{"title": "Plan", "phase": "blocked"})
	if err != nil {
		t.Fatalf("the new choice can be chosen: %v", err)
	}
	page := get(t, h, "/t/note/"+n.ID).Body.String()
	if !strings.Contains(page, "Stage") || !strings.Contains(page, "Waiting on someone") {
		t.Errorf("the page shows the field and the choice by their new labels:\n%s", truncate(page))
	}

	change(t, a, map[string]any{"type": "note", "field": "phase", "change": "hide"})
	if page := get(t, h, "/t/note/"+n.ID).Body.String(); strings.Contains(page, "Waiting on someone") {
		t.Error("a hidden field is off the page")
	}
	note, _ := a.Types.Get("note")
	if _, offered := note.JSONSchema()["properties"].(map[string]any)["phase"]; offered {
		t.Error("a hidden field is not offered to the assistant")
	}
	if got, _ := a.Store.Get("note", n.ID); got.Fields["phase"] != "blocked" {
		t.Errorf("hiding keeps what it held: %v", got.Fields["phase"])
	}
	change(t, a, map[string]any{"type": "note", "field": "phase", "change": "show"})
	if page := get(t, h, "/t/note/"+n.ID).Body.String(); !strings.Contains(page, "Waiting on someone") {
		t.Error("shown again, it is all back")
	}
}

// Deleting is asked, with hiding offered first; hiding instead keeps
// everything, and deleting clears it. What Sameway relies on is refused.
func TestDeletingIsAskedWithHidingOfferedFirst(t *testing.T) {
	a, h := newApp(t)
	withPhase(t, a)
	n, _ := a.Store.Create("note", map[string]any{"title": "Plan", "phase": "draft"})

	change(t, a, map[string]any{"type": "note", "field": "phase", "change": "delete"})
	ps := a.Chat.Proposals()
	if len(ps) != 1 {
		t.Fatalf("deleting is asked, not done: %d questions", len(ps))
	}
	card := get(t, h, "/").Body.String()
	hide, del := strings.Index(card, "Hide it instead"), strings.Index(card, "Delete it")
	if hide < 0 || del < 0 || hide > del || !strings.Contains(card, "can&#39;t be undone") {
		t.Errorf("the question offers hiding first, then deleting, and says deleting can't be undone")
	}
	if r := postForm(t, h, "/proposal/"+ps[0].ID+"/instead", url.Values{}); r.Code >= 400 {
		t.Fatalf("hiding instead: %d", r.Code)
	}
	note, _ := a.Types.Get("note")
	if f, ok := note.Field("phase"); !ok || !f.Hidden {
		t.Fatal("hiding instead hides the field and keeps it")
	}
	if got, _ := a.Store.Get("note", n.ID); got.Fields["phase"] != "draft" {
		t.Error("hiding instead keeps what it held")
	}

	change(t, a, map[string]any{"type": "note", "field": "phase", "change": "delete"})
	id := a.Chat.Proposals()[0].ID
	if r := postForm(t, h, "/proposal/"+id+"/accept", url.Values{}); r.Code >= 400 {
		t.Fatalf("deleting: %d", r.Code)
	}
	if _, ok := note.Field("phase"); ok {
		t.Error("deleted, the field is gone")
	}
	if got, _ := a.Store.Get("note", n.ID); got.Fields["phase"] != nil {
		t.Errorf("deleted, what it held is cleared: %v", got.Fields["phase"])
	}

	raw, _ := json.Marshal(map[string]any{"type": "note", "field": "title", "change": "delete", "agreed": true})
	if _, isErr := a.Chat.Call("change_field", raw); !isErr {
		t.Error("a type's title is never deleted")
	}
	raw, _ = json.Marshal(map[string]any{"type": "person", "change": "delete", "agreed": true})
	if _, isErr := a.Chat.Call("change_field", raw); !isErr {
		t.Error("a type Sameway relies on is never deleted")
	}
}

// Choices added on two computers both stay; a label, hiding and a
// deletion made on one reach the other.
func TestSchemaChangesTravelBetweenHosts(t *testing.T) {
	a, ha := newApp(t)
	b, hb := newApp(t)
	withPhase(t, a)
	keepInStep(t, a, ha, b)
	a.AddChoice("note", "phase", "blocked", "")
	b.AddChoice("note", "phase", "archived", "Put away")
	a.Relabel("note", "phase", "", "Stage")
	a.AddField("note", schema.Field{Name: "mood", Type: "string"})
	keepInStep(t, a, ha, b)
	keepInStep(t, b, hb, a)
	a.SetHidden("note", "mood", true)
	a.RemoveField("note", "phase")
	keepInStep(t, a, ha, b)

	note, _ := b.Types.Get("note")
	if _, ok := note.Field("phase"); ok {
		t.Error("a field deleted on one computer is deleted on the other")
	}
	if f, ok := note.Field("mood"); !ok || !f.Hidden {
		t.Error("a field hidden on one computer is hidden on the other")
	}

	// Choices added on both computers are on both.
	c, hc := newApp(t)
	d, hd := newApp(t)
	withPhase(t, c)
	keepInStep(t, c, hc, d)
	c.AddChoice("note", "phase", "blocked", "")
	d.AddChoice("note", "phase", "archived", "Put away")
	keepInStep(t, c, hc, d)
	keepInStep(t, d, hd, c)
	for name, x := range map[string]*app.App{"c": c, "d": d} {
		n, _ := x.Types.Get("note")
		f, _ := n.Field("phase")
		if f == nil || !contains(f.Values, "blocked") || !contains(f.Values, "archived") || f.Labels["archived"] != "Put away" {
			t.Errorf("%s should offer both new choices: %+v", name, f)
		}
	}
}

func contains(list []string, s string) bool { return slices.Contains(list, s) }
