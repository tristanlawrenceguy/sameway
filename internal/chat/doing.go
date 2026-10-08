package chat

import (
	"encoding/json"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// What the assistant is doing, in words a person can watch go by: the
// status line says it while a turn runs ("Adding a chart of water",
// "Looking up your tasks"), so a person waiting hears what is happening
// rather than watching a spinner. The words come from the call itself,
// what it makes and of what, never from the tool's name.

// Target is the place on the canvas a call is about to change: a block
// already there (Block), or a new one in a region (Region, Span).
type Target struct {
	Block  string
	Region string
	Span   int
}

// callArgs is what the words and the aim are read from.
type callArgs struct {
	ID        string         `json:"id"`
	Component string         `json:"component"`
	Props     map[string]any `json:"props"`
	Fields    map[string]any `json:"fields"`
	Region    string         `json:"region"`
	Span      int            `json:"span"`
	Type      string         `json:"type"`
	Query     string         `json:"query"`
	Key       string         `json:"key"`
	Name      string         `json:"name"`
}

// thing names a block by what it is and what it shows: "list of tasks",
// "card called Shopping", "calendar", or "block" when nothing is known.
func thing(component string, props map[string]any) string {
	if component == "" {
		return "block"
	}
	noun := blocks.Noun(component)
	typeName, _ := props["type"].(string)
	if component == "record" && typeName != "" {
		return schema.Words(typeName)
	}
	if typeName != "" {
		return noun + " of " + schema.Plural(typeName)
	}
	for _, k := range []string{"title", "label", "heading", "name"} {
		if s, _ := props[k].(string); strings.TrimSpace(s) != "" {
			return noun + " called " + trim.Clip(strings.TrimSpace(s), 40)
		}
	}
	return noun
}

// describe says a tool call in a few words a person can watch go by, as
// its Op's Doing says it, or its name in words.
func describe(call llm.ToolCall) string {
	var args callArgs
	json.Unmarshal(call.Args, &args)
	if op := opNamed(call.Name); op.Doing != nil {
		return op.Doing(args)
	}
	return strings.ToUpper(call.Name[:1]) + strings.ReplaceAll(call.Name[1:], "_", " ")
}

// saying is an op's Doing when it is the same words every time.
func saying(words string) func(callArgs) string {
	return func(callArgs) string { return words }
}

// called is the title a call gives a record: " called Dentist", or "".
func (a callArgs) called() string {
	s, _ := a.Fields["title"].(string)
	return called(s)
}

func searching(a callArgs) string {
	if q := strings.TrimSpace(a.Query); q != "" {
		return "Searching for " + trim.Clip(q, 40)
	}
	return "Searching"
}

func reshaping(a callArgs) string {
	return "Changing the shape of" + or(an(schema.Words(a.Type)), " the content")
}

// doing is describe with what the store knows: a block changed or removed
// is named by what it is ("Changing the list of tasks"), and the call's
// aim says where on the canvas it lands.
func (s *Service) doing(call llm.ToolCall) (string, Target) {
	var args callArgs
	json.Unmarshal(call.Args, &args)
	label := describe(call)
	switch call.Name {
	case "add_component":
		return label, Target{Region: or(args.Region, "main"), Span: args.Span}
	case "update_component", "remove_component":
		if args.ID == "" || s.Store == nil {
			return label, Target{}
		}
		aim := Target{Block: args.ID}
		blk, err := s.Store.Get(records.BlockType, args.ID)
		if err != nil {
			return label, aim
		}
		component, _ := blk.Fields["component"].(string)
		props, _ := blk.Fields["props"].(map[string]any)
		verb := "Changing the "
		if call.Name == "remove_component" {
			verb = "Removing the "
		}
		return verb + thing(component, props), aim
	}
	return label, Target{}
}

// an is " a card" or " an image": a noun with its article, or nothing.
func an(noun string) string {
	if noun == "" {
		return ""
	}
	if strings.ContainsAny(noun[:1], "aeiou") {
		return " an " + noun
	}
	return " a " + noun
}

// describeChange says a change that landed in the log, in a few words:
// "Added a card".
func describeChange(c records.Change) string {
	verb := c.Action
	if verb == "" {
		verb = "changed"
	}
	return strings.ToUpper(verb[:1]) + verb[1:] + or(an(c.Component), " a block")
}

func or(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// called is " called Weekend", or "" for no name.
func called(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return " called " + trim.Clip(strings.TrimSpace(name), 40)
}
