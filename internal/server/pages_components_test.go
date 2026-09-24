package server_test

import (
	"fmt"
	"strings"
	"testing"
)

// TestALongListIsReadAPageAtATime: a list longer than a page shows one page
// of it, says which page in its title, and offers the rest by number.
func TestALongListIsReadAPageAtATime(t *testing.T) {
	a, h := newApp(t)
	for i := 0; i < 120; i++ {
		if _, err := a.Store.Create("note", map[string]any{"title": fmt.Sprintf("Note %03d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	first := get(t, h, "/t/note").Body.String()
	if !strings.Contains(first, `data-component="pagination"`) || !strings.Contains(first, `href="/t/note?page=3"`) {
		t.Errorf("120 notes should offer pages 2 and 3")
	}
	if strings.Contains(first, `rel="prev"`) {
		t.Errorf("the first page has nothing before it")
	}
	last := get(t, h, "/t/note?page=3").Body.String()
	for _, want := range []string{`<title>Notes, page 3 of 3`, `aria-current="page"><span class="sw-visually-hidden">Page </span>3</a>`, `rel="prev"`} {
		if !strings.Contains(last, want) {
			t.Errorf("page 3 should carry %s", want)
		}
	}
	if n := strings.Count(last, `data-component="card"`) + strings.Count(last, `class="sw-row`); n == 0 {
		t.Errorf("page 3 should show the last notes")
	}
	if past := get(t, h, "/t/note?page=9").Body.String(); !strings.Contains(past, "page 3 of 3") {
		t.Errorf("a page past the end should show the last")
	}
}

// TestManyResultsAreReadAPageAtATime: search keeps its words on every page.
func TestManyResultsAreReadAPageAtATime(t *testing.T) {
	a, h := newApp(t)
	for i := 0; i < 25; i++ {
		a.Store.Create("note", map[string]any{"title": fmt.Sprintf("Fern %d", i)})
	}
	body := get(t, h, "/search?q=fern").Body.String()
	if !strings.Contains(body, "25 things found") || !strings.Contains(body, `href="/search?page=2&amp;q=fern"`) {
		t.Errorf("25 results should say so and offer page 2 with the same words")
	}
	if n := strings.Count(get(t, h, "/search?q=fern&page=2").Body.String(), `class="sw-dotted"`); n != 5 {
		t.Errorf("page 2 should hold the last 5 results, got %d", n)
	}
}

// TestAPictureIsTheImageComponent: a file's picture is drawn by the image
// component, with words for someone who cannot see it.
func TestAPictureIsTheImageComponent(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("file", map[string]any{"title": "IMG_4032", "kind": "image"})
	body := get(t, h, "/t/file/"+rec.ID).Body.String()
	if !strings.Contains(body, `data-component="image"`) || !strings.Contains(body, `alt="Picture: IMG_4032, not described yet"`) {
		t.Errorf("an undescribed picture should be the image component, saying it is not described")
	}
}
