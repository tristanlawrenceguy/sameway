package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// maxUpload bounds a file sent inside a JSON body, which is held whole;
// a file sent as a form streams to disk and may be far larger (keep.go).
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
	// Added from a meeting's page, it is that meeting's (meeting.go).
	if m := r.FormValue("meeting"); m != "" {
		said := s.toMeeting(r, m, rec)
		s.tellAt(w, r, outcome{Title: "Added", Text: title + " is added." + said}, backOf(r, "/t/"+records.EventType+"/"+m))
		return
	}
	// The file's own page shows it; anywhere else, the message does.
	if from := r.FormValue("from"); local(from) {
		s.tellAt(w, r, outcome{Title: "Added", Text: title + " is in your files."}, from)
		return
	}
	http.Redirect(w, r, own, http.StatusSeeOther)
}

// storeUpload does the work of upload for any form with a file part: the
// chat composer uses it too. It answers http.ErrMissingFile when the form
// has no file, so a message without one is not a mistake. The file goes
// to disk as it arrives (keep.go), however large, up to 4 GB.
func (s *Server) storeUpload(r *http.Request) (*store.Record, error) {
	if _, ok := s.app.Types.Get(FileType); !ok {
		return nil, errors.New("this workspace has no file type; run sameway init --force to add it")
	}
	if r.MultipartForm == nil {
		r.Body = http.MaxBytesReader(nil, r.Body, maxFile+(1<<20))
	}
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return nil, errors.New("the selected file must be smaller than 4 GB")
		}
		if strings.Contains(err.Error(), "multipart") || err == http.ErrNotMultipart {
			return nil, fmt.Errorf("select a file to add: %w", err)
		}
		return nil, fmt.Errorf("the file did not arrive whole: %w", err)
	}
	part, header, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer part.Close()
	rec, path, err := s.keepFile(s.who(r), part, header.Filename, r.FormValue("title"), r.FormValue("description"))
	if err != nil {
		return nil, err
	}
	s.keepVoices(r, rec) // who was heard each second, for a call; voices.go
	s.readKept(rec.ID, filepath.Base(header.Filename), path, false)
	return s.app.Store.Get(FileType, rec.ID)
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
	s.fileSays(id, fields)
}

// fileSays writes what reading a file found: its status, its note, its
// text. It goes through Apply like every write, but it is the file's own
// bookkeeping, not anyone's change, so it is not logged; and a file gone
// meanwhile stays gone.
func (s *Server) fileSays(id string, fields map[string]any) (*store.Record, error) {
	if _, err := s.app.Store.Get(FileType, id); err != nil {
		return nil, err
	}
	if _, err := records.ApplyOps(s.app.Store, records.Op{Type: FileType, ID: id, After: fields}); err != nil {
		return nil, err
	}
	return s.app.Store.Get(FileType, id)
}

// convertLater hands a file to the workspace's converter and finishes the
// record when it answers, however long that takes; the log says how it went.
func (s *Server) convertLater(id, converter, name, path string) {
	md, err := convert.External(context.Background(), converter, name, path)
	if err != nil {
		s.fileSays(id, map[string]any{"status": "failed", "note": err.Error()})
		records.Record(s.app.Store, "system", records.Change{Action: "failed", Detail: "converting " + name + ": " + err.Error()})
		s.Changed()
		log.Printf("files: %s: %v", name, err)
		return
	}
	rec, err := s.fileSays(id, map[string]any{"status": "ready", "text": md, "note": "converted by " + converter})
	if err != nil {
		return
	}
	title, _ := rec.Fields["title"].(string)
	records.Record(s.app.Store, "system", records.Change{Action: "updated", Component: FileType, ID: id, Detail: title + " (converted)", Href: "/t/" + FileType + "/" + id})
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
			b.WriteString(`<p class="sw-muted">This picture has no description yet, so someone who cannot see it hears only its name. Press Edit to say what it shows, or ask the assistant for a draft to check.</p>`)
			ask := "Describe this picture (/t/" + FileType + "/" + rec.ID + ") for someone who cannot see it, as a draft I will check."
			fmt.Fprintf(&b, `<p>%s</p>`, s.part(ui.Link{Href: "/chat?prompt=" + url.QueryEscape(ask), Label: "Ask the assistant to describe it", Look: ui.LookButton}))
		}
		b.WriteString(string(s.component("image", s.pictureOf(rec, alt))))
	}
	audio := isRecording(rec)
	if audio {
		props := s.recordingOf(rec)
		b.WriteString(s.speechOffer(r, rec, props))
		b.WriteString(s.recordingOffer(r, rec)) // meeting.go
		b.WriteString(string(s.component("media", props)))
	}
	// Reading a file through a converter says so, and says how it ended:
	// the page follows when it does (convertLater calls Changed).
	switch rec.Fields["status"] {
	case "converting":
		message := "Reading the file. Its text appears here when the converter answers."
		if audio {
			message = "Writing down what is said, on this computer. The transcript appears here when it is done."
			if note, _ := rec.Fields["note"].(string); strings.Contains(note, "parts done") {
				message = note + " The transcript appears here when it is done."
			}
		}
		b.WriteString(string(s.part(ui.Status{ID: "file-status", Message: message, State: ui.Working})))
	case "failed":
		note, _ := rec.Fields["note"].(string)
		b.WriteString(string(s.part(ui.Status{ID: "file-status", Message: "Could not read the file: " + note, State: ui.Failed})))
	}
	// A calendar's events are a press from the calendar.
	if rec.Fields["kind"] == "calendar" {
		if t, ok := s.app.Types.Get("event"); ok && s.importable(t) {
			fmt.Fprintf(&b, `<p>%s</p>`, s.part(ui.Link{Href: "/t/event/import?file=" + rec.ID, Label: "Add these events to the calendar", Look: ui.LookButton}))
		}
	}
	fmt.Fprintf(&b, `<p>%s</p>`, s.part(ui.Link{Href: "/files/" + rec.ID, Label: "Open the original", Look: ui.LookButton}))
	return b.String()
}
