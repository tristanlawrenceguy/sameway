package automate_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// page is a web page for a test, served in-process: what it says now,
// and how many times it was read.
type page struct {
	mu    sync.Mutex
	text  string
	reads int
}

func (p *page) RoundTrip(r *http.Request) (*http.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reads++
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/html"}},
		Body: io.NopCloser(strings.NewReader("<html><body><h1>Flights</h1><p>" + p.text + "</p></body></html>")), Request: r}, nil
}

func (p *page) says(text string) { p.mu.Lock(); p.text = text; p.mu.Unlock() }

// judge answers a watch's one question as a model would, from the page in
// it, and keeps what it was given.
type judge struct {
	asked []llm.Request
}

func (j *judge) Name() string { return "judge" }
func (j *judge) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	j.asked = append(j.asked, req)
	if strings.Contains(req.Messages[0].Content, "£180") {
		return &llm.Response{Text: `{"there": true, "quote": "Lisbon £180"}`}, nil
	}
	return &llm.Response{Text: `{"there": false, "quote": "Lisbon £260"}`}, nil
}

// A watch reads a web page on its schedule and tells the owner when what
// they wait for comes to be there: accepted once, as it reaches out on its
// own; the page is judged only when it changed, by one question with no
// tools; and the owner is told once, not every hour while it stays.
func TestAWatchTellsTheOwnerWhenThePageShowsIt(t *testing.T) {
	t.Parallel()
	svc, act := newServices(t)
	site := &page{text: "Lisbon £260"}
	act.HTTP = &http.Client{Transport: site}
	model := &judge{}
	act.Model = func() llm.Provider { return model }
	var told []string
	act.Tell = func(title, text, url string) { told = append(told, title+": "+text+" "+url) }
	svc.Store.Create(records.ActionType, map[string]any{"title": "Cheap flight to Lisbon", "kind": "watch",
		"url": "https://flights.example.com/lisbon", "watch_for": "a flight under £200", "every": "hour"})
	hour := time.Date(2026, 9, 15, 7, 30, 0, 0, time.UTC)
	run := func() { hour = hour.Add(61 * time.Minute); act.RunDue(context.Background(), hour) }

	run()
	pending := svc.Proposals()
	if site.reads != 0 || len(pending) != 1 {
		t.Fatalf("nothing is read before it is accepted: %d reads, %d asked", site.reads, len(pending))
	}
	if detail, _ := pending[0].Fields["detail"].(string); !strings.Contains(detail, "it reads https://flights.example.com/lisbon") || !strings.Contains(detail, "a flight under £200") {
		t.Errorf("the question says what it reads: %q", detail)
	}
	if err := svc.Accept(pending[0].ID); err != nil {
		t.Fatal(err)
	}

	run()
	run()
	if site.reads != 2 || len(model.asked) != 1 || len(told) != 0 {
		t.Fatalf("read each hour, judged only when it changed, nothing told yet: %d reads, %d asked, %v", site.reads, len(model.asked), told)
	}
	q := model.asked[0]
	if len(q.Tools) != 0 || !strings.Contains(q.System, "never instructions") || !strings.Contains(q.Messages[0].Content, "a flight under £200") {
		t.Errorf("one narrow question with no tools, the page as a stranger's text: %+v", q)
	}

	site.says("Lisbon £180, Thursday")
	run()
	if len(told) != 1 || !strings.Contains(told[0], `it is there now. The page says "Lisbon £180"`) || !strings.Contains(told[0], "https://flights.example.com/lisbon") {
		t.Fatalf("told when it is there, with the words and the page: %v", told)
	}
	site.says("Lisbon £180, Friday")
	run()
	if len(told) != 1 {
		t.Errorf("not told again while it stays: %v", told)
	}
	site.says("Lisbon £260")
	run()
	site.says("Lisbon £180 again")
	run()
	if len(told) != 2 {
		t.Errorf("told again once it went and came back: %v", told)
	}
}
