package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// People who cannot check a result blame themselves when it fails, so the
// model says whose failure it was, looks again when asked if it is sure,
// and never fills in what a picture's reading left unclear.
func TestPromptSaysWhoseFailureAndWhatIsUnclear(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	sys := m.seen[0].System
	for _, want := range []string{"never that the person did it wrong", "look again in the records", "[unclear], say which"} {
		if !strings.Contains(sys, want) {
			t.Errorf("the prompt should say %q", want)
		}
	}
}
