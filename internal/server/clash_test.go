package server_test

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func clashOn(t *testing.T, s *store.Store, page, other string) (string, string) {
	t.Helper()
	n, err := s.Create("note", map[string]any{"title": "Plan", "body": page})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	s.Put(store.ClashType, "c1", map[string]any{"target": "note", "target_id": n.ID, "field": "body", "text": other, "origin": "elsewhere", "state": "open"}, now, now)
	return n.ID, "/t/note/" + n.ID
}

// The offer shows what differs, marked in words as well as by shape, and
// names the page's version rather than where it sits.
func TestAClashShowsWhatDiffers(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	_, href := clashOn(t, a.Store, "Tea and toast at nine.", "Tea and cake at nine.")
	page := get(t, h, href).Body.String()
	for _, want := range []string{
		"What differs from the page&#39;s version",
		`<span class="sw-clash__page"><span class="sw-visually-hidden">(only in the page&#39;s version: </span>toast`,
		`<span class="sw-clash__this"><span class="sw-visually-hidden">(only in this one: </span>cake`,
		"Keep both", "Keep the page&#39;s version",
	} {
		if !strings.Contains(page, want) && !strings.Contains(page, strings.ReplaceAll(want, "&#39;", "'")) {
			t.Errorf("the offer should have %q\n%s", want, truncate(page))
		}
	}
	if strings.Contains(page, "the one below") {
		t.Error("the page's version is named, not placed")
	}
}

// Each answer returns to the record's page and says what it did, with
// its Undo; Keep both keeps both texts.
func TestAClashAnswerIsSaidAndCanBeUndone(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	id, href := clashOn(t, a.Store, "the later words", "the earlier words")
	r := postForm(t, h, "/clash/c1/both", url.Values{"from": {href}})
	if loc := r.Header().Get("Location"); !strings.HasPrefix(loc, href) {
		t.Errorf("the answer returns to the record's page, not %q", loc)
	}
	page := after(t, h, r).Body.String()
	if !strings.Contains(page, "Both versions are kept") || !strings.Contains(page, "sw-outcome__undo") {
		t.Errorf("Keep both is said, with its Undo\n%s", truncate(page))
	}
	if got, _ := a.Store.Get("note", id); got.Fields["body"] != "the later words\n\nthe earlier words" {
		t.Errorf("both are kept, the page's first: %q", got.Fields["body"])
	}
	if strings.Contains(page, "Another version of") {
		t.Error("once chosen, it is no longer offered")
	}
}

// Keeping the page's version sets the other aside, and Undo offers it
// again: no answer loses anyone's words for good.
func TestKeepingThePagesVersionCanBeUndone(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	_, href := clashOn(t, a.Store, "the later words", "the earlier words")
	page := after(t, h, postForm(t, h, "/clash/c1/keep", url.Values{"from": {href}})).Body.String()
	undo := regexp.MustCompile(`action="(/activity/[^/"]+/undo)" class="sw-outcome__undo"`).FindStringSubmatch(page)
	if undo == nil || strings.Contains(page, "Another version of") {
		t.Fatalf("Keep is said with its Undo, and the offer goes\n%s", truncate(page))
	}
	postForm(t, h, undo[1], url.Values{"from": {href}})
	if page := get(t, h, href).Body.String(); !strings.Contains(page, "Another version of") || !strings.Contains(page, "the earlier words") {
		t.Errorf("Undo offers the other version again\n%s", truncate(page))
	}
}

// Using the other version can be undone: the page's words come back.
func TestUsingTheOtherVersionCanBeUndone(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	id, href := clashOn(t, a.Store, "the later words", "the earlier words")
	page := after(t, h, postForm(t, h, "/clash/c1/use", url.Values{"from": {href}})).Body.String()
	undo := regexp.MustCompile(`action="(/activity/[^/"]+/undo)" class="sw-outcome__undo"`).FindStringSubmatch(page)
	if undo == nil {
		t.Fatalf("Use is said with its Undo\n%s", truncate(page))
	}
	postForm(t, h, undo[1], url.Values{"from": {href}})
	if got, _ := a.Store.Get("note", id); got.Fields["body"] != "the later words" {
		t.Errorf("Undo puts the page's words back: %q", got.Fields["body"])
	}
}
