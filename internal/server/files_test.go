package server_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// multipartFile builds a form with one file part and any other fields.
func multipartFile(t *testing.T, name, content string, fields url.Values) (io.Reader, string) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, vs := range fields {
		for _, v := range vs {
			w.WriteField(k, v)
		}
	}
	if name != "" {
		part, err := w.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(part, content)
	}
	w.Close()
	return &body, w.FormDataContentType()
}

// A person adds a file: it becomes a record with its contents as Markdown,
// the original stays reachable, and the files page offers the upload.
func TestAFileBecomesARecordWithItsText(t *testing.T) {
	a, h := newApp(t)
	body, ct := multipartFile(t, "Plan.md", "# The plan\n\nDig the pond.", nil)
	rec := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
	wantStatus(t, rec, http.StatusSeeOther)
	loc := rec.Header().Get("Location")
	id := strings.TrimPrefix(loc, "/t/file/")
	if id == "" || id == loc {
		t.Fatalf("upload should land on the file's page, got %q", loc)
	}
	file, err := a.Store.Get("file", id)
	if err != nil {
		t.Fatal(err)
	}
	if file.Fields["title"] != "Plan" || file.Fields["kind"] != "markdown" || file.Fields["status"] != "ready" || !strings.Contains(file.Fields["text"].(string), "Dig the pond") {
		t.Errorf("the record should carry the file's words: %v", file.Fields)
	}
	page := get(t, h, loc).Body.String()
	if !strings.Contains(page, "Dig the pond") || !strings.Contains(page, "Open the original") {
		t.Errorf("the file's page shows its text and leads to the original: %.500s", page)
	}
	orig := get(t, h, "/files/"+id)
	wantStatus(t, orig, http.StatusOK)
	if orig.Body.String() != "# The plan\n\nDig the pond." || !strings.HasPrefix(orig.Header().Get("Content-Type"), "text/") {
		t.Errorf("the original comes back as it was, as the type it is: %q %q", orig.Body.String(), orig.Header().Get("Content-Type"))
	}
	if list := get(t, h, "/t/file").Body.String(); !strings.Contains(list, `data-component="upload"`) {
		t.Error("the files page should offer the upload")
	}
	if !strings.Contains(get(t, h, "/api/activity").Body.String(), `"added"`) {
		t.Error("adding a file is a change in the activity")
	}
}

// A workspace that names a converter hands the file to it and the record
// says it is converting until the answer comes back.
func TestAConverterFillsInTheTextLater(t *testing.T) {
	a, h := newApp(t)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, err := r.FormFile("files"); err != nil {
			http.Error(w, "no file", http.StatusBadRequest)
			return
		}
		io.WriteString(w, `{"document": {"md_content": "# Converted\n\nBy the service."}}`)
	}))
	defer remote.Close()
	a.Workspace.Config.Files.Convert = map[string]string{"pdf": remote.URL}
	body, ct := multipartFile(t, "scan.pdf", "%PDF-1.4 not really", nil)
	rec := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
	wantStatus(t, rec, http.StatusSeeOther)
	id := strings.TrimPrefix(rec.Header().Get("Location"), "/t/file/")
	var file *store.Record
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		file, _ = a.Store.Get("file", id)
		if file != nil && file.Fields["status"] == "ready" {
			break
		}
	}
	if file == nil || file.Fields["status"] != "ready" || !strings.Contains(file.Fields["text"].(string), "By the service") {
		t.Fatalf("the converter's Markdown should fill the record: %v", file)
	}
	if note, _ := file.Fields["note"].(string); !strings.Contains(note, "converted by") {
		t.Errorf("the record says who read it: %q", note)
	}
}

// A file sent with a message is filed as a record, its words go to the
// model with the message, and the transcript shows which file went along.
func TestChatTakesAFileWithTheMessage(t *testing.T) {
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
	msgs, _ := a.Store.List(chat.MessageType, store.ListOptions{OrderBy: "created_at"})
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
	msgs, _ = a.Store.List(chat.MessageType, store.ListOptions{OrderBy: "created_at"})
	if file, _ := msgs[2].Fields["file"].(string); len(msgs) != 4 || file != "" {
		t.Errorf("no file, no attachment: %v", msgs[2].Fields)
	}
}

// POST to /t/file/upload with a non-multipart Content-Type returns an HTML
// page (not plain text) with status 400 and an accessible error message.
func TestUploadWithWrongContentTypeReturnsHTMLPage(t *testing.T) {
	_, h := newApp(t)
	rec := postForm(t, h, "/t/file/upload", url.Values{"from": {"/t/file"}})
	wantStatus(t, rec, http.StatusBadRequest)
	body := rec.Body.String()
	if !strings.Contains(strings.ToLower(body), "<!doctype html>") {
		t.Errorf("upload error should be an HTML page, not plain text; first 200 chars: %s", truncate(body))
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "text/html") {
		t.Errorf("response Content-Type header should be text/html for upload errors, got %q", ct)
	}
	if !strings.Contains(body, "Please select a file") {
		t.Errorf("the HTML page should carry an error message about selecting a file; first 500 chars: %s", truncate(body))
	}
}

// POST to /t/file/upload with multipart/form-data but no file part returns
// an HTML page with status 400, an aria-live region and the alert component.
func TestUploadWithNoFileShowsAccessibleError(t *testing.T) {
	a, h := newApp(t)
	body, ct := multipartFile(t, "", "", url.Values{"from": {"/t/file"}})
	rec := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
	wantStatus(t, rec, http.StatusBadRequest)
	bodyStr := rec.Body.String()
	if !strings.Contains(strings.ToLower(bodyStr), "<!doctype html>") {
		t.Errorf("upload without file should return an HTML page; first 200 chars: %s", truncate(bodyStr))
	}
	if !strings.Contains(bodyStr, `aria-live="assertive"`) {
		t.Error(`the error page must include aria-live="assertive" for screen-reader announcements`)
	}
	if !strings.Contains(bodyStr, "data-component=\"alert\"") || !strings.Contains(bodyStr, "kind=\"danger\"") {
		t.Error("the error page should render a danger alert component with the message")
	}
	if !strings.Contains(bodyStr, "Please select a file") {
		t.Errorf("the error message should say to select a file; first 500 chars: %s", truncate(bodyStr))
	}
	if !strings.Contains(bodyStr, `data-component="upload"`) {
		t.Error("the error page should re-render the upload form for a quick retry")
	}
	count, _ := a.Store.Count("file")
	if count > 0 {
		t.Errorf("no file record should exist after an empty upload; got %d records", count)
	}
}
