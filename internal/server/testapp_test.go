package server_test

import (
	"context"
	"encoding/json"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
	"github.com/tristanlawrenceguy/sameway/internal/server/servertest"
)

// scripted is a model that answers with a fixed sequence of responses.
type scripted struct {
	steps []*llm.Response
	seen  []llm.Request
}

func (s *scripted) Name() string { return "scripted" }

func (s *scripted) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	s.seen = append(s.seen, req)
	if len(s.steps) == 0 {
		return &llm.Response{Text: "ok"}, nil
	}
	next := s.steps[0]
	s.steps = s.steps[1:]
	return next, nil
}

func toolCall(name string, args map[string]any) *llm.Response {
	raw, _ := json.Marshal(args)
	return &llm.Response{ToolCalls: []llm.ToolCall{{ID: "call", Name: name, Args: raw}}}
}

// assertAllComponentsKnown walks the parsed HTML and fails if any element uses a
// data-component value that is not registered in the app's component registry.
func assertAllComponentsKnown(t *testing.T, doc *htmltest.Doc, names []string) {
	t.Helper()
	known := make(map[string]bool, len(names))
	for _, n := range names {
		known[n] = true
	}
	doc.Walk(func(n *html.Node) {
		if c, ok := htmltest.Attr(n, "data-component"); ok && !known[c] {
			t.Errorf("%s: data-component=%q is not a registered component (known: %v)", n.Data, c, names)
		}
	})
}

// The helpers the server's tests share with its features' (servertest).
var (
	newApp        = servertest.New
	newAppWith    = servertest.NewWith
	do            = servertest.Do
	get           = servertest.Get
	postForm      = servertest.PostForm
	postJSON      = servertest.PostJSON
	parse         = servertest.Parse
	decode        = servertest.Decode
	wantStatus    = servertest.WantStatus
	truncate      = servertest.Truncate
	testMachine   = servertest.Machine
	as            = servertest.As
	landed        = servertest.Landed
	after         = servertest.After
	waitFor       = servertest.WaitFor
	multipartFile = servertest.MultipartFile
	said          = servertest.Said
)
