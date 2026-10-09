package server_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A whole copy goes to the cloud folder chosen, at once and then daily,
// and Start from a copy brings it back as a workspace with what it had.
func TestACopyInTheCloudBringsTheWorkspaceBack(t *testing.T) {
	a, _ := newApp(t)
	h := server.New(a).WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() {}})
	if _, err := a.Store.Create("note", map[string]any{"title": "Passport number is in the drawer"}); err != nil {
		t.Fatal(err)
	}
	cloud := t.TempDir()
	page := after(t, h, postForm(t, h, "/backup/cloud", url.Values{"folder": {cloud}})).Body.String()
	if !strings.Contains(page, "A copy is in your cloud folder") {
		t.Fatalf("a copy is made at once: %s", truncate(page))
	}
	copies, _ := filepath.Glob(filepath.Join(cloud, "Sameway copies", "*", "* ????-??-??.zip"))
	if len(copies) != 1 {
		t.Fatalf("one zip in the cloud folder: %v", copies)
	}
	data, _ := os.ReadFile(copies[0])

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "Home 2026-10-07.zip")
	fw.Write(data)
	mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/workspaces/from-copy", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Origin", "http://example.com")
	r.Host = "example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	back := filepath.Join(filepath.Dir(a.Workspace.Dir), "home") // named by its slug, as a sibling is
	b, err := app.Load(back, false)
	if err != nil {
		t.Fatalf("the copy is a workspace again beside this one (%d): %v", w.Code, err)
	}
	defer b.Close()
	if n, _ := b.Store.Count("note"); n != 1 {
		t.Errorf("with what it had: %d notes", n)
	}
}
