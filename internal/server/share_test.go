package server_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A page shared from a bookmark or a phone becomes a note with its title,
// its link and its own words, without the menus around them; words and
// files shared from a phone come along; a link to this computer is kept
// but never read.
func TestWhatIsSharedIsKeptAsANote(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<html><head><title>Site | Pancakes</title><meta property="og:title" content="Best pancakes"></head><body>
<nav>Home Recipes Login</nav><article><h1>Best pancakes</h1><p>Whisk flour, two eggs and milk.</p><script>track()</script></article><footer>Cookies</footer></body></html>`))
	}))
	defer site.Close()
	a, h := newApp(t)

	page := get(t, h, "/share?url="+url.QueryEscape(site.URL)+"&title=Pancakes&text=for+Sunday").Body.String()
	for _, want := range []string{`value="` + site.URL + `"`, `value="Pancakes"`, "for Sunday", "Save to Sameway", "javascript:"} {
		if !strings.Contains(page, want) {
			t.Errorf("the form is filled from the bookmark (%s): %s", want, truncate(page))
		}
	}

	server.ReadLocal(true)
	rec := postForm(t, h, "/share", url.Values{"url": {site.URL}, "text": {"for Sunday"}})
	server.ReadLocal(false)
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/t/note/") {
		t.Fatalf("saved, and its note opens: %d %s", rec.Code, truncate(rec.Body.String()))
	}
	note := lastNote(t, a)
	body, _ := note.Fields["body"].(string)
	if note.Fields["title"] != "Best pancakes" || !strings.Contains(body, "for Sunday") || !strings.Contains(body, "Whisk flour, two eggs and milk.") || !strings.Contains(body, "]("+site.URL+")") {
		t.Errorf("the page's title, link and words: %v", note.Fields)
	}
	for _, noise := range []string{"Login", "Cookies", "track()"} {
		if strings.Contains(body, noise) {
			t.Errorf("not the page's %q: %s", noise, body)
		}
	}

	// A phone shares words with the link in them, and a photo.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("text", "Look at this http://127.0.0.1:1/private")
	fw, _ := mw.CreateFormFile("files", "garden.txt")
	fw.Write([]byte("tomatoes by the wall"))
	mw.Close()
	rec = do(t, h, http.MethodPost, "/share", &buf, mw.FormDataContentType())
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("a phone's share is saved: %d %s", rec.Code, truncate(rec.Body.String()))
	}
	note = lastNote(t, a)
	body, _ = note.Fields["body"].(string)
	if note.Fields["title"] != "Look at this" || !strings.Contains(body, "Look at this http://127.0.0.1:1/private") || !strings.Contains(body, "[garden](/t/file/") {
		t.Errorf("the words, the link and the file: %v", note.Fields)
	}

	if rec := postForm(t, h, "/share", url.Values{}); rec.Code == http.StatusSeeOther && strings.HasPrefix(rec.Header().Get("Location"), "/t/note/") {
		t.Error("nothing shared, nothing saved")
	}
	if !strings.Contains(get(t, h, "/manifest.webmanifest").Body.String(), `"share_target"`) {
		t.Error("the installed app is in the phone's Share menu")
	}
}

func lastNote(t *testing.T, a *app.App) *store.Record {
	t.Helper()
	recs, err := a.Store.List("note", store.ListOptions{})
	if err != nil || len(recs) == 0 {
		t.Fatalf("no notes: %v", err)
	}
	last := recs[0]
	for _, r := range recs {
		if !r.CreatedAt.Before(last.CreatedAt) {
			last = r
		}
	}
	return last
}
