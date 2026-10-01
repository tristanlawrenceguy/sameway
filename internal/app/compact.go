package app

import (
	"encoding/json"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// CompactComponent is one component as someone writing a block needs it:
// what it is for, the props they may give (not the ones the server fills
// in), and one example to start from. Over MCP, the whole manifest of a
// component asked for by name was more than the client would take, and the
// agent guessed at props (x and y, a component called board) instead.
type CompactComponent struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Use         *render.Use     `json:"use,omitempty"`
	Props       json.RawMessage `json:"props"`
	Example     map[string]any  `json:"example,omitempty"`
	Add         string          `json:"add"`
}

// Compact is the component cut down to what writing it takes.
func (c DescribedComponent) Compact() CompactComponent {
	out := CompactComponent{Name: c.Name, Description: c.Description, Use: c.Use, Props: chat.ForModel(c.Props),
		Add: `POST /api/block with {"component": "` + c.Name + `", "props": <props like the example>}; the whole manifest, with its accessibility contract, is at /api/describe/components/` + c.Name + `?full=1`}
	if c.PageOnly {
		out.Add = "Not a block: Sameway shows it itself on the page it belongs to, so it is met there, never added."
	}
	if len(c.Examples) > 0 {
		filled := serverFilledProps(c.Props)
		out.Example = map[string]any{}
		for k, v := range c.Examples[0].Props {
			if !filled[k] {
				out.Example[k] = v
			}
		}
	}
	return out
}

// ComponentSummary is one component in a line, for the list of them.
type ComponentSummary struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Use         *render.Use `json:"use,omitempty"`
}

// serverFilledProps names the top-level props the server fills in, so an
// example does not teach a writer to send them.
func serverFilledProps(raw json.RawMessage) map[string]bool {
	var s struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	json.Unmarshal(raw, &s)
	out := map[string]bool{}
	for k, p := range s.Properties {
		if chat.ServerFilled(p.Description) {
			out[k] = true
		}
	}
	return out
}
