package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Undo is the activity log run backwards, and the log is the records'
// (internal/records/undo.go). Here is the assistant's side of it: the
// tool, and the changes it is told it could take back.

var undoOp = Op{Title: "Undo a change",
	Access: ForOwner,
	Core:   true,
	Doing:  saying("Undoing a change"),
	Run:    func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.undo(a.ID) },
	Tool: llm.Tool{
		Name:        "undo_change",
		Description: "Reverse one change from the activity log, yours or the person's: an added thing is removed, a removed thing is put back with everything it had, an update goes back to what it was. Undoing an undo puts it back again. Without an id, the newest change that can still be undone.",
		Schema: map[string]any{"type": "object", "properties": map[string]any{
			"id": map[string]any{"type": "string", "description": "The activity entry's id, from the recent changes listed in the prompt."},
		}, "additionalProperties": false},
	}}

// undo is undo_change: the reversal, ready to be recorded with the turn.
func (s *Service) undo(id string) toolResult {
	said, c, err := s.Undo(id)
	if err != nil {
		return fail("%v", err)
	}
	return toolResult{text: said, change: &c}
}

// undoDigest lists the changes the model could reverse, newest first, so
// "undo that" has an id to point at.
func (s *Service) undoDigest() string {
	var lines []string
	for _, a := range s.Recent(20) {
		if s.Undoable(a) {
			lines = append(lines, fmt.Sprintf("%s: %s", a.ID, a.Fields["summary"]))
		}
		if len(lines) == 5 {
			break
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "\nRecent changes that can be undone, newest first (id: what happened); undo_change takes the id:\n" + strings.Join(lines, "\n") + "\n"
}

// removeBlock takes one block off the canvas and logs what it was.
func (s *Service) removeBlock(id string) toolResult {
	c, err := records.RemoveBlock(s.Store, id)
	if err != nil {
		return fail("%v", err)
	}
	return toolResult{text: "removed block " + id, change: &c}
}
