package server

import (
	"context"
	"errors"
	"fmt"
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
		if errors.Is(err, http.ErrMissingFile) || strings.Contains(err.Error(), "select a file") {
			err = errors.New("choose a file first")
		}
		s.failed(w, r, "Not added", err, "/t/"+FileType)
		return
	}
	own := "/t/" + FileType + "/" + rec.ID
	title, _ := rec.Fields["title"].(string)
	// The file's own page shows it; anywhere else, the message does.
	if from := r.FormValue("from"); local(from) {
		s.tellAt(w, r, outcome{Title: "Added", Text: title + " is in your files."}, from)
		return
	}
	http.Redirect(w, r, own, http.StatusSeeOther)
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
			return nil, fmt.Errorf("select a file to add: %w", err)
		}
		return nil, fmt.Errorf("the selected file must be smaller than 64 MB")
	}
	part, header, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer part.Close()
	data, err := io.ReadAll(io.LimitReader(part, maxUpload+1))
	if err != nil || len(data) > maxUpload {
		return nil, errors.New("the selected file must be smaller than 64 MB")
	}
	if len(data) == 0 {
		return nil, errors.New("the selected file is empty")
	}
	name := filepath.Base(header.Filename)
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = strings.TrimSuffix(name, filepath.Ext(name))
	}
	fields := map[string]any{"title": title, "name": name, "kind": convert.Kind(name), "size": len(data), "status": "converting"}
	// What a picture shows, said by whoever added it, for whoever cannot see it.
	if d := strings.TrimSpace(r.FormValue("description")); d != "" {
		fields["description"] = d
	}
	rec, err := s.app.Store.Create(FileType, fields)
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
	s.record(r, chat.Change{Action: "added", Component: FileType, ID: rec.ID, Detail: title, Href: "/t/" + FileType + "/" + rec.ID})

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
	case res.Audio:
		// A .webm is heard or seen; what is in it says which.
		fields["kind"] = convert.KindOf(name, s.headOf(id))
		fields["note"] = "A recording's text is its transcript; there is none yet."
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
		s.Changed()
		log.Printf("files: %s: %v", name, err)
		return
	}
	rec, err := s.app.Store.Update(FileType, id, map[string]any{"status": "ready", "text": md, "note": "converted by " + converter})
	if err != nil {
		return
	}
	title, _ := rec.Fields["title"].(string)
	chat.Record(s.app.Store, "system", chat.Change{Action: "updated", Component: FileType, ID: id, Detail: title + " (converted)", Href: "/t/" + FileType + "/" + id})
	s.Changed()
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
	kind, _ := rec.Fields["kind"].(string)
	ct := convert.MediaType(stored, kind)
	if ct == "" {
		ct = mime.TypeByExtension(filepath.Ext(stored))
	}
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	fileSafety(w, ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, strings.ReplaceAll(name, `"`, "")))
	http.ServeFile(w, r, filepath.Join(s.app.Workspace.FilesDir(), stored))
}

// fileExtras is what a file's own page shows beyond its fields: the
// picture itself when it is one, and the way to the original.
func (s *Server) fileExtras(r *http.Request, rec *store.Record) string {
	var b strings.Builder
	if kind, _ := rec.Fields["kind"].(string); kind == "image" {
		alt, _ := rec.Fields["description"].(string)
		if alt == "" {
			// Its name alone, "IMG_4032", tells nobody what it shows: it
			// says that it is a picture no one has described yet, and the
			// page asks for a description.
			title, _ := rec.Fields["title"].(string)
			alt = title + ", not described yet"
			b.WriteString(`<p class="sw-muted">This picture has no description yet, so someone who cannot see it hears only its name. Press Edit to say what it shows.</p>`)
		}
		b.WriteString(string(s.component("image", s.pictureOf(rec, alt))))
	}
	audio := isRecording(rec)
	if audio {
		props := s.recordingOf(rec)
		b.WriteString(s.speechOffer(r, rec, props))
		b.WriteString(string(s.component("media", props)))
	}
	// Reading a file through a converter says so, and says how it ended:
	// the page follows when it does (convertLater calls Changed).
	switch rec.Fields["status"] {
	case "converting":
		message := "Reading the file. Its text appears here when the converter answers."
		if audio {
			message = "Writing down what is said, on this computer. The transcript appears here when it is done."
		}
		b.WriteString(string(s.component("status", map[string]any{"id": "file-status", "message": message, "state": "working"})))
	case "failed":
		note, _ := rec.Fields["note"].(string)
		b.WriteString(string(s.component("status", map[string]any{"id": "file-status", "message": "Could not read the file: " + note, "state": "error"})))
	}
	fmt.Fprintf(&b, `<p>%s</p>`, s.component("link", map[string]any{"href": "/files/" + rec.ID, "label": "Open the original", "look": "button"}))
	return b.String()
}
