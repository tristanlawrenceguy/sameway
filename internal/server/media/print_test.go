package media_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/ipp"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A PDF kept here prints on a printer at home: once the network has
// announced one that takes it, the file's page offers Print, to those who
// may change things, and pressing it sends the file as it is. A kind the
// printer does not take is not offered to it.
func TestAFileKeptHerePrintsOnAPrinterAtHome(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	srv := h.(*server.Server)
	var mu sync.Mutex
	var job ipp.Message
	printer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		job, _ = ipp.Decode(body)
		mu.Unlock()
		w.Write(ipp.Message{RequestID: 1}.Encode())
	}))
	defer printer.Close()
	srv.UsePrinters(func(context.Context) ([]ipp.Printer, error) {
		return []ipp.Printer{{Name: "Kitchen printer", URL: printer.URL + "/ipp/print", Formats: []string{"application/pdf", "image/jpeg"}}}, nil
	})
	add := func(name, content string) string {
		t.Helper()
		body, ct := multipartFile(t, name, content, nil)
		rec := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
		wantStatus(t, rec, http.StatusSeeOther)
		return rec.Header().Get("Location")
	}
	letter, picture := add("Letter.pdf", "%PDF-1.4 the letter"), add("Plan.png", "not really a png")
	if strings.Contains(get(t, h, letter).Body.String(), "Print on") {
		t.Error("no printer is offered before one is heard from")
	}
	for _, j := range srv.BackgroundJobs() {
		if j.Name == "Printers looked for on the home network" {
			j.Run(context.Background(), time.Now())
		}
	}
	page := get(t, h, letter).Body.String()
	if !strings.Contains(page, "Print on Kitchen printer") {
		t.Fatalf("the PDF's page offers the printer: %s", truncate(said(page)))
	}
	if strings.Contains(get(t, h, picture).Body.String(), "Print on") {
		t.Error("a PNG is not offered to a printer that does not take one")
	}
	if strings.Contains(as(t, h, records.Visitor{Name: "Sam", Access: records.View}, http.MethodGet, letter, "", "").Body.String(), "Print on") {
		t.Error("someone who may only look is not offered it")
	}

	id := strings.TrimPrefix(letter, "/t/file/")
	done := after(t, h, postForm(t, h, "/files/"+id+"/print", url.Values{"printer": {"Kitchen printer"}})).Body.String()
	if !strings.Contains(said(done), "Kitchen printer has Letter and prints it now") {
		t.Errorf("the page says it went: %s", truncate(said(done)))
	}
	mu.Lock()
	defer mu.Unlock()
	if string(job.Data) != "%PDF-1.4 the letter" || job.Value("document-format") != "application/pdf" || job.Value("job-name") != "Letter" {
		t.Errorf("the printer got the file as it is, as a PDF: %q %+v", job.Data, job.Attrs)
	}
}
