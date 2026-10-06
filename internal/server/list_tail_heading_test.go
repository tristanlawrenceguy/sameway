package server_test

import (
	"strings"
	"testing"
)

// After a list's rows, each a heading, adding, importing and downloading
// have a heading of their own, so an agent or a screen reader does not
// take them for the last record's.
func TestAListsOwnControlsAreNotTheLastRecords(t *testing.T) {
	_, h := newApp(t)
	postForm(t, h, "/t/note/add", nil)
	page := get(t, h, "/t/note").Body.String()
	at := strings.Index(page, `<h2 class="sw-visually-hidden">Add and download</h2>`)
	if at < 0 || strings.Index(page, "/t/note/import") < at {
		t.Errorf("the list's own controls come under a heading of their own: %s", truncate(page))
	}
}
