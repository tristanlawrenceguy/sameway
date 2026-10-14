package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Taking back what a reply did. Asked to sort a week out, a model made
// eight changes; told "undo all that", it could name only the five newest,
// the most the prompt listed, and undid them one call at a time, leaving
// three it never saw. The prompt now names every change of the last reply,
// and undo_change takes them all in one call, all or none, as the page's
// Undo all does.

// undoSome is undo_change: the one id, the newest when none, or each of
// ids in the order given, all or none (records.UndoEach): one that cannot
// be undone leaves every one as it was, and is named.
func (s *Service) undoSome(id string, ids []string) toolResult {
	if len(ids) == 0 {
		return s.undo(id)
	}
	said, changes, err := s.UndoEach(ids)
	if err != nil {
		return fail("%v. Call undo_change again without that one to take back the others.", err)
	}
	return toolResult{text: fmt.Sprintf("undid all %d: %s", len(ids), strings.Join(said, "; ")), changes: changes}
}

// lastReplyUndos is the log entries of the newest reply's changes, newest
// first, as undo_change's ids take them.
func (s *Service) lastReplyUndos() []string {
	msgs, err := s.Store.List(records.MessageType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 10})
	if err != nil {
		return nil
	}
	for _, m := range msgs {
		if m.Fields["role"] != "assistant" {
			continue
		}
		list, _ := m.Fields["changes"].([]any)
		var ids []string
		for i := len(list) - 1; i >= 0; i-- {
			if c, ok := list[i].(map[string]any); ok {
				if id, _ := c["activity"].(string); id != "" {
					ids = append(ids, id)
				}
			}
		}
		return ids
	}
	return nil
}

// lastReplySaid is the prompt's line for them, when there are two or more.
func (s *Service) lastReplySaid() string {
	ids := s.lastReplyUndos()
	if len(ids) < 2 {
		return ""
	}
	return fmt.Sprintf("Your last reply made %d changes; undo_change with ids %s takes them all back, newest first.\n", len(ids), strings.Join(ids, ", "))
}
