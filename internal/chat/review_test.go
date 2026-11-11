package chat_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A model server's turn that changed something is shown once what it
// changed, in the person's time, and answers again; the second answer is
// the reply. A turn that changed nothing is not.
func TestATurnThatChangedIsCheckedOnce(t *testing.T) {
	t.Parallel()
	var last []string
	answers := []string{
		`{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"create_record","arguments":"{\"type\":\"task\",\"fields\":{\"title\":\"Pay rent\"}}"}}]}`,
		`{"role":"assistant","content":"Done: rent and bank."}`,
		`{"role":"assistant","content":"I made Pay rent; Call the bank is not made yet."}`,
		`{"role":"assistant","content":"There is nothing to do."}`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Messages []struct {
				Content any `json:"content"`
			} `json:"messages"`
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &b)
		said, _ := b.Messages[len(b.Messages)-1].Content.(string)
		last = append(last, said)
		io.WriteString(w, `{"choices":[{"message":`+answers[0]+`,"finish_reason":"stop"}]}`)
		answers = answers[1:]
	}))
	defer server.Close()
	svc := newFullService(t)
	svc.Provider = &llm.OpenAI{BaseURL: server.URL + "/v1", Model: "small"}

	rec, err := svc.Send(context.Background(), "Add tasks to pay rent and call the bank")
	if err != nil {
		t.Fatal(err)
	}
	if len(last) != 3 || !strings.Contains(last[2], "This turn changed") || !strings.Contains(last[2], `made task "Pay rent": status: todo; title: Pay rent`) {
		t.Fatalf("the model is shown what the turn made, once: %q", last)
	}
	if got := rec.Fields["content"]; got != "I made Pay rent; Call the bank is not made yet." {
		t.Errorf("the answer after the check is the reply: %v", got)
	}
	if rec, _ = svc.Send(context.Background(), "Anything else?"); rec.Fields["content"] != "There is nothing to do." || len(last) != 4 {
		t.Errorf("a turn that changed nothing is not checked: %v, %d asks", rec.Fields["content"], len(last))
	}
}

// A model server that fails while a reply is being checked leaves that
// reply standing: the work is done, and an error with Send again would
// have the person do it twice.
func TestAReplyStandsWhenItsCheckFails(t *testing.T) {
	t.Parallel()
	answers := []string{
		`{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"create_record","arguments":"{\"type\":\"task\",\"fields\":{\"title\":\"Pay rent\"}}"}}]}`,
		`{"role":"assistant","content":"Made Pay rent."}`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(answers) == 0 {
			http.Error(w, `{"error":{"message":"bad request"}}`, http.StatusBadRequest)
			return
		}
		io.WriteString(w, `{"choices":[{"message":`+answers[0]+`,"finish_reason":"stop"}]}`)
		answers = answers[1:]
	}))
	defer server.Close()
	svc := newFullService(t)
	svc.Provider = &llm.OpenAI{BaseURL: server.URL + "/v1", Model: "small"}

	rec, err := svc.Send(context.Background(), "Add a task to pay rent")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Fields["role"] != "assistant" || rec.Fields["content"] != "Made Pay rent." {
		t.Errorf("the reply stands: %v %v", rec.Fields["role"], rec.Fields["content"])
	}
}
