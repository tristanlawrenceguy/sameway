package media_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A person adds a file: it becomes a record with its contents as Markdown,
// the original stays reachable, and the files page offers the upload.
func TestAFileBecomesARecordWithItsText(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

// An upload sent without a file returns the person to where they chose
// it, as a page, with an alert saying what to do and the upload there to
// try again; nothing is filed.
func TestUploadWithWrongContentTypeReturnsHTMLPage(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	rec := postForm(t, h, "/t/file/upload", url.Values{"from": {"/t/file"}})
	page, at := landed(t, h, rec)
	body := page.Body.String()
	if at != "/t/file" || !strings.HasPrefix(page.Header().Get("Content-Type"), "text/html") {
		t.Errorf("the person is back on the files page, got %q %q", at, page.Header().Get("Content-Type"))
	}
	if !strings.Contains(body, "Choose a file first.") {
		t.Errorf("the page says to choose a file; first 500 chars: %s", truncate(body))
	}
}

func TestUploadWithNoFileShowsAccessibleError(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	body, ct := multipartFile(t, "", "", url.Values{"from": {"/t/file"}})
	rec := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
	bodyStr := after(t, h, rec).Body.String()
	if !strings.Contains(bodyStr, `role="alert"`) || !strings.Contains(bodyStr, `data-kind="danger"`) {
		t.Error("the refusal is a danger alert, announced as it appears")
	}
	if !strings.Contains(bodyStr, "Not added") || !strings.Contains(bodyStr, "Choose a file first.") {
		t.Errorf("the alert says what to do; first 500 chars: %s", truncate(bodyStr))
	}
	if !strings.Contains(bodyStr, `data-component="upload"`) {
		t.Error("the upload is there to try again")
	}
	count, _ := a.Store.Count("file")
	if count > 0 {
		t.Errorf("no file record should exist after an empty upload; got %d records", count)
	}
}
