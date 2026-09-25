package server_test

import (
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// A link to a record in a reply reads as the record's name, not its address.
func TestALinkInAReplyReadsAsTheRecordsName(t *testing.T) {
	a, h := newApp(t)
	note, _ := a.Store.Create("note", map[string]any{"title": "Water the plants"})
	a.Store.Create(chat.MessageType, map[string]any{"role": "assistant", "content": "Done: it is at /t/note/" + note.ID + "."})
	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, `href="/t/note/`+note.ID+`">Water the plants</a>`) {
		t.Errorf("the link says the note's name; body: %s", truncate(page))
	}
}

// A block's tone is said in words, not only tinted.
func TestAToneIsSaidInWords(t *testing.T) {
	a, h := newApp(t)
	a.Store.Create("block", map[string]any{"component": "heading", "props": map[string]any{"text": "Rent is due"}, "tone": "danger"})
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `<p class="sw-visually-hidden">Important.</p>`) {
		t.Errorf("a danger tone says Important; body: %s", truncate(page))
	}
	css, _ := os.ReadFile("../../design/base/06-layout.css")
	if !strings.Contains(string(css), `.sw-block[data-tone="danger"]  { border-inline-start: 4px solid`) {
		t.Error("a tone is also an edge, which survives forced colours")
	}
}

// A picture says what it shows, when someone said so; one nobody has
// described yet says that, not its file name, and asks for words. look
// flags a file name as a description.
func TestAPictureIsDescribedNotNamedByItsFile(t *testing.T) {
	a, h := newApp(t)
	pic, _ := a.Store.Create("file", map[string]any{"title": "IMG_4032", "name": "IMG_4032.jpg", "kind": "image"})
	page := get(t, h, "/t/file/"+pic.ID).Body.String()
	if !strings.Contains(page, `alt="Picture: IMG_4032, not described yet"`) || !strings.Contains(page, "has no description yet") {
		t.Errorf("an undescribed picture says so and asks; body: %s", truncate(page))
	}
	o, _ := look.Fragment(`<img src="/files/x" alt="IMG_4032.jpg">`)
	if len(o.Problems) == 0 || !strings.Contains(o.Problems[0], "file name") {
		t.Errorf("look flags a file name as a picture's description, got %v", o.Problems)
	}
	if up := get(t, h, "/t/file").Body.String(); !strings.Contains(up, "What the picture shows") {
		t.Error("the upload asks what a picture shows")
	}
}

// Help is in the same place on every page, and makes Sameway easier to
// use without the assistant: larger words, in one press, undone in one.
func TestHelpIsEverywhereAndChangesTheTextSize(t *testing.T) {
	_, h := newApp(t)
	for _, path := range []string{"/", "/chat", "/t/note"} {
		if !strings.Contains(get(t, h, path).Body.String(), `href="/help"`) {
			t.Errorf("%s leads to Help", path)
		}
	}
	help := get(t, h, "/help").Body.String()
	for _, want := range []string{"Asking the assistant", "Taking things back", "Keyboard", "Size of the words"} {
		if !strings.Contains(help, want) {
			t.Errorf("Help covers %q", want)
		}
	}
	r := postForm(t, h, "/help/set", url.Values{"key": {"ui.text"}, "value": {"larger"}})
	page := after(t, h, r).Body.String()
	if !strings.Contains(page, `data-text="larger"`) || !strings.Contains(page, "Now: larger.") {
		t.Fatalf("the words are larger, and it says so; body: %s", truncate(page))
	}
	undo := regexp.MustCompile(`action="(/activity/[^/"]+/undo)" class="sw-outcome__undo"`).FindStringSubmatch(page)
	if undo == nil {
		t.Fatal("the change carries Undo")
	}
	postForm(t, h, undo[1], url.Values{"from": {"/help"}})
	if strings.Contains(get(t, h, "/help").Body.String(), `data-text="larger"`) {
		t.Error("undo puts the size back")
	}
	if rec := postForm(t, h, "/help/set", url.Values{"key": {"llm.base_url"}, "value": {"https://elsewhere"}}); !strings.Contains(after(t, h, rec).Body.String(), "not something this page changes") {
		t.Error("Help changes only what it offers")
	}
}

// The workspace's language is the pages' language.
func TestThePagesSayTheWorkspacesLanguage(t *testing.T) {
	a, h := newApp(t)
	a.Workspace.Config.UI.Language = "de"
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `<html lang="de"`) {
		t.Errorf("the page says it is German; got %.120s", page)
	}
}

// Scripts: an arriving block is never out of the accessibility tree, and
// a refresh keeps focus on a control with no id.
func TestArrivalAndRefreshKeepThingsReachable(t *testing.T) {
	motion, _ := os.ReadFile("../../design/base/03-motion.css")
	if m := regexp.MustCompile(`@keyframes sw-arrive-content[^\n]*`).Find(motion); m == nil || strings.Contains(string(m), "visibility") {
		t.Errorf("arrival fades with opacity only: %s", m)
	}
	refresh, _ := os.ReadFile("../../design/base/17-refresh.js")
	if !strings.Contains(string(refresh), "var focusMark = mark(focused);") || !strings.Contains(string(refresh), "find(focusMark)") {
		t.Error("a refresh finds a focused control with no id again")
	}
	_ = http.StatusOK
}
