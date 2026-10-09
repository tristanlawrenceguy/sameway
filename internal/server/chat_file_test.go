package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A file sent with a message is filed as a record, its words go to the
// model with the message, and the transcript shows which file went along.
func TestChatTakesAFileWithTheMessage(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	model := &scripted{steps: []*llm.Response{{Text: "It is about the pond. I filed it at /t/file."}}}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil
	body, ct := multipartFile(t, "notes.txt", "The pond needs a liner.", url.Values{"from": {"/chat"}, "message": {"What does this say?"}})
	rec := do(t, h, http.MethodPost, "/chat", body, ct)
	wantStatus(t, rec, http.StatusSeeOther)
	if len(model.seen) != 1 {
		t.Fatalf("the model should be asked once, got %d", len(model.seen))
	}
	last := model.seen[0].Messages[len(model.seen[0].Messages)-1].Content
	if !strings.Contains(last, "What does this say?") || !strings.Contains(last, "The pond needs a liner.") || !strings.Contains(last, "filed at /t/file/") {
		t.Errorf("the model gets the words, the file's text and where it is filed: %q", last)
	}
	msgs, _ := a.Store.List(records.MessageType, store.ListOptions{OrderBy: "created_at"})
	if len(msgs) != 2 || msgs[0].Fields["file"] == "" {
		t.Fatalf("the user message carries the file id: %v", msgs)
	}
	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, "Attached:") || !strings.Contains(page, `href="/t/file/`+msgs[0].Fields["file"].(string)+`"`) {
		t.Errorf("the transcript shows the attached file as a link to its page: %.800s", page)
	}
	if !strings.Contains(page, `enctype="multipart/form-data"`) || !strings.Contains(page, `type="file"`) {
		t.Error("the composer takes a file")
	}

	// A message with the field left empty is just a message.
	body, ct = multipartFile(t, "", "", url.Values{"from": {"/chat"}, "message": {"Only words."}})
	wantStatus(t, do(t, h, http.MethodPost, "/chat", body, ct), http.StatusSeeOther)
	msgs, _ = a.Store.List(records.MessageType, store.ListOptions{OrderBy: "created_at"})
	if file, _ := msgs[2].Fields["file"].(string); len(msgs) != 4 || file != "" {
		t.Errorf("no file, no attachment: %v", msgs[2].Fields)
	}
}
