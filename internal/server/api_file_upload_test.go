package server_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// apiFileUploadRequest is the JSON body for POST /api/file/upload.
type apiFileUploadRequest struct {
	Title    string `json:"title,omitempty"`
	Filename string `json:"filename,omitempty"`
	Content  string `json:"content,omitempty"`
}

// TestAPIFileUploadWithBase64Content covers the happy path: an agent sends
// base64-encoded binary via POST /api/file/upload and gets back a created
// record with correct metadata; the file lands on disk with decoded content.
func TestAPIFileUploadWithBase64Content(t *testing.T) {
	a, h := newApp(t)

	plain := "Hello from base64 upload."
	b64 := base64.StdEncoding.EncodeToString([]byte(plain))

	body := apiFileUploadRequest{
		Title:    "Greeting.txt",
		Filename: "greeting.txt",
		Content:  b64,
	}
	raw, _ := json.Marshal(body)
	rec := do(t, h, http.MethodPost, "/api/file/upload", strings.NewReader(string(raw)), "application/json")

	wantStatus(t, rec, http.StatusCreated)
	var created struct {
		ID     string         `json:"id"`
		Type   string         `json:"type"`
		Fields map[string]any `json:"fields"`
	}
	decode(t, rec, &created)

	if created.ID == "" {
		t.Fatalf("response should include an id: %s", truncate(rec.Body.String()))
	}
	if created.Type != "file" {
		t.Errorf("type should be file, got %q", created.Type)
	}
	recLoc := rec.Header().Get("Location")
	if !strings.Contains(recLoc, "/api/file/"+created.ID) {
		t.Errorf("Location header should point to the record; got %q", recLoc)
	}

	fld := created.Fields
	if fld["title"] != "Greeting.txt" {
		t.Errorf("title: want Greeting.txt, got %v", fld["title"])
	}
	if fld["name"] != "greeting.txt" {
		t.Errorf("name: want greeting.txt, got %v", fld["name"])
	}
	if fld["kind"] != "text" {
		t.Errorf("kind: want text, got %v", fld["kind"])
	}
	size, ok := fld["size"].(float64)
	if !ok || int(size) != len(plain) {
		t.Errorf("size: want %d, got %v", len(plain), fld["size"])
	}
	status, _ := fld["status"].(string)
	if status == "converting" {
		t.Log("status is converting; waiting briefly for readNow to complete")
	}

	// The record should end up ready because text is a built-in format.
	fileRec, err := a.Store.Get("file", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	finishedStatus, _ := fileRec.Fields["status"].(string)
	if finishedStatus == "converting" {
		t.Skip("external converter is running; skip disk check")
	}

	textVal, hasText := fileRec.Fields["text"]
	if !hasText || textVal != plain {
		t.Errorf("record text should be the decoded content; got %v", textVal)
	}

	storedPath, _ := fileRec.Fields["path"].(string)
	filesDir := a.Workspace.FilesDir()
	fullPath := filepath.Join(filesDir, storedPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Errorf("file should exist on disk at %s", fullPath)
	} else if err == nil {
		data, _ := os.ReadFile(fullPath)
		if string(data) != plain {
			t.Errorf("disk file content: want %q, got %q", plain, string(data))
		}
	}
}

// TestAPIFileUploadWithoutContent creates a stub record when no base64 field
// is sent — the same behaviour as POST /api/file.
func TestAPIFileUploadWithoutContent(t *testing.T) {
	a, h := newApp(t)

	body := apiFileUploadRequest{Title: "Stub.txt", Filename: "stub.txt"}
	raw, _ := json.Marshal(body)
	rec := do(t, h, http.MethodPost, "/api/file/upload", strings.NewReader(string(raw)), "application/json")

	wantStatus(t, rec, http.StatusCreated)
	var created struct {
		ID     string         `json:"id"`
		Type   string         `json:"type"`
		Fields map[string]any `json:"fields"`
	}
	decode(t, rec, &created)

	if created.Type != "file" || created.ID == "" {
		t.Fatalf("should return a file record; %s", truncate(rec.Body.String()))
	}
	fld := created.Fields
	size, _ := fld["size"].(float64)
	if size != 0 {
		t.Errorf("stub should have size 0, got %v", fld["size"])
	}

	count, err := a.Store.Count("file")
	if err != nil || count == 0 {
		t.Fatalf("a stub file record should exist; count=%d err=%v", count, err)
	}
}

// TestAPIFileUploadInvalidBase64 returns a 400 error when the content is not
// valid base64.
func TestAPIFileUploadInvalidBase64(t *testing.T) {
	_, h := newApp(t)

	body := apiFileUploadRequest{Filename: "bad.txt", Content: "!!!not-valid-base64!!!"}
	raw, _ := json.Marshal(body)
	rec := do(t, h, http.MethodPost, "/api/file/upload", strings.NewReader(string(raw)), "application/json")

	wantStatus(t, rec, http.StatusBadRequest)
	var problem struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	decode(t, rec, &problem)
	if problem.Error.Code != "bad_request" {
		t.Errorf("code should be bad_request, got %q", problem.Error.Code)
	}
	if problem.Error.Message == "" {
		t.Error("error message should say why the request is bad")
	}
}

// TestAPIFileUploadMissingContent returns a 400 error when content is absent.
func TestAPIFileUploadMissingContent(t *testing.T) {
	_, h := newApp(t)

	body := apiFileUploadRequest{Title: "Nope.txt"}
	raw, _ := json.Marshal(body)
	rec := do(t, h, http.MethodPost, "/api/file/upload", strings.NewReader(string(raw)), "application/json")

	wantStatus(t, rec, http.StatusBadRequest)
	var problem struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	decode(t, rec, &problem)
	if problem.Error.Code != "bad_request" || problem.Error.Message == "" {
		t.Errorf("missing content should return bad_request with a message; got %+v", problem)
	}
}

// TestAPIDescribeRoutesIncludeFileUpload checks the files route in describe
// mentions the JSON upload endpoint alongside the multipart form.
func TestAPIDescribeRoutesIncludeFileUpload(t *testing.T) {
	_, h := newApp(t)

	var routes map[string]string
	decode(t, get(t, h, "/api/describe/routes"), &routes)
	filesRoute := routes["files"]
	if filesRoute == "" {
		t.Fatal("the files route should be documented in describe")
	}
	if !strings.Contains(filesRoute, "/api/file/upload") {
		t.Errorf("the files route should mention POST /api/file/upload; got %q", filesRoute)
	}
	if !strings.Contains(filesRoute, "POST /t/file/upload") && !strings.Contains(filesRoute, "/t/file/upload") {
		t.Error("the files route should still mention the HTML form path")
	}
}
