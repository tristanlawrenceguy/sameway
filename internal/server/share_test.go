package server_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

const pancakes = `<html><head><title>Site | Pancakes</title><meta property="og:title" content="Best pancakes"></head><body>
<nav>Home Recipes Login</nav><div role="banner">Subscribe today</div><article><h1>Best pancakes</h1><p>Whisk flour, two eggs and milk.</p>
<div aria-hidden="true">Share on Pinterest</div><p hidden>Sign up</p><script>track()</script></article><footer>Cookies</footer></body></html>`

// A page shared from a bookmark or a phone becomes a note with its title,
// its link and its own words, without the menus around them; words and
// files shared from a phone come along.
func TestWhatIsSharedIsKeptAsANote(t *testing.T) {
	t.Parallel()
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(pancakes))
	}))
	defer site.Close()
	a, h := newAppWith(t, app.Options{Pages: site.Client()})

	page := get(t, h, "/share?url="+url.QueryEscape(site.URL)+"&title=Pancakes&text=for+Sunday").Body.String()
	for _, want := range []string{`value="` + site.URL + `"`, `value="Pancakes"`, "for Sunday", "Save to Sameway", "javascript:", "noopener", "The bookmark&#39;s address"} {
		if !strings.Contains(page, want) {
			t.Errorf("the form is filled from the bookmark, and the bookmark can be copied (%s): %s", want, truncate(page))
		}
	}

	rec := postForm(t, h, "/share", url.Values{"url": {site.URL}, "text": {"for Sunday"}})
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/t/note/") {
		t.Fatalf("saved, and its note opens: %d %s", rec.Code, truncate(rec.Body.String()))
	}
	note := lastNote(t, a)
	body, _ := note.Fields["body"].(string)
	if note.Fields["title"] != "Best pancakes" || !strings.Contains(body, "for Sunday") || !strings.Contains(body, "Whisk flour, two eggs and milk.") || !strings.Contains(body, "]("+site.URL+")") {
		t.Errorf("the page's title, link and words: %v", note.Fields)
	}
	for _, noise := range []string{"Login", "Subscribe", "Pinterest", "Sign up", "Cookies", "track()"} {
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

// A link to this computer or its network is kept, but its page is never
// asked for: a shared link cannot make Sameway read a router's settings or
// another program's page for whoever shared it.
func TestALinkNearbyIsKeptButNotRead(t *testing.T) {
	t.Parallel()
	var asked atomic.Int32
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Add(1)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<title>Router</title><body>admin password</body>`))
	}))
	defer local.Close()
	a, h := newApp(t)
	rec := postForm(t, h, "/share", url.Values{"url": {local.URL + "/settings"}})
	page := said(after(t, h, rec).Body.String())
	if asked.Load() != 0 {
		t.Error("a page on this computer is never asked for")
	}
	body, _ := lastNote(t, a).Fields["body"].(string)
	if !strings.Contains(body, local.URL+"/settings") || strings.Contains(body, "admin password") {
		t.Errorf("the link kept, its page not: %s", body)
	}
	if !strings.Contains(page, "a page on this computer or its network is not read") {
		t.Errorf("the person is told why only the link is kept: %s", truncate(page))
	}
}

// A link is kept without what was added to follow whoever clicked it, and
// the same page shared again with nothing new is the note already made,
// said once; with new words, it is a note of its own.
func TestTheSamePageIsSavedOnce(t *testing.T) {
	t.Parallel()
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=windows-1252")
		w.Write([]byte("<title>Caf\xe9 menu</title><main><p>Cr\xe8me br\xfbl\xe9e on Sundays.</p></main>"))
	}))
	defer site.Close()
	a, h := newAppWith(t, app.Options{Pages: site.Client()})

	first := postForm(t, h, "/share", url.Values{"url": {site.URL + "/menu?day=sun&utm_source=news&fbclid=abc"}})
	note := lastNote(t, a)
	body, _ := note.Fields["body"].(string)
	if !strings.Contains(body, "]("+site.URL+"/menu?day=sun)") || strings.Contains(body, "utm_") || strings.Contains(body, "fbclid") {
		t.Errorf("the link without its tracking: %s", body)
	}
	if note.Fields["title"] != "Café menu" || !strings.Contains(body, "Crème brûlée") {
		t.Errorf("a page in another encoding is read in it: %v", note.Fields)
	}

	count := countNotes(t, a)
	again := postForm(t, h, "/share", url.Values{"url": {site.URL + "/menu?utm_medium=email&day=sun"}})
	if again.Header().Get("Location") != first.Header().Get("Location") {
		t.Errorf("the same page opens the note it is in: %s, not %s", again.Header().Get("Location"), first.Header().Get("Location"))
	}
	if countNotes(t, a) != count || strings.Count(said(after(t, h, again).Body.String()), "Already saved") != 1 {
		t.Errorf("no second note, and the person is told once")
	}
	before := countNotes(t, a)
	postForm(t, h, "/share", url.Values{"url": {site.URL + "/menu?day=sun"}, "text": {"for Mum's birthday"}})
	if countNotes(t, a) != before+1 {
		t.Error("new words with it are a note of their own")
	}
}

// What stops a save is said on the form, filled again with what came, so
// nothing is typed twice.
func TestANoteNotSavedKeepsWhatWasTyped(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	rec := postForm(t, h, "/share", url.Values{"url": {"ftp://example.com/x"}, "title": {"Old files"}, "text": {"from Sam"}})
	at := rec.Header().Get("Location")
	if !strings.HasPrefix(at, "/share?") || !strings.Contains(at, "title=Old+files") || !strings.Contains(at, "text=from+Sam") {
		t.Fatalf("back on the form, filled: %s", at)
	}
	page := after(t, h, rec).Body.String()
	if !strings.Contains(page, `href="#field-url"`) || !strings.Contains(said(page), "https://") || !strings.Contains(page, `value="Old files"`) {
		t.Errorf("the problem points at the link and says how to fix it: %s", truncate(said(page)))
	}
}

func countNotes(t *testing.T, a *app.App) int {
	t.Helper()
	n, err := a.Store.Count("note")
	if err != nil {
		t.Fatal(err)
	}
	return n
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
