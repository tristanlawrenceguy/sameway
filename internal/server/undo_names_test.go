package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// An undo of a setting change logged before setting names existed reads in
// words on the log, with one colon: "You undid: You changed text size to
// Large", not "You undid:: You set ui.text large".
func TestOldSettingUndoReadsInWords(t *testing.T) {
	a, h := newApp(t)
	old, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "You set ui.text large", "actor": "human", "action": "set", "target": "ui.text", "detail": "large",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Store.Create(chat.ActivityType, map[string]any{
		"summary": "You undid: You set ui.text large", "actor": "human", "action": "set", "target": "ui.text", "detail": "normal", "undoes": old.ID,
	}); err != nil {
		t.Fatal(err)
	}
	page := said(get(t, h, "/activity").Body.String())
	if !strings.Contains(page, "You undid: You changed text size to Large") {
		t.Errorf("the undo should read in words with one colon, got:\n%s", page)
	}
	if strings.Contains(page, "ui.text") || strings.Contains(page, "undid::") {
		t.Errorf("the log should show no key and no doubled colon, got:\n%s", page)
	}
}
