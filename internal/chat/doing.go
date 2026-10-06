package chat

import (
	"encoding/json"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
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

// nouns are what people call a component when it is not its own name.
var nouns = map[string]string{"collection": "list", "text": "piece of text", "record": "record", "filters": "set of filters"}

// thing names a block by what it is and what it shows: "list of tasks",
// "card called Shopping", "calendar", or "block" when nothing is known.
func thing(component string, props map[string]any) string {
	if component == "" {
		return "block"
	}
	noun := nouns[component]
	if noun == "" {
		noun = schema.Words(component)
	}
	typeName, _ := props["type"].(string)
	if component == "record" && typeName != "" {
		return schema.Words(typeName)
	}
	if typeName != "" {
		return noun + " of " + plural(typeName)
	}
	for _, k := range []string{"title", "label", "heading", "name"} {
		if s, _ := props[k].(string); strings.TrimSpace(s) != "" {
			return noun + " called " + clip(strings.TrimSpace(s), 40)
		}
	}
	return noun
}

// describe says a tool call in a few words a person can watch go by.
func describe(call llm.ToolCall) string {
	var args callArgs
	json.Unmarshal(call.Args, &args)
	a := an
	title := ""
	if s, _ := args.Fields["title"].(string); strings.TrimSpace(s) != "" {
		title = " called " + clip(strings.TrimSpace(s), 40)
	}
	switch call.Name {
	case "add_component":
		return "Adding" + an(thing(args.Component, args.Props))
	case "update_component":
		return "Changing a block"
	case "remove_component":
		return "Removing a block"
	case "create_record":
		return "Adding" + or(a(schema.Words(args.Type)), " a record") + title
	case "import_records":
		return "Importing " + or(plural(args.Type), "records") + " from a file"
	case "organise_writing":
		return "Organising the writing"
	case "suggest_edits":
		return "Suggesting changes"
	case "record_meeting":
		return "Setting the meeting to ask to be recorded"
	case "write_up_meeting":
		return "Writing up the meeting"
	case "update_record":
		return "Updating" + or(a(schema.Words(args.Type)), " a record")
	case "delete_record":
		return "Deleting" + or(a(schema.Words(args.Type)), " a record")
	case "find_records":
		if args.Type == "" {
			return "Looking up your records"
		}
		return "Looking up your " + plural(args.Type)
	case "get_record":
		return "Reading" + or(a(schema.Words(args.Type)), " a record")
	case "look_at_page":
		return "Looking at the page"
	case "search":
		if q := strings.TrimSpace(args.Query); q != "" {
			return "Searching for " + clip(q, 40)
		}
		return "Searching"
	case "propose_change":
		return "Asking you about a change"
	case "set_setting":
		return "Changing " + or(args.Key, "a setting")
	case "add_type":
		return "Changing the shape of" + or(a(schema.Words(args.Type)), " the content")
	case "run_action":
		return "Running an action"
	case "add_arrangement", "arrange_canvas":
		return "Arranging the page"
	case "clear_canvas":
		return "Clearing the page"
	case "undo_change":
		return "Undoing a change"
	case "create_canvas":
		return "Adding a tab" + or(called(args.Name), "")
	case "remove_canvas":
		return "Removing a tab"
	case "add_field", "change_field":
		return "Changing the shape of" + or(a(schema.Words(args.Type)), " the content")
	case "clear_conversation":
		return "Clearing the conversation"
	case "add_workspace":
		return "Making a workspace"
	case "open_workspace":
		return "Opening a workspace"
	case "restore_workspace":
		return "Bringing a workspace back"
	case "let_in", "take_agent_away":
		return "Changing who can use this"
	case "update_sameway":
		return "Updating Sameway"
	case "write_down":
		return "Writing down a recording"
	}
	return strings.ToUpper(call.Name[:1]) + strings.ReplaceAll(call.Name[1:], "_", " ")
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
		blk, err := s.Store.Get(BlockType, args.ID)
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
func describeChange(c Change) string {
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

func plural(s string) string {
	return schema.Plural(s)
}

// called is " called Weekend", or "" for no name.
func called(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return " called " + clip(strings.TrimSpace(name), 40)
}
