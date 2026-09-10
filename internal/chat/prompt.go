package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

const basePrompt = `You are the assistant inside a Sameway workspace.

The person is looking at a canvas that fills the page. Everything on it is a block you control with tools, including the chat block this conversation is inside.

Each block has a shape and a look, set independently of its props:
- span: width in columns of twelve. 12 is full width, 6 half, 4 a third. Narrow screens ignore it.
- frame: card for a block with its own surface, bare to sit flush on the page with no border. Use bare for headings and short text so the page does not become a wall of boxes.
- tone: none, accent, success, warning, danger, or info. Tints the surface. Use it sparingly, to mark one thing that matters.
- position: sort order, lower first.
- region: main is the body of the page; side is a collapsible pane beside it, for things the person glances at rather than works in, like a calendar or what is due next.

How to work:
- When the person asks for something, build it on the canvas with the tools, then reply with one or two short sentences saying what you did. Do not paste HTML or props into the reply.
- Use only components from the catalogue below, with props that match each schema exactly. If a tool returns an error, fix the props and call the tool again.
- Lay things out deliberately. A heading that introduces a section wants span 12 and frame bare; cards and tables sit well at 6 or 7; small items at 4. Vary the widths so the page looks composed rather than stacked.
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
	if s.ExtraPrompt != "" {
		b.WriteString("\n\nWorkspace instructions:\n")
		b.WriteString(s.ExtraPrompt)
	}
	b.WriteString("\n\nComponent catalogue (name: description, then props schema):\n")
	for _, c := range s.Registry.Components() {
		fmt.Fprintf(&b, "\n%s: %s\n%s\n", c.Manifest.Name, c.Manifest.Description, compactJSON(c.Manifest.Props))
	}
	b.WriteString("\nCurrent canvas, top to bottom (id, component, span, frame, tone, props):\n")
	blocks, err := s.Store.List(BlockType, store.ListOptions{OrderBy: "position"})
	if err != nil || len(blocks) == 0 {
		b.WriteString("(empty)\n")
		return b.String()
	}
	for _, blk := range blocks {
		props, _ := json.Marshal(blk.Fields["props"])
		span := int64(6)
		if v, ok := blk.Fields["span"].(int64); ok {
			span = v
		}
		frame, _ := blk.Fields["frame"].(string)
		if frame == "" {
			frame = "card"
		}
		tone, _ := blk.Fields["tone"].(string)
		if tone == "" {
			tone = "none"
		}
		region := "main"
		if v, ok := blk.Fields["region"].(string); ok && v != "" {
			region = v
		}
		fmt.Fprintf(&b, "%s %s region=%s span=%d frame=%s tone=%s %s\n", blk.ID, blk.Fields["component"], region, span, frame, tone, props)
	}
	return b.String()
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
