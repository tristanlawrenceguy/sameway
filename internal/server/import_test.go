package server_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// From a list page a person imports a file: it is kept with their files,
// read as a table, shown with each column matched to a field to confirm,
// and its rows become records, said in the chat and logged. An agent
// does the same through the API with a kept file.
func TestAPersonImportsPeopleFromAFile(t *testing.T) {
	a, h := newApp(t)
	if !strings.Contains(get(t, h, "/t/person").Body.String(), "Import people from a file") {
		t.Error("the list page offers to import from a file")
	}
	if !strings.Contains(get(t, h, "/t/person/import").Body.String(), `name="file"`) {
		t.Error("the import page asks for a file")
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("file", "contacts.csv")
	part.Write([]byte("Full name,E-mail,Company\nSandra Lee,sandra@example.com,Acme\nTom Ash,tom@example.com,Bee Ltd\n"))
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/t/person/import", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/t/person/import?file=") {
		t.Fatalf("the file is kept and the preview comes next, got %d to %q", rec.Code, rec.Header().Get("Location"))
	}
	preview := get(t, h, rec.Header().Get("Location")).Body.String()
	if !strings.Contains(preview, "2 rows in contacts.csv") || !strings.Contains(preview, `<option value="organisation" selected>`) || !strings.Contains(preview, "Sandra Lee") {
		t.Errorf("the preview shows the rows with each column matched\n%s", preview)
	}
	fileID := strings.TrimPrefix(rec.Header().Get("Location"), "/t/person/import?file=")

	res := postForm(t, h, "/t/person/import/"+fileID+"/run", url.Values{"map-Full name": {"name"}, "map-E-mail": {"email"}, "map-Company": {""}})
	list, at := landed(t, h, res)
	people, _ := a.Store.List("person", store.ListOptions{})
	if org, _ := people[0].Fields["organisation"].(string); len(people) != 2 || org != "" {
		t.Errorf("two people, as the mapping said, without the column set to nothing: %v", people)
	}
	if at != "/t/person" || !strings.Contains(list.Body.String(), "imported 2") {
		t.Errorf("the list the person lands on says what happened, got %q", at)
	}
	if strings.Contains(get(t, h, "/chat").Body.String(), "imported 2") {
		t.Error("an import that went well is not an error in the chat")
	}
	if !strings.Contains(get(t, h, "/activity").Body.String(), "imported") {
		t.Error("the import is in the activity log")
	}

	// The API, with a kept file, matching by name.
	body.Reset()
	mw = multipart.NewWriter(&body)
	part, _ = mw.CreateFormFile("file", "mail.mbox")
	part.Write([]byte("From sandra@example.com Fri Sep 05 09:12:00 2026\nFrom: Sandra Lee <sandra@example.com>\nSubject: September hours\nDate: Fri, 05 Sep 2026 09:12:00 +0100\n\nHere they are.\n"))
	mw.Close()
	req = httptest.NewRequest(http.MethodPost, "/t/file/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	files, _ := a.Store.List("file", store.ListOptions{})
	mailFile := ""
	for _, f := range files {
		if f.Fields["name"] == "mail.mbox" {
			mailFile = f.ID
		}
	}
	if mailFile == "" {
		t.Fatal("the mailbox was kept as a file")
	}
	req = httptest.NewRequest(http.MethodPost, "/api/import/interaction", strings.NewReader(`{"file": "`+mailFile+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	if rec.Code != http.StatusOK || out["made"] != float64(1) || out["linked"] != float64(1) {
		t.Errorf("the API imports the mailbox as an interaction linked to Sandra, got %d %s", rec.Code, rec.Body.String())
	}
	inter, _ := a.Store.List("interaction", store.ListOptions{})
	if len(inter) != 1 || inter[0].Fields["person"] != people[0].ID && inter[0].Fields["person"] != people[1].ID {
		t.Errorf("the interaction is with the Sandra already here: %v", inter)
	}
}
