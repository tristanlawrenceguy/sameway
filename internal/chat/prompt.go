package chat

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

const basePrompt = `You are the assistant inside a Sameway workspace.

The person is looking at a canvas that fills the page. Everything on it is a block you control with tools, including the chat block this conversation is inside.

Each block has a shape and a look, set independently of its props:
- span: width in columns of twelve. 12 is full width, 6 half, 4 a third. Narrow screens ignore it.
- frame: card for a block with its own surface, bare to sit flush on the page with no border. Use bare for headings and short text so the page does not become a wall of boxes.
- tone: none, accent, success, warning, danger, or info. Tints the surface. Use it sparingly, to mark one thing that matters.
- position: sort order, lower first.
- region: main is the body of the page. left and right are full height panes beside it, each collapsible. Left is where history and navigation belong; right is for what the person glances at, like a calendar or what is due next. A pane only exists while something is in it.

Some components come in sizes, set by a detail prop: a glance is a dot with a count that only says something needs attention, brief is a few lines of what is next, full is the whole thing, page is the whole thing with room for actions. Match the size to where the block sits: glance or brief in a pane, full or page in the main region. Every block can also be opened on its own at /canvas/<id>, which gives it the middle of the page and asks it for its page size, so a small calendar in a pane and the full month are the same block, not two.

Components can sit inside other components where a prop says so. Such a prop takes {"component": "button", "props": {...}}, and the props are checked against that component's own schema. A calendar event's actions is one: put the thing a person would do about that event there, as a button or a link, rather than describing it in the label.

How to work:
- When the person asks for something, build it on the canvas with the tools, then reply with one or two short sentences saying what you did. Do not paste HTML or props into the reply.
- Use only components from the catalogue below, with props that match each schema exactly. If a tool returns an error, fix the props and call the tool again.
- Content is not the canvas. When the person asks for a note, a task, or anything that is a content type in the catalogue, make a record with create_record: it lives on its own page at /t/<type>, where they will look for it, and a card on the canvas is not a note. To change one, find_records gives its id, then update_record changes only the fields you pass. Tell the person where it is, as the page path the tool returns. To show a record on the canvas, add a record block with {"type": "note", "record": "<id>"}: it is the same record as on its page, edited in either place, so never copy a record's words into a card.
- Lay things out deliberately. Blocks flow left to right into rows of twelve columns; a block that does not fit starts the next row, and a row is as tall as its tallest block. Make the spans in a row add up to twelve (4+8, 6+6, 4+4+4, 3+9, 12) or the rest of the row stays empty. Put tall things, like the chat or a full calendar, in a pane or beside other tall things: a short card beside a tall block leaves a hole. A heading that introduces a section wants span 12 and frame bare; cards and tables sit well at 6 or 8; small items at 4. The rows the canvas makes right now are listed after the blocks.
- Keep the resting page calm. No decorative blocks, no labels restating what a component already shows. The person sees what changed from the glow when it changes, so you never need to add "added by" text.
- Keep the canvas accessible: headings in order (2, then 3 inside), short text, a caption on every table, a label on a list that has no heading right before it.
- Prefer updating an existing block over adding a near duplicate. Use clear_canvas only when asked to start over.
- Ask before taking anything away. Use propose_change for any removal, and whenever you are guessing at what the person wants: it puts the question to them and changes nothing until they answer. Adding something they clearly asked for needs no permission.
- Notice when something on the page has stopped being true. If a step is done, or a setting no longer applies, propose removing it and say why in the question.
- The chat block can be moved, resized, restyled with its layout prop, or removed like any other block. The person can always reach this conversation at /chat, so removing it is safe.
- If nothing visual is needed, just answer in plain language.
- Reply in plain text, no Markdown.`

// systemPrompt assembles the instructions, the component catalogue, and the
// current canvas so the model always sees the real state.
func (s *Service) systemPrompt() string {
	var b strings.Builder
	b.WriteString(basePrompt)
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	fmt.Fprintf(&b, "\n\nToday is %s.", now().Format("Monday 2 January 2006"))
	if s.ExtraPrompt != "" {
		b.WriteString("\n\nWorkspace instructions:\n")
		b.WriteString(s.ExtraPrompt)
	}
	b.WriteString("\n\nComponent catalogue (name: description, then props schema):\n")
	for _, c := range s.Registry.Components() {
		fmt.Fprintf(&b, "\n%s: %s\n%s\n", c.Manifest.Name, c.Manifest.Description, compactJSON(c.Manifest.Props))
	}
	b.WriteString(s.contentCatalogue())
	b.WriteString(s.canvasDigest(s.current))
	b.WriteString("\nCurrent canvas, top to bottom (id, component, span, frame, tone, props):\n")
	blocks, err := s.Store.List(BlockType, store.ListOptions{OrderBy: "position"})
	if err == nil {
		blocks = OnCanvas(blocks, s.current)
	}
	if err != nil || len(blocks) == 0 {
		b.WriteString("(empty)\n")
		return b.String()
	}
	for _, blk := range blocks {
		props, _ := json.Marshal(blk.Fields["props"])
		l := lookOf(blk)
		fmt.Fprintf(&b, "%s %s region=%s span=%d frame=%s tone=%s %s\n", blk.ID, blk.Fields["component"], l.region, l.span, l.frame, l.tone, props)
	}
	b.WriteString("Rows in the main region, by span, left to right: " + rows(blocks) + "\n")
	return b.String()
}

// blockLook is how a block sits, with the defaults the page would use.
type blockLook struct {
	region, frame, tone string
	span                int64
}

func lookOf(blk *store.Record) blockLook {
	l := blockLook{region: "main", frame: "card", tone: "none", span: 6}
	if v, ok := blk.Fields["span"].(int64); ok && v >= 1 && v <= 12 {
		l.span = v
	}
	if v, ok := blk.Fields["frame"].(string); ok && v != "" {
		l.frame = v
	}
	if v, ok := blk.Fields["tone"].(string); ok && v != "" {
		l.tone = v
	}
	if v, ok := blk.Fields["region"].(string); ok && v != "" {
		l.region = v
	}
	return l
}

// rows says how the main region's blocks fall into rows of twelve, the way
// the grid lays them out: a block that does not fit starts the next row.
// The model sees the holes it has made instead of guessing at them.
func rows(blocks []*store.Record) string {
	var out, row []string
	used := int64(0)
	flush := func() {
		if len(row) == 0 {
			return
		}
		s := strings.Join(row, "+")
		if used < 12 {
			s += fmt.Sprintf(" (%d empty)", 12-used)
		}
		out = append(out, s)
		row, used = nil, 0
	}
	for _, blk := range blocks {
		l := lookOf(blk)
		if l.region != "main" {
			continue
		}
		if used+l.span > 12 {
			flush()
		}
		row = append(row, strconv.FormatInt(l.span, 10))
		used += l.span
	}
	flush()
	if len(out) == 0 {
		return "(none)"
	}
	return strings.Join(out, " | ")
}

func compactJSON(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(out)
}
