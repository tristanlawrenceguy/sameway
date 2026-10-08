package chat

import (
	"context"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
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
	Change *records.Change
	// Aim is where a call is about to change the canvas, so the page can
	// mark the place while the call runs; zero for a call that does not.
	Aim Target
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
	resp, err := s.completeOnce(ctx, req, on)
	// A model that cannot see answers again, told a picture was there.
	if err != nil && cannotSee(err) {
		if plain, had := withoutPictures(req); had {
			sight.Store(s.Provider.Name(), false)
			return s.completeOnce(ctx, plain, on)
		}
	}
	if err == nil && hasPictures(req) {
		sight.Store(s.Provider.Name(), true)
	}
	return resp, err
}

func (s *Service) completeOnce(ctx context.Context, req llm.Request, on func(Event)) (*llm.Response, error) {
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
