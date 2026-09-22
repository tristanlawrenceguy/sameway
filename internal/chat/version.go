package chat

import (
	"context"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// Keeping sameway current is something a person asks for in words, like
// every other setting: "is there a new version?", "update". The program
// itself does the looking while the server runs, in whichever way
// update.mode says; this is the same thing on request, so a person in
// manual mode never has to leave the conversation to get a new version.

var updateTool = llm.Tool{
	Name:        "update_sameway",
	Description: "Look for a new version of the sameway program itself. With install true, install it when there is one; without, only say whether there is. Say what came back word for word: a new version runs from the next start, so the person has to restart it. Not for content and not for the canvas.",
	Schema: map[string]any{"type": "object", "properties": map[string]any{
		"install": map[string]any{"type": "boolean", "description": "true when the person asked to update or upgrade; false when they only asked whether a new version is out."},
	}, "additionalProperties": false},
}

// updateSameway looks for a new version, and installs it when the person
// asked for that. The sentence it answers with is the outcome's own, so
// the assistant, the terminal and the activity log all say the same thing.
func (s *Service) updateSameway(install bool) toolResult {
	if s.Update == nil {
		return fail("this sameway cannot update itself")
	}
	out, err := s.Update(context.Background(), install)
	if err != nil {
		return fail("%v", err)
	}
	if out.Installed {
		return toolResult{text: out.Says, change: &Change{Action: "updated to", Component: "sameway " + out.Latest}}
	}
	if out.Newer {
		return toolResult{text: out.Says + update.AskFor}
	}
	return toolResult{text: out.Says}
}
