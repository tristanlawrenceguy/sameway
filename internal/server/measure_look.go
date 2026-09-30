package server

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// LookMeasured is how the blocks on a page of blocks were drawn: in the
// browsers of the people who may change it, by the latest reading of each
// block on each kind of screen since the block last changed, and in the
// look browser's own window when the page was read with its scripts.
type LookMeasured struct {
	Note   string          `json:"note"`
	Person []chat.Reading  `json:"person"`
	Look   json.RawMessage `json:"look_browser,omitempty"`
}

const (
	measuredNote = "Each block as the person's own browser drew it (their window width, zoom, text size and spacing), on each kind of screen, taken since the block last changed: height and width in pixels, top where its row starts. content taller than box is a block scrolling inside when it was not meant to; cut is content clipped where nothing scrolls; past_edge runs off the right of the screen; pane is the height of the side pane it sits in; by_design is what scrolls on purpose (a conversation, a wide table)."
	unmeasured   = "Not measured yet: nobody who may change this page has opened it since its blocks last changed, so Layout now estimates heights from each block's kind until someone does."
	lookNote     = " look_browser is the same measuring in the look browser's 1280 by 900 window, not the person's."
)

// measuredAt is what is measured of the page at path, or nil when it is
// not a page of blocks: Home, a tab, or one block's own page.
func (s *Server) measuredAt(path string, inLook json.RawMessage) *LookMeasured {
	u, err := url.Parse(path)
	if err != nil {
		return nil
	}
	canvas, focus := "", ""
	switch p := u.Path; {
	case p == "/":
	case strings.HasPrefix(p, "/c/"):
		canvas = strings.TrimPrefix(p, "/c/")
	case strings.HasPrefix(p, "/canvas/") && !strings.Contains(strings.TrimPrefix(p, "/canvas/"), "/"):
		focus = strings.TrimPrefix(p, "/canvas/")
	default:
		return nil
	}
	if focus == "" && !s.app.Chat.HasCanvas(canvas) {
		return nil
	}
	m := &LookMeasured{Person: s.app.Chat.MeasuredOn(canvas, focus), Note: measuredNote}
	if len(m.Person) == 0 {
		m.Person, m.Note = []chat.Reading{}, unmeasured
	}
	if len(inLook) > 0 && string(inLook) != "null" {
		m.Look = inLook
		m.Note += lookNote
	}
	return m
}
