package chat_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func newest(t *testing.T, svc *chat.Service) *store.Record {
	t.Helper()
	log, _ := svc.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if len(log) == 0 {
		t.Fatal("nothing logged")
	}
	return log[0]
}

func undo(t *testing.T, svc *chat.Service, id string) {
	t.Helper()
	if err := svc.UndoAs("human", id); err != nil {
		t.Fatalf("undo: %v", err)
	}
}

// A setting changed goes back to what it was, even to nothing, and the
// undo can itself be undone.
func TestASettingChangeCanBeUndone(t *testing.T) {
	svc := newFullService(t)
	cfg := withSettings(svc, map[string]string{"ui.pace": "calm"})
	use(t, svc, "set_setting", map[string]any{"key": "ui.pace", "value": "quick"})
	set := newest(t, svc)
	undo(t, svc, set.ID)
	if cfg["ui.pace"] != "calm" {
		t.Fatalf("undo puts the pace back, got %q", cfg["ui.pace"])
	}
	undo(t, svc, newest(t, svc).ID)
	if cfg["ui.pace"] != "quick" {
		t.Errorf("undoing the undo puts it back again, got %q", cfg["ui.pace"])
	}
}

// Records made from a file go together, in one undo, and come back
// together when that is undone.
func TestAnImportIsUndoneInOneGo(t *testing.T) {
	svc := newFullService(t)
	var ids []string
	for _, name := range []string{"Sandra", "Lee", "Priya"} {
		rec, _ := svc.Store.Create("person", map[string]any{"name": name})
		ids = append(ids, rec.ID)
	}
	chat.Record(svc.Store, "human", chat.Change{Action: "imported", Component: "person", Detail: "3 people from contacts.csv", Before: chat.Imported("person", ids)})
	undo(t, svc, newest(t, svc).ID)
	if n, _ := svc.Store.Count("person"); n != 0 {
		t.Fatalf("one undo takes the whole import back, %d left", n)
	}
	undo(t, svc, newest(t, svc).ID)
	if n, _ := svc.Store.Count("person"); n != 3 {
		t.Errorf("undoing that brings all three back, got %d", n)
	}
}

// A turn that fails after changing things keeps its receipt: the change
// happened, and it can be undone from the reply.
func TestAFailedTurnKeepsWhatItDid(t *testing.T) {
	svc := newFullService(t)
	raw, _ := json.Marshal(map[string]any{"component": "heading", "props": map[string]any{"text": "Garden"}})
	svc.Provider = &failingAfter{first: raw}
	rec, err := svc.SendOn(t.Context(), "", "make a heading")
	if err == nil || rec == nil || rec.Fields["role"] != "error" {
		t.Fatalf("the turn fails, got %v %v", rec, err)
	}
	changes, _ := rec.Fields["changes"].([]any)
	if len(changes) != 1 || !strings.Contains(rec.Fields["content"].(string), "can be undone") {
		t.Errorf("the failure carries what the turn did, got %v %q", changes, rec.Fields["content"])
	}
}

// failingAfter adds a heading, then the model goes away.
type failingAfter struct {
	first json.RawMessage
	done  bool
}

func (f *failingAfter) Name() string { return "failing" }
func (f *failingAfter) Complete(_ context.Context, _ llm.Request) (*llm.Response, error) {
	if !f.done {
		f.done = true
		return &llm.Response{ToolCalls: []llm.ToolCall{{ID: "c", Name: "add_component", Args: f.first}}}, nil
	}
	return nil, errors.New("the AI model at http://127.0.0.1:1/v1 isn't answering")
}

// What the person says they need, and the workspace's language, are in
// every prompt; and the prompt asks for everyday words.
func TestTheAssistantHearsNeedsAndLanguage(t *testing.T) {
	svc := newFullService(t)
	svc.Needs, svc.Language = "I use a screen reader; keep things simple", "de"
	m := &scripted{}
	svc.Provider = m
	svc.SendOn(t.Context(), "", "hallo")
	if len(m.seen) == 0 {
		t.Fatal("the model was asked")
	}
	sys := m.seen[0].System
	for _, want := range []string{"I use a screen reader; keep things simple", "language is de", "short, everyday words"} {
		if !strings.Contains(sys, want) {
			t.Errorf("the prompt says %q", want)
		}
	}
}
