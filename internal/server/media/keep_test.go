package media_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A file larger than a document is ever read is streamed to disk whole,
// however it comes; a recording that size is kept to be written down,
// and a document that size is kept as it is, saying so.
func TestAFileBeyondReadingSizeIsKeptWhole(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	size := 65 << 20
	send := func(name string) (string, map[string]any) {
		pr, pw := io.Pipe()
		mw := multipart.NewWriter(pw)
		go func() {
			part, _ := mw.CreateFormFile("file", name)
			chunk := bytes.Repeat([]byte("a"), 1<<20)
			for i := 0; i < size>>20; i++ {
				part.Write(chunk)
			}
			mw.Close()
			pw.Close()
		}()
		req := httptest.NewRequest(http.MethodPost, "/t/file/upload", pr)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("%s: a file past 64 MB is taken, got %d: %.300s", name, rec.Code, rec.Body.String())
		}
		id := strings.TrimPrefix(rec.Header().Get("Location"), "/t/file/")
		f, err := a.Store.Get("file", id)
		if err != nil {
			t.Fatal(err)
		}
		return id, f.Fields
	}

	_, meeting := send("Meeting.m4a")
	st, err := os.Stat(filepath.Join(a.Workspace.FilesDir(), meeting["path"].(string)))
	if err != nil || st.Size() != int64(size) || meeting["kind"] != "audio" || meeting["status"] != "ready" {
		t.Errorf("the recording is kept whole, ready to be written down: %v %v", meeting, err)
	}
	_, notes := send("Notes.txt")
	if notes["status"] != "ready" || !strings.Contains(notes["note"].(string), "kept as it is") || (notes["text"] != nil && notes["text"] != "") {
		t.Errorf("a document too big to read is kept as it is, and says so: %v", notes)
	}
}
