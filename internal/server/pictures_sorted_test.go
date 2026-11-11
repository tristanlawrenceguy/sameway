package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// readsLetter reads the picture of a letter when shown it, as a model
// that can see does, and tags to do a record whose words say a fine is
// to pay; blind refuses pictures.
type readsLetter struct {
	mu    sync.Mutex
	blind bool
	saw   int
	asked []string
}

func (m *readsLetter) Name() string { return "letter" }
func (m *readsLetter) Complete(_ context.Context, req llm.Request) (*llm.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := req.Messages[0].Content
	m.asked = append(m.asked, q)
	if len(req.Messages[0].Images) > 0 {
		if m.blind {
			return nil, errors.New("400: this model does not support image input")
		}
		m.saw++
	}
	switch {
	case strings.Contains(req.System, "You read a picture"):
		said, _ := json.Marshal("Penalty charge notice. Pay 35 pounds by Friday 20 November. Read: " + q)
		return &llm.Response{Text: `{"name": "Parking fine from the council", "description": "A printed letter from the council about a parking fine.", "text": ` + string(said) + `}`}, nil
	case strings.Contains(q, "Pay 35 pounds") || len(req.Messages[0].Images) > 0:
		return &llm.Response{Text: `{"tags": [{"tag": "to do", "why": "a fine to pay by Friday"}]}`}, nil
	}
	return &llm.Response{Text: `{"tags": []}`}, nil
}

// A photo of a letter shared from a phone is a note whose words are only
// the file's name. The first sorting reads the picture once: its words
// and what it shows are kept on the file, the note takes the name of
// what it is, and it is tagged from those words, so a fine to pay is to
// do; sorting it again does not send the picture again. A model that
// cannot see is asked from the words, and nothing is renamed.
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
		act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Sort", "kind": "classify", "tags": []any{"to do"}})
		a.Automate.Classify(context.Background(), act, "note", note.ID)
		got, _ := a.Store.Get("note", note.ID)
		tags := strings.Join(records.StringList(got.Fields["tags"]), ",")
		files, _ := a.Store.List(records.FileType, store.ListOptions{})
		if blind {
			if strings.Contains(tags, "to do") || got.Fields["title"] != "IMG_2041" {
				t.Errorf("a model that cannot see is asked from the words, and nothing renamed: %s %v", tags, got.Fields["title"])
			}
			continue
		}
		if model.saw != 1 || !strings.Contains(tags, "to do") || got.Fields["title"] != "Parking fine from the council" {
			t.Errorf("read once, tagged from its words, named for what it is: saw %d, tags %s, %v", model.saw, tags, got.Fields["title"])
		}
		if len(files) != 1 || !strings.Contains(fmt.Sprint(files[0].Fields["text"]), "Pay 35 pounds") || !strings.Contains(fmt.Sprint(files[0].Fields["description"]), "parking fine") {
			t.Errorf("its words and what it shows are kept on the file: %v", files[0].Fields)
		}
		a.Automate.Classify(context.Background(), act, "note", note.ID)
		if model.saw != 1 {
			t.Errorf("a picture read is not sent again: saw %d", model.saw)
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

// A letter photographed page by page and shared from a phone at once is
// one note: every photo is read, each as one of them, its words kept, so
// the whole letter is searched and sorted, and the note is named once.
func TestPhotosSharedTogetherAreReadAsOne(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	model := &readsLetter{}
	a.Chat.Provider = model
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for i := 1; i <= 4; i++ {
		fw, _ := mw.CreateFormFile("files", fmt.Sprintf("IMG_30%d.png", i))
		fw.Write(pngOf(t, 40+i, 30))
	}
	mw.Close()
	do(t, h, http.MethodPost, "/share", &buf, mw.FormDataContentType())
	note := lastNote(t, a)
	act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Sort", "kind": "classify", "tags": []any{"to do"}})
	a.Automate.Classify(context.Background(), act, "note", note.ID)
	files, _ := a.Store.List(records.FileType, store.ListOptions{})
	read := 0
	for _, f := range files {
		if strings.Contains(fmt.Sprint(f.Fields["text"]), " of 4 shared together") {
			read++
		}
	}
	got, _ := a.Store.Get("note", note.ID)
	if len(files) != 4 || read != 4 || model.saw != 4 || got.Fields["title"] != "Parking fine from the council" {
		t.Errorf("every photo read as one of four, the note named: %d files, %d read, saw %d, %v", len(files), read, model.saw, got.Fields["title"])
	}
}
