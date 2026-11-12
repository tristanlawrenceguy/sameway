package server_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// told is a model that hands over each request it is sent.
type told chan llm.Request

func (m told) Name() string { return "told" }
func (m told) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m <- req
	return &llm.Response{Text: "Added oat milk to your shopping list."}, nil
}

// A photo taken on the phone is kept as a file and goes to the
// assistant, asked to put what it shows on the shopping list, and the
// person is taken to the chat to watch; with no photo nothing is asked.
func TestAPhotoGoesToTheShoppingList(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	model := make(told, 4)
	a.Chat.Provider, a.Chat.ProviderErr = model, nil

	if page := get(t, h, "/scan").Body.String(); !strings.Contains(page, `capture="environment"`) || !strings.Contains(page, ">Add to the shopping list<") {
		t.Fatalf("the page opens the camera, with one button: %s", truncate(page))
	}
	files, _ := a.Store.List(records.FileType, store.ListOptions{})
	body, ct := multipartFile(t, "label.png", string(pngOf(t, 40, 30)), nil)
	res := do(t, h, http.MethodPost, "/scan", body, ct)
	if res.Code != http.StatusSeeOther || !strings.HasPrefix(res.Header().Get("Location"), "/chat") {
		t.Fatalf("the person goes to the chat: %d %s", res.Code, res.Header().Get("Location"))
	}
	var req llm.Request
	select {
	case req = <-model:
	case <-time.After(3 * time.Second):
		t.Fatal("the assistant was not asked")
	}
	last := req.Messages[len(req.Messages)-1]
	if !strings.Contains(last.Content, "shopping list") || len(last.Images) != 1 {
		t.Errorf("the assistant is asked for the shopping list, with the photo: %q, %d pictures", last.Content, len(last.Images))
	}
	waitFor(t, func() bool {
		msgs, _ := a.Store.List(records.MessageType, store.ListOptions{})
		return len(msgs) >= 2
	})
	if now, _ := a.Store.List(records.FileType, store.ListOptions{}); len(now) != len(files)+1 {
		t.Errorf("the photo is kept as a file: %d, was %d", len(now), len(files))
	}
	for len(model) > 0 {
		<-model
	}

	body, ct = multipartFile(t, "", "", nil)
	if res := do(t, h, http.MethodPost, "/scan", body, ct); res.Code != http.StatusSeeOther || len(model) != 0 {
		t.Errorf("with no photo, nothing is asked: %d", res.Code)
	}
}
