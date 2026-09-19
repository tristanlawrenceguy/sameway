package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// FileType is the content type a person's files become.
const FileType = "file"

// maxUpload bounds one file; a workspace is a person's folder, not a store.
const maxUpload = 64 << 20

// upload takes a file from the form, keeps the original under files/,
// makes the record, and reads the contents into text: at once for the
// formats the binary reads, and in the background when the workspace
// names a converter, with the record saying "converting" meanwhile.
func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	rec, err := s.storeUpload(r)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) || strings.Contains(err.Error(), "please select a file") {
			s.uploadError(w, r, err)
		} else {
			s.fail(w, err)
		}
		return
	}
	back := "/t/" + FileType + "/" + rec.ID
	if from := r.FormValue("from"); strings.HasPrefix(from, "/") && !strings.HasPrefix(from, "//") {
		back = from
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// storeUpload does the work of upload for any form with a file part: the
// chat composer uses it too. It answers http.ErrMissingFile when the form
// has no file, so a message without one is not a mistake.
func (s *Server) storeUpload(r *http.Request) (*store.Record, error) {
	if _, ok := s.app.Types.Get(FileType); !ok {
		return nil, errors.New("this workspace has no file type; run sameway init --force to add it")
	}
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		if strings.Contains(err.Error(), "multipart") || err == http.ErrNotMultipart {
			return nil, fmt.Errorf("please select a file: %w", err)
		}
		return nil, fmt.Errorf("the file is too large: 64 MB is the most one can be")
	}
	part, header, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer part.Close()
	data, err := io.ReadAll(io.LimitReader(part, maxUpload+1))
	if err != nil || len(data) > maxUpload {
		return nil, errors.New("the file is too large: 64 MB is the most one can be")
	}
	name := filepath.Base(header.Filename)
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = strings.TrimSuffix(name, filepath.Ext(name))
	}
	rec, err := s.app.Store.Create(FileType, map[string]any{
		"title": title, "name": name, "kind": convert.Kind(name), "size": len(data), "status": "converting",
	})
	if err != nil {
		return nil, err
	}
	stored := rec.ID + strings.ToLower(filepath.Ext(name))
	dir := s.app.Workspace.FilesDir()
	if err := os.MkdirAll(dir, 0o755); err == nil {
		err = os.WriteFile(filepath.Join(dir, stored), data, 0o644)
	}
	if err != nil {
		s.app.Store.Delete(FileType, rec.ID)
		return nil, fmt.Errorf("could not keep the file: %w", err)
	}
	s.app.Store.Update(FileType, rec.ID, map[string]any{"path": stored})
	chat.Record(s.app.Store, "human", chat.Change{Action: "added", Component: FileType, ID: rec.ID, Detail: title, Href: "/t/" + FileType + "/" + rec.ID})

	if converter := s.app.Workspace.Config.Files.Convert[convert.Ext(name)]; converter != "" {
		go s.convertLater(rec.ID, converter, name, filepath.Join(dir, stored))
	} else {
		s.readNow(rec.ID, name, data)
	}
	return rec, nil
}

// readNow reads a file the binary understands and finishes the record.
func (s *Server) readNow(id, name string, data []byte) {
	res, err := convert.Read(name, data)
	fields := map[string]any{"status": "ready", "kind": res.Kind}
	switch {
	case err != nil:
		fields["status"], fields["note"] = "failed", err.Error()
	case res.Image:
		fields["note"] = "An image has no text of its own; its description is what anyone who cannot see it gets."
	default:
		fields["text"] = res.Markdown
	}
	s.app.Store.Update(FileType, id, fields)
}

// convertLater hands a file to the workspace's converter and finishes the
// record when it answers, however long that takes; the log says how it went.
func (s *Server) convertLater(id, converter, name, path string) {
	md, err := convert.External(context.Background(), converter, name, path)
	if err != nil {
		s.app.Store.Update(FileType, id, map[string]any{"status": "failed", "note": err.Error()})
		chat.Record(s.app.Store, "system", chat.Change{Action: "failed", Detail: "converting " + name + ": " + err.Error()})
		log.Printf("files: %s: %v", name, err)
		return
	}
	rec, err := s.app.Store.Update(FileType, id, map[string]any{"status": "ready", "text": md, "note": "converted by " + converter})
	if err != nil {
		return
	}
	title, _ := rec.Fields["title"].(string)
	chat.Record(s.app.Store, "system", chat.Change{Action: "updated", Component: FileType, ID: id, Detail: title + " (converted)", Href: "/t/" + FileType + "/" + id})
}

// serveFile gives back the original, as the type it is.
func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(FileType, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	stored, _ := rec.Fields["path"].(string)
	if stored == "" || strings.ContainsAny(stored, `/\`) {
		http.NotFound(w, r)
		return
	}
	name, _ := rec.Fields["name"].(string)
	if ct := mime.TypeByExtension(filepath.Ext(stored)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, strings.ReplaceAll(name, `"`, "")))
	http.ServeFile(w, r, filepath.Join(s.app.Workspace.FilesDir(), stored))
}

// uploadError renders an HTML error page on the files list with an accessible
// alert banner (aria-live) so screen readers announce the validation failure.
func (s *Server) uploadError(w http.ResponseWriter, r *http.Request, err error) {
	var b strings.Builder
	b.WriteString(string(s.component("alert", map[string]any{
		"kind":    "danger",
		"title":   "Could not upload",
		"message": template.HTMLEscapeString(err.Error()),
	})))
	b.WriteString(string(s.component("upload", map[string]any{"id": "upload-error"})))
	b.WriteString(`<script>(function(){var f=document.querySelector('.sw-upload__field');var err=document.getElementById("upload-error-error");f.addEventListener('invalid',function(e){err.textContent="Please select a file."},false);document.querySelector(".sw-upload").addEventListener('submit',function(e){if(!f.value){e.preventDefault();err.textContent="Please select a file.";f.reportValidity()}},{once:true});f.addEventListener('change',function(){err.textContent=""})})();</script>`)
	// Re-render recent activity so the user can undo deletions.
	b.WriteString(string(s.recentActivity(5, "/t/"+FileType)))
	s.page(w, r, "Files", template.HTML(b.String()), pageOptions{JSONURL: "/api/file", Status: http.StatusBadRequest})
}

// fileExtras is what a file's own page shows beyond its fields: the
// picture itself when it is one, and the way to the original.
func (s *Server) fileExtras(rec *store.Record) string {
	var b strings.Builder
	if kind, _ := rec.Fields["kind"].(string); kind == "image" {
		alt, _ := rec.Fields["description"].(string)
		if alt == "" {
			alt, _ = rec.Fields["title"].(string)
		}
		fmt.Fprintf(&b, `<p><img class="sw-file__image" src="/files/%s" alt="%s"></p>`, rec.ID, template.HTMLEscapeString(alt))
	}
	if status, _ := rec.Fields["status"].(string); status == "converting" {
		b.WriteString(string(s.component("status", map[string]any{"id": "file-status", "message": "Reading the file. Its text appears here when the converter answers.", "state": "working"})))
	}
	fmt.Fprintf(&b, `<p>%s</p>`, s.component("link", map[string]any{"href": "/files/" + rec.ID, "label": "Open the original"}))
	return b.String()
}
