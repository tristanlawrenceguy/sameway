package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The page measures its blocks where it is drawn and says how they came
// out: design/base/29-measure.js sends, this keeps, chat.LayoutNow and
// /api/look read. Only numbers and ids travel, never what a block says,
// and only from the people who may change the page: a published page and
// someone who may only look are never measured, since nothing they could
// do with the answer would change it.

// measureBody is what a page sends: its screen, and each block's numbers.
type measureBody struct {
	View     string         `json:"view"`   // tab or focus
	Canvas   string         `json:"canvas"` // the tab's id; "" is Home
	Viewport int            `json:"vw"`
	Ratio    float64        `json:"dpr"`
	Font     float64        `json:"font"`
	Columns  int            `json:"cols"`
	Blocks   []measureBlock `json:"blocks"`
}

type measureBlock struct {
	ID       string `json:"id"`
	Version  string `json:"v"`
	Width    int    `json:"w"`
	Height   int    `json:"h"`
	Top      int    `json:"top"`
	Content  int    `json:"sh"`
	Box      int    `json:"sb"`
	Sideways int    `json:"sx"`
	Cut      int    `json:"cut"`
	Designed int    `json:"d"`
	Past     int    `json:"past"`
	Pane     int    `json:"pane"`
}

// measureMost caps each number: no screen or block is taller than this,
// and a page sends no more blocks than this.
const (
	measureMost   = 50000
	measureBlocks = 200
)

var (
	measureID      = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	measureVersion = regexp.MustCompile(`^[0-9a-f]{16}$`)
)

// measuring says whether this request's page is measured: someone who may
// change it, in a browser, not an agent and not the public.
func measuring(r *http.Request) bool {
	v := records.VisitorOf(r.Context())
	return !v.Agent && (v.Owner() || v.Access == records.Edit || v.Access == records.Host)
}

// measurePage is the attribute that turns the measuring on for a page, and
// says which page it is; "" where it is not measured.
func measurePage(r *http.Request, view, canvas string) string {
	if !measuring(r) {
		return ""
	}
	return fmt.Sprintf(` data-measure="%s" data-measure-canvas="%s"`, view, template.HTMLEscapeString(canvas))
}

// measureAttr is a block's version on a measured page, sent back with its
// numbers so a reading of a block since changed is not kept.
func measureAttr(blk *store.Record, convo *conversation) string {
	if convo == nil || !convo.Measure || convo.FocusID != "" {
		return ""
	}
	return fmt.Sprintf(` data-measure-v="%s"`, chat.MeasureVersion(blk, convo.Panes))
}

// focusMeasured marks a block's own page: the block, drawn at its fullest,
// is the one measured there.
func focusMeasured(r *http.Request, blk *store.Record) string {
	if !measuring(r) {
		return ""
	}
	return fmt.Sprintf(` data-measure-block="%s" data-measure-v="%s"`, blk.ID, chat.MeasureVersion(blk, ""))
}

// measurePost keeps what a page measured of its blocks.
func (s *Server) measurePost(w http.ResponseWriter, r *http.Request) {
	if !measuring(r) {
		http.Error(w, "only the people who may change this page measure it", http.StatusForbidden)
		return
	}
	var body measureBody
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, errors.New("a measurement is numbers and ids only: "+jsonTrouble(err)))
		return
	}
	if err := body.check(); err != nil {
		writeError(w, err)
		return
	}
	if body.View == "tab" && !s.app.Chat.HasCanvas(body.Canvas) {
		writeError(w, fmt.Errorf("no tab %q", body.Canvas))
		return
	}
	s.app.Chat.Measured.Put(s.readings(body))
	w.WriteHeader(http.StatusNoContent)
}

func (b measureBody) check() error {
	if b.View != "tab" && b.View != "focus" {
		return errors.New("view is tab or focus")
	}
	if b.Canvas != "" && !measureID.MatchString(b.Canvas) {
		return errors.New("canvas is a tab's id")
	}
	if b.Viewport < 100 || b.Viewport > measureMost || b.Ratio < 0 || b.Ratio > 16 || b.Font < 0 || b.Font > 200 || b.Columns < 0 || b.Columns > 12 {
		return errors.New("the screen's numbers are out of range")
	}
	if len(b.Blocks) > measureBlocks {
		return fmt.Errorf("at most %d blocks", measureBlocks)
	}
	for _, m := range b.Blocks {
		if !measureID.MatchString(m.ID) || !measureVersion.MatchString(m.Version) {
			return errors.New("each block is its id and version")
		}
		for _, n := range []int{m.Width, m.Height, m.Top, m.Content, m.Box, m.Sideways, m.Cut, m.Designed, m.Past, m.Pane} {
			if n < 0 || n > measureMost {
				return fmt.Errorf("a block's numbers are between 0 and %d", measureMost)
			}
		}
	}
	return nil
}

// readings are the blocks sent that are on the page named and drawn as
// they are now; any other is left out.
func (s *Server) readings(b measureBody) []chat.Reading {
	var blocks []*store.Record
	panes := ""
	if b.View == "tab" {
		blocks = s.app.Chat.CanvasBlocks(b.Canvas)
		panes = chat.Panes(blocks)
	}
	ui := s.app.Workspace.Config.UI
	now := time.Now()
	var out []chat.Reading
	for _, m := range b.Blocks {
		blk, err := s.app.Store.Get(records.BlockType, m.ID)
		if err != nil || b.View == "tab" && !contains(blocks, m.ID) {
			continue
		}
		if chat.MeasureVersion(blk, panes) != m.Version {
			continue
		}
		out = append(out, chat.Reading{Block: m.ID, View: b.View, Device: chat.DeviceOf(b.Viewport), Version: m.Version, At: now,
			Viewport: b.Viewport, Ratio: b.Ratio, Font: b.Font, Columns: b.Columns, Text: ui.Text, Spacing: ui.Spacing,
			Width: m.Width, Height: m.Height, Top: m.Top, Content: m.Content, Box: m.Box, Sideways: m.Sideways,
			Cut: m.Cut, Designed: m.Designed, Past: m.Past, Pane: m.Pane})
	}
	return out
}

func contains(blocks []*store.Record, id string) bool {
	for _, b := range blocks {
		if b.ID == id {
			return true
		}
	}
	return false
}
