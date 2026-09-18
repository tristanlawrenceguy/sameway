package chat

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A turn takes as long as the model and its tools take. A page that shows
// the turn as it goes gets each thing that happens, in order: the words
// as they come, each tool as it starts, each change as it lands, and the
// end. Without a listener the turn runs exactly as before.

// Event is one thing that happened during a turn.
type Event struct {
	// Kind is said (the person's message is recorded), delta (a piece of
	// the reply's text), text (a whole round's words, from a model that
	// does not stream), tool (a call starts), change (a change landed),
	// done (the reply is recorded) or error (the turn failed; a message
	// says so).
	Kind string
	Text string
	// Tool is the call's name and Label the call in words: "Adding a
	// calendar". Early marks a call the model has only begun: its name is
	// known, its arguments are still coming, so the label is broad. The
	// same call comes again, in full, when it runs. For a change, the
	// change itself.
	Tool   string
	Label  string
	Early  bool
	Change *Change
	// ID is the record made: the person's message for said, the reply for
	// done, the error message for error.
	ID string
}

// SendFile is a turn with nobody watching.
func (s *Service) SendFile(ctx context.Context, canvas, text, fileID string) (*store.Record, error) {
	return s.SendLive(ctx, canvas, text, fileID, nil)
}

// SendLive is a turn told as it happens to on, which may be nil.
func (s *Service) SendLive(ctx context.Context, canvas, text, fileID string, on func(Event)) (*store.Record, error) {
	rec, err := s.sendTurn(ctx, canvas, text, fileID, on)
	if on != nil && rec != nil {
		if rec.Fields["role"] == "error" {
			content, _ := rec.Fields["content"].(string)
			on(Event{Kind: "error", ID: rec.ID, Text: content})
		} else {
			on(Event{Kind: "done", ID: rec.ID})
		}
	} else if on != nil && err != nil {
		on(Event{Kind: "error", Text: err.Error()})
	}
	return rec, err
}

// complete asks the model for its next round: as a stream when someone is
// listening and the model can, so the words show as they come.
func (s *Service) complete(ctx context.Context, req llm.Request, on func(Event)) (*llm.Response, error) {
	if st, ok := s.Provider.(llm.Streamer); ok && on != nil {
		return st.Stream(ctx, req, func(d llm.Delta) {
			if d.Text != "" {
				on(Event{Kind: "delta", Text: d.Text})
			}
			if d.Call != "" {
				on(Event{Kind: "tool", Tool: d.Call, Label: describe(llm.ToolCall{Name: d.Call}), Early: true})
			}
		})
	}
	resp, err := s.Provider.Complete(ctx, req)
	if err == nil && on != nil && strings.TrimSpace(resp.Text) != "" {
		on(Event{Kind: "text", Text: resp.Text})
	}
	return resp, err
}

// describe says a tool call in a few words a person can watch go by.
func describe(call llm.ToolCall) string {
	var args struct {
		Component string `json:"component"`
		Type      string `json:"type"`
		Query     string `json:"query"`
		Key       string `json:"key"`
	}
	json.Unmarshal(call.Args, &args)
	a := an
	switch call.Name {
	case "add_component":
		return "Adding" + or(a(args.Component), " a block")
	case "update_component":
		return "Changing a block"
	case "remove_component":
		return "Removing a block"
	case "create_record":
		return "Creating" + or(a(args.Type), " a record")
	case "update_record":
		return "Updating" + or(a(args.Type), " a record")
	case "delete_record":
		return "Deleting" + or(a(args.Type), " a record")
	case "find_records":
		return "Looking through" + or(" "+plural(args.Type), " the records")
	case "get_record":
		return "Reading" + or(a(args.Type), " a record")
	case "search":
		return "Searching for " + strings.TrimSpace(args.Query)
	case "propose_change":
		return "Proposing a change"
	case "set_setting":
		return "Changing " + or(args.Key, "a setting")
	case "add_field", "add_type":
		return "Changing the shape of" + or(a(args.Type), " the content")
	case "run_action":
		return "Running an action"
	case "add_arrangement":
		return "Arranging the page"
	case "clear_canvas":
		return "Clearing the page"
	case "undo_change":
		return "Undoing a change"
	}
	return strings.ToUpper(call.Name[:1]) + strings.ReplaceAll(call.Name[1:], "_", " ")
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
	if s == "" {
		return ""
	}
	if strings.HasSuffix(s, "s") {
		return s
	}
	return s + "s"
}
