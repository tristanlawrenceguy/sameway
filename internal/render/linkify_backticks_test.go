package render

import (
	"strings"
	"testing"
)

// A link a model wrapped in code marks reads as the link, without the marks.
func TestALinkInCodeMarksLosesTheMarks(t *testing.T) {
	got := string(linkify("You can open it at `[Call the dentist](/t/task/abc)`."))
	if strings.Contains(got, "`") || !strings.Contains(got, `<a class="sw-link" href="/t/task/abc">Call the dentist</a>`) {
		t.Errorf("got %s", got)
	}
}
