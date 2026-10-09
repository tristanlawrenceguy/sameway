package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A change is said in a person's words: a canvas is a tab, and a piece of
// text that began with a list's dash is named by its words.
func TestAChangeIsSaidInAPersonsWords(t *testing.T) {
	t.Parallel()
	w := records.Say(nil, map[string]any{"action": "added", "target": "canvas", "detail": "Weekend"})
	if w.Target != "tab" {
		t.Errorf("a canvas is a tab: %+v", w)
	}
	w = records.Say(nil, map[string]any{"action": "added", "target": "text", "detail": "- Morning walk on Saturday"})
	if w.Detail != "Morning walk on Saturday" {
		t.Errorf("a text's words, without the dash: %+v", w)
	}
	if got := records.Sentence(nil, map[string]any{"actor": "assistant", "action": "added", "target": "canvas", "detail": "Weekend"}); got != "Assistant added tab Weekend" {
		t.Errorf("the log says it the same way: %q", got)
	}
}
