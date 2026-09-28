package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

func TestUpdateSamewayAsksTheUpdaterAndSaysWhatItSaid(t *testing.T) {
	for _, c := range []struct {
		what    string
		install bool
		out     update.Outcome
		told    string
	}{
		// A question is a question: nothing is installed, and the answer
		// says how to get the new version if they want it.
		{"a question", false, update.Outcome{Current: "0.3.0", Latest: "0.4.0", Newer: true, Says: "sameway 0.4.0 is out; this is 0.3.0"}, "sameway update"},
		{"nothing newer", false, update.Outcome{Current: "0.4.0", Latest: "0.4.0", Says: "sameway 0.4.0 is the latest"}, "is the latest"},
		{"an install", true, update.Outcome{Current: "0.3.0", Latest: "0.4.0", Newer: true, Installed: true, Says: update.Installed("0.4.0")}, "next start"},
	} {
		svc, m := withModel(t, call("update_sameway", map[string]any{"install": c.install}))
		asked := 0
		var with bool
		svc.Update = func(_ context.Context, install bool) (update.Outcome, error) {
			asked, with = asked+1, install
			return c.out, nil
		}
		if _, err := svc.Send(context.Background(), "update"); err != nil {
			t.Fatalf("%s: %v", c.what, err)
		}
		if asked != 1 || with != c.install {
			t.Errorf("%s: the updater was asked %d times with install=%v, want once with %v", c.what, asked, with, c.install)
		}
		said := lastToolResult(m.seen[len(m.seen)-1])
		if said.IsError || !strings.Contains(said.Content, c.told) {
			t.Errorf("%s: the model was told %q, which does not contain %q", c.what, said.Content, c.told)
		}
	}
}

func TestAnInstallGoesOnTheReceiptAndInTheLog(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = &scripted{steps: []*llm.Response{call("update_sameway", map[string]any{"install": true})}}
	svc.Update = func(_ context.Context, install bool) (update.Outcome, error) {
		return update.Outcome{Current: "0.3.0", Latest: "0.4.0", Newer: true, Installed: true, Says: update.Installed("0.4.0")}, nil
	}
	reply, err := svc.Send(context.Background(), "update sameway")
	if err != nil {
		t.Fatal(err)
	}
	changes, _ := reply.Fields["changes"].([]any)
	if len(changes) != 1 {
		t.Fatalf("the install should be one change on the receipt, got %v", reply.Fields["changes"])
	}
	if got := changes[0].(map[string]any); got["action"] != "updated to" || got["component"] != "sameway 0.4.0" {
		t.Errorf("receipt entry wrong: %v", got)
	}
	log, _ := svc.Store.List(chat.ActivityType, store.ListOptions{})
	var found string
	for _, entry := range log {
		if summary, _ := entry.Fields["summary"].(string); strings.Contains(summary, "sameway 0.4.0") {
			found = summary
		}
	}
	if found != "Assistant updated to sameway 0.4.0" {
		t.Errorf("the log should read as a sentence, got %q from %d entries", found, len(log))
	}
}

func TestWithoutAnUpdaterTheToolSaysSo(t *testing.T) {
	svc, m := withModel(t, call("update_sameway", map[string]any{"install": true}))
	svc.Update = nil
	if _, err := svc.Send(context.Background(), "update"); err != nil {
		t.Fatal(err)
	}
	said := lastToolResult(m.seen[len(m.seen)-1])
	if !said.IsError || !strings.Contains(said.Content, "cannot update itself") {
		t.Errorf("the model was told %q", said.Content)
	}
}
