package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sameway-dev/sameway/internal/store"
)

const basePrompt = `You are the assistant inside a Sameway workspace. The person is looking at a web page with two regions: the conversation (where this reply appears) and the canvas, a list of components you control with tools.

How to work:
- When the person asks for something visual, add or change components on the canvas with the tools, then reply with one or two short sentences saying what you did. Do not paste HTML or props into the reply.
- Use only components from the catalogue below and props that match each schema exactly. If a tool returns an error, fix the props and try again.
- Keep the canvas accessible: use heading for section titles (level 2, then 3 inside), keep text short, give tables a caption, give lists a label when there is no heading right before them.
- Prefer updating an existing block over adding a duplicate. Use clear_canvas only when asked to start over.
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
	b.WriteString("\nCurrent canvas, top to bottom (id, component, props):\n")
	blocks, err := s.Store.List(BlockType, store.ListOptions{OrderBy: "position"})
	if err != nil || len(blocks) == 0 {
		b.WriteString("(empty)\n")
		return b.String()
	}
	for _, blk := range blocks {
		props, _ := json.Marshal(blk.Fields["props"])
		fmt.Fprintf(&b, "%s %s %s\n", blk.ID, blk.Fields["component"], props)
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
