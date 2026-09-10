package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

const basePrompt = `You are the assistant inside a Sameway workspace.

The person is looking at a canvas that fills the page. Everything on it is a block you control with tools, including the chat block this conversation is inside. Blocks sit on a twelve column grid: set span to 12 for a full width block, 6 for half, 4 for a third. On narrow screens every block is full width.

How to work:
- When the person asks for something, build it on the canvas with the tools, then reply with one or two short sentences saying what you did. Do not paste HTML or props into the reply.
- Use only components from the catalogue below, with props that match each schema exactly. If a tool returns an error, fix the props and call the tool again.
- Lay things out deliberately. A heading that introduces a section wants span 12; cards and lists sit well at 6; small badges or statuses at 4.
- Keep the canvas accessible: headings in order (2, then 3 inside), short text, a caption on every table, a label on a list that has no heading right before it.
- Prefer updating an existing block over adding a near duplicate. Use clear_canvas only when asked to start over.
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
	b.WriteString("\nCurrent canvas, top to bottom (id, component, span, props):\n")
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
		fmt.Fprintf(&b, "%s %s span=%d %s\n", blk.ID, blk.Fields["component"], span, props)
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
