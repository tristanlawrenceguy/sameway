package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// What providers answer is sorted into what a person can do about it.
func TestAFailureIsSortedForAPerson(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		status int
		said   string
		kind   FailKind
		words  string
	}{
		{400, "Your credit balance is too low to access the Anthropic API.", FailCredit, "no credit left"},
		{402, "Insufficient credits", FailCredit, "no credit left"},
		{401, "invalid x-api-key", FailKey, "did not accept the key"},
		{429, "rate limit", FailRate, "Wait a minute"},
		{529, "Overloaded", FailBusy, "busy just now"},
		{400, "prompt is too long: 210000 tokens > 200000 maximum", FailLong, "Start a new chat"},
		{404, "model 'x' not found", FailModel, "is not there any more"},
		{400, "something odd", FailOther, "did not answer: something odd"},
	} {
		f := Classify(c.status, c.said, "Anthropic", "x")
		if f.Kind != c.kind || !strings.Contains(f.Error(), c.words) {
			t.Errorf("%d %q: kind %d, %q", c.status, c.said, f.Kind, f.Error())
		}
	}
	if m := apiMessage(`POST "https://api.anthropic.com/v1/messages": 400 Bad Request {"type":"error","error":{"type":"invalid_request_error","message":"Your credit balance is too low"}}`); m != "Your credit balance is too low" {
		t.Errorf("the message inside an API error: %q", m)
	}
}

// A provider busy for a moment is sent the same again, as a person would.
func TestABusyProviderIsAskedAgain(t *testing.T) {
	was := retryWaits
	retryWaits = []time.Duration{time.Millisecond, time.Millisecond}
	defer func() { retryWaits = was }()
	tries := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tries++
		if tries < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()
	resp, err := (&OpenAI{BaseURL: srv.URL, Model: "m"}).Complete(context.Background(), Request{})
	if err != nil || resp.Text != "hello" || tries != 3 {
		t.Errorf("asked again until it answered: %v %v %d", resp, err, tries)
	}
}
