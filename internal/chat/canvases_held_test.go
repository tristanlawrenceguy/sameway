package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The tab list the model reads says what each other tab holds, by the
// names its blocks go by, and says a tab with nothing on it is empty, so a
// model adds to the tab that has the thing and sees a tab left over.
func TestTheTabListSaysWhatEachTabHolds(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Call("create_canvas", []byte(`{"name":"Garden"}`))
	svc.Call("create_canvas", []byte(`{"name":"Spare"}`))
	garden := svc.Canvases()[1].ID
	svc.Call("add_component", []byte(`{"component":"heading","props":{"text":"Beds to dig"},"canvas":"`+garden+`"}`))
	m := &scripted{steps: []*llm.Response{{Text: "ok"}}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	sys := m.seen[0].System
	if !strings.Contains(sys, "/c/"+garden+"): Beds to dig") {
		t.Errorf("the Garden tab says it holds Beds to dig:\n%s", tail(sys))
	}
	if !strings.Contains(sys, "): empty") {
		t.Errorf("the Spare tab says it is empty:\n%s", tail(sys))
	}
}
