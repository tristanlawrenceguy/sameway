package media

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/ipp"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// A PDF or a photo kept here prints on a printer at home, from wherever
// the person is: from a phone on the way back, it is on paper when they
// arrive. The printers are those the home network announces (mDNS, as
// sameway discover finds them). Asking takes seconds, so it is done in
// the background every few minutes and a file's page offers the printers
// last heard from that take it.

// printers are the printers last heard from, and how to look for them.
type printers struct {
	mu    sync.Mutex
	known []ipp.Printer
	find  func(context.Context) ([]ipp.Printer, error)
}

// UsePrinters sets how printers are found, for tests.
func (s *Service) UsePrinters(find func(context.Context) ([]ipp.Printer, error)) {
	s.printers.mu.Lock()
	defer s.printers.mu.Unlock()
	s.printers.find = find
}

// printersJob looks for printers at the start and every few minutes: one
// switched on later is offered soon after.
func (s *Service) printersJob() runner.Job {
	return runner.Job{Name: "Printers looked for on the home network", Every: 5 * time.Minute, Run: func(ctx context.Context, _ time.Time) error {
		s.printers.mu.Lock()
		find := s.printers.find
		s.printers.mu.Unlock()
		if find == nil {
			find = func(ctx context.Context) ([]ipp.Printer, error) { return ipp.Find(ctx, 3*time.Second) }
		}
		found, err := find(ctx)
		if err != nil {
			return err
		}
		s.printers.mu.Lock()
		s.printers.known = found
		s.printers.mu.Unlock()
		return nil
	}}
}

// printable is a file's kind as a printer takes it, "" when no printer
// does: a PDF, a JPEG or a PNG.
func printable(path string) string {
	return map[string]string{".pdf": "application/pdf", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png"}[strings.ToLower(filepath.Ext(path))]
}

// printersFor are the printers known that take a kind of file.
func (s *Service) printersFor(format string) []ipp.Printer {
	s.printers.mu.Lock()
	defer s.printers.mu.Unlock()
	var out []ipp.Printer
	for _, p := range s.printers.known {
		if format != "" && p.Takes(format) {
			out = append(out, p)
		}
	}
	return out
}

// printOffer is the way to print a file on its page, for someone who may
// change things, when a printer at home takes it.
func (s *Service) printOffer(r *http.Request, rec *store.Record) string {
	path, _ := s.StoredPath(rec)
	found := s.printersFor(printable(path))
	if !web.MayChange(r) || len(found) == 0 {
		return ""
	}
	form := ui.Form{Action: "/files/" + rec.ID + "/print", Button: &ui.Button{Label: "Print on " + found[0].Name, Variant: ui.Secondary}}
	if len(found) == 1 {
		form.Hidden = ui.Hidden("printer", found[0].Name)
		return string(s.Form(form))
	}
	var names []string
	for _, p := range found {
		names = append(names, p.Name)
	}
	form.Class, form.Button.Label = "sw-stack", "Print"
	form.Body = s.Component("select", map[string]any{"label": "Print on", "name": "printer", "id": "print-on", "as": "dropdown", "options": names, "value": names[0]})
	return string(s.Form(form))
}

// printFile sends a file to a printer at home and says how it went. With
// no printer named, the one printer that takes it.
func (s *Service) printFile(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(records.FileType, r.PathValue("id"))
	if err != nil {
		s.Failed(w, r, "Not printed", errors.New("there is no such file"), "/t/"+records.FileType)
		return
	}
	page := relate.Page(records.FileType, rec.ID)
	path, _ := s.StoredPath(rec)
	format := printable(path)
	if format == "" {
		s.Failed(w, r, "Not printed", errors.New("a printer takes a PDF, a JPEG or a PNG, and this is none of them"), page)
		return
	}
	found, name := s.printersFor(format), r.FormValue("printer")
	var to *ipp.Printer
	for i, p := range found {
		if p.Name == name || name == "" && len(found) == 1 {
			to = &found[i]
		}
	}
	if to == nil {
		s.Failed(w, r, "Not printed", errors.New("no printer at home that takes it answers by that name; the file's page lists those that do"), page)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		s.Failed(w, r, "Not printed", errors.New("the file is not on this computer"), page)
		return
	}
	defer f.Close()
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	title, _ := rec.Fields["title"].(string)
	if err := ipp.Print(ctx, *to, f, format, title); err != nil {
		s.Failed(w, r, "Not printed", err, page)
		return
	}
	s.Tell(w, r, web.Outcome{Title: "Sent to the printer", Text: to.Name + " has " + title + " and prints it now."}, page)
}
