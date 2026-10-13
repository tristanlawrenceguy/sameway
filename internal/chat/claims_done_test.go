package chat_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A reply that says it made something in a turn that changed nothing is
// the model answering in words where a tool was needed (backlog 0616): it
// is told once and asked again, and the reply kept is the one that made it.
func TestASaidChangeThatDidNotHappenIsAskedAgain(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		{Text: "Created your task Call dentist, due Friday 4 Oct."},
		call("create_record", map[string]any{"type": "task", "fields": map[string]any{"title": "Call dentist"}}),
		{Text: "Created your task Call dentist."},
	}}
	svc.Provider = m
	reply, err := svc.Send(context.Background(), "create a task: Call dentist, due Friday")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.seen) != 3 {
		t.Fatalf("the model should be told once and then run its tool, got %d calls", len(m.seen))
	}
	if told := m.seen[1].Messages[len(m.seen[1].Messages)-1]; told.Role != llm.RoleUser || !strings.Contains(told.Content, "Nothing has changed") || !strings.Contains(told.Content, "not the person") {
		t.Errorf("the model is told nothing changed: %+v", told)
	}
	tasks, _ := svc.Store.List("task", store.ListOptions{})
	if len(tasks) != 1 || reply.Fields["content"] != "Created your task Call dentist." {
		t.Errorf("the task is made and its reply kept as said: %d tasks, %q", len(tasks), reply.Fields["content"])
	}
}

// Still saying so when asked again, the reply says under it that nothing
// changed, so the person is not left believing it was done.
func TestASaidChangeStillUnmadeSaysNothingChanged(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		{Text: "Added your habit Read 20 pages."},
		{Text: "I've added it."},
	}}
	svc.Provider = m
	reply, err := svc.Send(context.Background(), "add a habit: read 20 pages")
	if err != nil {
		t.Fatal(err)
	}
	if said, _ := reply.Fields["content"].(string); !strings.HasPrefix(said, "I've added it.") || !strings.Contains(said, "Nothing was changed") {
		t.Errorf("the reply says nothing changed under it: %q", said)
	}
}

// An answer that says nothing was done, and a change really made, are
// left alone.
func TestAnAnswerOrAMadeChangeIsNotQuestioned(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	// What happened on the crew's chat: a list that says who made each
	// action was taken for a claim, and the person was told their message
	// came through empty.
	m := &scripted{steps: []*llm.Response{{Text: "You have two tasks due today. Both were added last week.\nNew action, made by you.\nLogged twice this week."}}}
	svc.Provider = m
	reply, err := svc.Send(context.Background(), "what is due today?")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.seen) != 1 || strings.Contains(reply.Fields["content"].(string), "Nothing was changed") {
		t.Errorf("a plain answer stands: %d calls, %q", len(m.seen), reply.Fields["content"])
	}
	m = &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Plan"}}),
		{Text: "Created the note Plan."},
	}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "a note called Plan"); err != nil || len(m.seen) != 2 {
		t.Errorf("a change made is not questioned: %v, %d calls", err, len(m.seen))
	}
}

// A failure is kept in the conversation in a person's words, as the page
// says it and the API and a title read it (backlog 0638, 0666).
func TestAFailureIsKeptInPlainWords(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Provider = refusing{}
	rec, _ := svc.Send(context.Background(), "hello")
	if rec == nil || rec.Fields["content"] != "The assistant could not reach the model. Try again." {
		t.Fatalf("the failure is kept in plain words: %+v", rec)
	}
}

type refusing struct{}

func (refusing) Name() string { return "refusing" }
func (refusing) Complete(context.Context, llm.Request) (*llm.Response, error) {
	return nil, errors.New("claude: exit status 1: ")
}
