package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// arrange_canvas lays a whole tab out in one change: every block in the
// order it is to be read, with its width and place. The owner: "AI just
// adds though and doesn't think about how it fits in and if other pieces
// should move or change shape." Moving five blocks one update at a time
// was five changes, five Undos and five chances to stop half way; this is
// one entry in the log and one Undo, and it is refused whole when it would
// lose a block or break the outline.

var arrangeOp = Op{Title: "Lay a tab out", Traits: Traits{Idempotent: true},
	Core:  true,
	Doing: saying("Arranging the page"),
	Run:   func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.arrangeCall(call.Args) },
	Tool: llm.Tool{
		Name:        "arrange_canvas",
		Description: "Lay out a whole tab in one change: list every block on it (from the canvas listing) in the order it should be read, top to bottom, each with the width and place it should have. The list order becomes the order on the page; what an item leaves out stays as it is. Use it after adding something, to move and reshape what was already there so the page reads well: what matters most first, related things together, rows of twelve filled, headings in order. Blocks in the header and footer may be left out. One Undo takes the whole arrangement back. Refused, with nothing changed, when a block is missing or listed twice, or a heading would skip a level.",
		Schema: obj(map[string]any{
			"blocks": map[string]any{"type": "array", "description": "Every block on the tab, in reading order.", "items": obj(map[string]any{
				"id":     map[string]any{"type": "string", "description": "Block id from the canvas listing."},
				"span":   map[string]any{"type": "integer", "description": "Width in columns of twelve; the spans in a row should add up to 12."},
				"region": map[string]any{"type": "string", "enum": []string{"main", "left", "right", "header", "footer"}},
				"frame":  map[string]any{"type": "string", "enum": []string{"card", "bare"}},
				"size":   map[string]any{"type": "string", "enum": []string{"full", "compact", "icon"}},
			}, "id")},
			"canvas": map[string]any{"type": "string", "description": "The tab, as a canvas id; empty string is Home. Defaults to the tab the person is looking at."},
		}, "blocks"),
	}}

func (s *Service) arrangeCall(raw json.RawMessage) toolResult {
	var args struct {
		Blocks []arrangeItem `json:"blocks"`
		Canvas *string       `json:"canvas"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return fail("%s", ArgsTrouble(err))
	}
	canvas := s.current
	if args.Canvas != nil {
		canvas = *args.Canvas
	}
	return s.arrange(canvas, args.Blocks)
}

// Arrange is arrange_canvas for a caller that is not the conversation,
// such as POST /api/arrange: the same checks, the same one entry.
func (s *Service) Arrange(canvas string, blocks json.RawMessage) (string, bool) {
	r := s.run(llm.ToolCall{ID: "call", Name: arrangeOp.Name, Args: mustJSON(map[string]any{"canvas": canvas, "blocks": blocks})})
	return r.text, r.isErr
}

func mustJSON(v any) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}

func (s *Service) arrange(canvas string, items []arrangeItem) toolResult {
	if !s.HasCanvas(canvas) {
		return fail("no canvas with id %q; the tabs and their ids are listed in the prompt, and \"\" is Home", canvas)
	}
	before := s.canvasBlocks(canvas)
	byID := map[string]*store.Record{}
	for _, b := range before {
		byID[b.ID] = b
	}
	listed := map[string]bool{}
	var problems []string
	for _, it := range items {
		switch {
		case byID[it.ID] == nil:
			problems = append(problems, fmt.Sprintf("%s is not a block on this tab", it.ID))
		case listed[it.ID]:
			problems = append(problems, fmt.Sprintf("%s is listed twice", it.ID))
		}
		listed[it.ID] = true
	}
	for _, b := range before {
		if r := lookOf(b).region; !listed[b.ID] && r != "header" && r != "footer" {
			problems = append(problems, fmt.Sprintf("%s (%s %s) is missing; list every block, even one that stays where it is", b.ID, placedOf(b).name(), placedOf(b).comp))
		}
	}
	if len(problems) > 0 {
		return fail("nothing was arranged: %s", strings.Join(problems, "; "))
	}
	// Work out every block's new fields first, so nothing is written
	// until the whole arrangement is known to be sound.
	next := map[string]map[string]any{}
	for i, it := range items {
		fields := s.marked(map[string]any{"position": i})
		if _, err := (look{Span: it.Span, Frame: it.Frame, Region: it.Region, Size: it.Size}).apply(fields); err != nil {
			return fail("nothing was arranged: %s: %v", it.ID, err)
		}
		next[it.ID] = fields
	}
	var after []*store.Record
	for _, b := range before {
		after = append(after, withFields(b, next[b.ID]))
	}
	byPosition(after)
	if why := newSkip(before, after); why != "" {
		return fail("nothing was arranged: %s", why)
	}
	var batch []records.BatchItem
	for _, it := range items {
		was := byID[it.ID]
		if !moves(was, next[it.ID]) {
			continue
		}
		if _, err := s.Store.Update(records.BlockType, it.ID, s.fields(records.BlockType, next[it.ID])); err != nil {
			s.putBack(batch)
			return fail("nothing was arranged: could not move %s: %v", it.ID, err)
		}
		batch = append(batch, records.BatchItem{Type: records.BlockType, ID: it.ID, Before: was.Fields})
	}
	if len(batch) == 0 {
		return toolResult{text: "the tab is already arranged that way; nothing changed"}
	}
	name, href := "Home", "/"
	for _, c := range s.Canvases() {
		if c.ID == canvas && canvas != "" {
			name, href = c.Name, records.CanvasPath(canvas)
		}
	}
	c := records.Change{Action: "arranged", Detail: fmt.Sprintf("%s, %d blocks", name, len(batch)), Href: href, Ops: records.OpsOf(s.Store, batch)}
	return toolResult{text: fmt.Sprintf("arranged %s: %d blocks moved or reshaped, in one change that one Undo takes back", name, len(batch)), change: &c}
}

// moves says whether new layout fields would change a block; who last
// touched it is not a change of its own.
func moves(b *store.Record, fields map[string]any) bool {
	for k, v := range fields {
		if k != "actor" && k != "agent" && !records.Same(map[string]any{k: v}, map[string]any{k: b.Fields[k]}) {
			return true
		}
	}
	return false
}

// putBack undoes the part of an arrangement already written, when a later
// block could not be.
func (s *Service) putBack(done []records.BatchItem) {
	for _, d := range done {
		s.Store.Update(records.BlockType, d.ID, d.Before)
	}
}

// withFields is a copy of a block with some fields changed, to see a
// change before making it.
func withFields(b *store.Record, fields map[string]any) *store.Record {
	c := *b
	c.Fields = map[string]any{}
	for k, v := range b.Fields {
		c.Fields[k] = v
	}
	for k, v := range fields {
		if n, ok := v.(int); ok {
			v = int64(n)
		}
		c.Fields[k] = v
	}
	return &c
}

// newSkip says how a change would make a heading skip a level that did
// not before, "" when it would not. A skipped level is the one layout
// fault refused rather than pointed out: it breaks moving by heading for a
// screen reader user (WCAG 1.3.1), and which fix is right (a lower level,
// or a section heading above) is the writer's to choose, not Sameway's.
func newSkip(before, after []*store.Record) string {
	main := func(bs []*store.Record) []placed {
		var all []placed
		for _, b := range bs {
			all = append(all, placedOf(b))
		}
		return region(all, "main")
	}
	was := headingSkips(main(before))
	for id, why := range headingSkips(main(after)) {
		if _, ok := was[id]; !ok {
			return why
		}
	}
	return ""
}

// skipIfWritten is newSkip for one block added or changed on a tab.
func (s *Service) skipIfWritten(canvas string, blk *store.Record) string {
	before := s.canvasBlocks(canvas)
	var after []*store.Record
	for _, b := range before {
		if b.ID != blk.ID {
			after = append(after, b)
		}
	}
	if on, _ := blk.Fields["canvas"].(string); on == canvas {
		after = append(after, blk)
	}
	byPosition(after)
	return newSkip(before, after)
}

// layoutAfter is the line every write of a block ends with: how the tab
// it is on reads now.
func (s *Service) layoutAfter(name string, r toolResult) toolResult {
	switch name {
	case "add_component", "update_component", "remove_component", "arrange_canvas", "add_arrangement":
	default:
		return r
	}
	c := r.change
	if c == nil && len(r.changes) > 0 {
		c = &r.changes[0]
	}
	if r.isErr || c == nil {
		return r
	}
	canvas, found := "", false
	if blk, err := s.Store.Get(records.BlockType, c.ID); err == nil && c.ID != "" {
		canvas, found = blk.Fields["canvas"].(string)
	}
	if was := c.Was(); !found && was != nil {
		canvas, found = was["canvas"].(string)
	}
	if !found && strings.HasPrefix(c.Href, "/c/") {
		canvas = strings.TrimPrefix(c.Href, "/c/")
	}
	r.text += "\n" + s.LayoutNow(canvas)
	return r
}
