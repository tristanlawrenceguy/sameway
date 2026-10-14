package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Measured on the person's phone, Layout now says what the first screen
// shows and how long the tab runs there; a long one is told to put what
// the person comes for first.
func TestLayoutNowSaysThePhonesFirstScreen(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Measured = chat.NewMeasures("", nil)
	var ids []string
	for _, label := range []string{"Today", "Notes", "Everything"} {
		ids = append(ids, idOf(run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": label}, "span": 12})))
	}
	var rs []chat.Reading
	for i, top := range []int{0, 500, 900} {
		r := reading(t, svc, ids[i], 390, []int{500, 400, 3000}[i])
		r.Top = top
		rs = append(rs, r)
	}
	svc.Measured.Put(rs)
	said := svc.LayoutNow("")
	for _, want := range []string{"first screen (about 700px) shows \"Today\" collection, \"Notes\" collection;", "runs 3,900px there, about 6 screens", "Put what the person comes for first"} {
		if !strings.Contains(said, want) {
			t.Errorf("Layout now says %q:\n%s", want, said)
		}
	}
}
