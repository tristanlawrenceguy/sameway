package chat_test

import (
	"context"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What a person says from their phone is logged with the phone's name, so
// the activity log says where it came from; said on this machine, it says
// nothing extra.
func TestSaidFromAnotherDeviceSaysWhich(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = &scripted{steps: []*llm.Response{}}
	ctx := chat.WithVia(context.Background(), "pixel-7")
	if _, err := svc.Send(ctx, "hello from the train"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Send(context.Background(), "hello from the desk"); err != nil {
		t.Fatal(err)
	}
	log, err := svc.Store.List(chat.ActivityType, store.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	summaries := map[string]string{}
	for _, e := range log {
		if e.Fields["action"] == "said" {
			d, _ := e.Fields["detail"].(string)
			summaries[d], _ = e.Fields["summary"].(string)
			if d == "hello from the train" && e.Fields["via"] != "pixel-7" {
				t.Errorf("via should be pixel-7, got %v", e.Fields["via"])
			}
		}
	}
	if got := summaries["hello from the train"]; got != "You said hello from the train, on pixel-7" {
		t.Errorf("said from the phone reads %q", got)
	}
	if got := summaries["hello from the desk"]; got != "You said hello from the desk" {
		t.Errorf("said on this machine reads %q", got)
	}
}
