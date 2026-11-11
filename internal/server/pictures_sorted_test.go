package server_test

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// readsLetter tags a record to do only when it is shown the picture of
// the letter, as a model that can see reads it; blind refuses pictures.
type readsLetter struct {
	mu    sync.Mutex
	blind bool
	saw   int
}

func (m *readsLetter) Name() string { return "letter" }
func (m *readsLetter) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(req.Messages[0].Images) > 0 {
		if m.blind {
			return nil, errors.New("400: this model does not support image input")
		}
		m.saw++
		return &llm.Response{Text: `{"tags": [{"tag": "to do", "why": "a fine to pay by Friday"}]}`}, nil
	}
	return &llm.Response{Text: `{"tags": []}`}, nil
}

// A photo of a letter shared from a phone is a note whose words are only
// the file's name; sorting it is shown the photo, so a fine to pay is
// tagged to do. A model that cannot see is asked again without it.
func TestAPhotoSharedIsReadWhenItIsSorted(t *testing.T) {
	t.Parallel()
	for _, blind := range []bool{false, true} {
		a, h := newApp(t)
		model := &readsLetter{blind: blind}
		a.Chat.Provider = model
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("files", "IMG_2041.png")
		fw.Write(pngOf(t, 40, 30))
		mw.Close()
		if rec := do(t, h, http.MethodPost, "/share", &buf, mw.FormDataContentType()); rec.Code != http.StatusSeeOther {
			t.Fatalf("shared: %d", rec.Code)
		}
		note := lastNote(t, a)
		a.Store.Create(records.TagType, map[string]any{"name": "to do", "means": "Something I have to do: pay, reply, book."})
		act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Sort", "kind": "classify", "tags": []any{"to do"}})
		a.Automate.Classify(context.Background(), act, "note", note.ID)
		got, _ := a.Store.Get("note", note.ID)
		tags := strings.Join(records.StringList(got.Fields["tags"]), ",")
		if !blind && (model.saw != 1 || !strings.Contains(tags, "to do")) {
			t.Errorf("the photo is read when sorted: saw %d, tags %s", model.saw, tags)
		}
		if blind && strings.Contains(tags, "to do") {
			t.Errorf("a model that cannot see is asked from the words: %s", tags)
		}
	}
}

// asksWords keeps each question it is put.
type asksWords struct {
	mu    sync.Mutex
	asked []string
}

func (m *asksWords) Name() string { return "words" }
func (m *asksWords) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.asked = append(m.asked, req.Messages[0].Content)
	return &llm.Response{Text: `{"tags": []}`}, nil
}

// A bill shared as a file, with no email connected, sets sorting up, and
// sorting reads the words of the file the note links to.
func TestAFileSharedIsReadForWhatItAsks(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	model := &asksWords{}
	a.Chat.Provider = model
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("files", "bill.txt")
	fw.Write([]byte("Water bill: 48.20 to pay by 3 December."))
	mw.Close()
	do(t, h, http.MethodPost, "/share", &buf, mw.FormDataContentType())
	note := lastNote(t, a)
	acts, _ := a.Store.List(records.ActionType, store.ListOptions{})
	var sort *store.Record
	for _, act := range acts {
		if act.Fields["kind"] == "classify" && act.Fields["what"] == "note" {
			sort = act
		}
	}
	if sort == nil {
		t.Fatalf("sharing sets up sorting what is shared: %d actions", len(acts))
	}
	a.Automate.Classify(context.Background(), sort, "note", note.ID)
	model.mu.Lock()
	defer model.mu.Unlock()
	if !strings.Contains(strings.Join(model.asked, "|"), "48.20 to pay by 3 December") {
		t.Errorf("the file's words are read when sorted: %q", model.asked)
	}
}
