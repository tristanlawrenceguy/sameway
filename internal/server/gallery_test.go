package server_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// seesReceipt reads a photo of a receipt, and tags what says Wickes as
// receipts.
type seesReceipt struct {
	mu  sync.Mutex
	saw int
}

func (m *seesReceipt) Name() string { return "receipt" }
func (m *seesReceipt) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(req.Messages[0].Images) > 0 {
		m.saw++
	}
	switch {
	case strings.Contains(req.System, "You read a picture"):
		return &llm.Response{Text: `{"name": "Receipt from Wickes", "description": "A till receipt for paint.", "text": "Wickes. Paint 24.00"}`}, nil
	case strings.Contains(req.Messages[0].Content, "Paint 24.00"):
		return &llm.Response{Text: `{"tags": [{"tag": "receipts", "why": "a till receipt"}]}`}, nil
	}
	return &llm.Response{Text: `{"tags": []}`}, nil
}

// A photo kept as a file is sorted like anything else: read once, named
// for what it is, and tagged; a gallery of the files tagged receipts
// shows it by its picture, with what it shows for someone who cannot see
// it, and its name under it.
func TestAPhotoIsSortedAndShownInAGallery(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	model := &seesReceipt{}
	a.Chat.Provider = model
	body, ct := multipartFile(t, "IMG_5120.png", string(pngOf(t, 40, 30)), nil)
	id := strings.TrimPrefix(do(t, h, "POST", "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
	a.Store.Create(records.TagType, map[string]any{"name": "receipts", "means": "a receipt for something bought"})
	act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Sort photos", "kind": "classify", "tags": []any{"receipts"}})
	a.Automate.Classify(context.Background(), act, records.FileType, id)
	f, _ := a.Store.Get(records.FileType, id)
	if model.saw != 1 || f.Fields["title"] != "Receipt from Wickes" || !strings.Contains(strings.Join(records.StringList(f.Fields["tags"]), ","), "receipts") {
		t.Fatalf("read once, named, tagged: saw %d, %v", model.saw, f.Fields)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "file", "as": "gallery", "where": []any{"tags=receipts"}, "label": "Receipts"},
	}), http.StatusCreated)
	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, `src="/files/`+id+`/small" alt="A till receipt for paint."`) || !strings.Contains(page, `<span class="sw-collection__caption">Receipt from Wickes</span>`) {
		t.Errorf("the gallery shows the receipt by its picture, said in words, named under it: %.3000s", page)
	}
}
