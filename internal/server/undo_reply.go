package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Taking back a whole reply. Asked to sort a week out, the assistant made
// eight changes, and the receipt under its reply offered eight Undos, to
// press newest first or have the older ones refused. A reply's changes are
// one thing the person asked for; when each can still be undone, one press
// takes them all back.

// replyUndos is the log entries of a reply's changes, oldest first, when
// there are two or more and each can still be undone; nil otherwise.
func replyUndos(receipt []any) []string {
	if len(receipt) < 2 {
		return nil
	}
	var ids []string
	for _, item := range receipt {
		c, _ := item.(map[string]any)
		href, _ := c["undo"].(string)
		id, prefixed := strings.CutPrefix(href, "/activity/")
		id, suffixed := strings.CutSuffix(id, "/undo")
		if !prefixed || !suffixed || id == "" {
			return nil
		}
		ids = append(ids, id)
	}
	return ids
}

// undoReply takes back every change of one reply, newest first, and says
// how many; one that cannot be stops it there, with the rest left done.
func (s *Server) undoReply(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	m, err := s.app.Store.Get(records.MessageType, r.PathValue("id"))
	if err != nil {
		s.failed(w, r, "Not undone", fmt.Errorf("no reply %s", r.PathValue("id")), "/chat")
		return
	}
	ids := replyUndos(s.receipt(m.Fields["changes"], true))
	if ids == nil {
		s.failed(w, r, "Not undone", fmt.Errorf("this reply's changes cannot all be undone now; undo them one at a time"), "/chat")
		return
	}
	for i := len(ids) - 1; i >= 0; i-- {
		if err := s.app.Records.UndoAs("human", ids[i]); err != nil {
			s.failed(w, r, "Not all undone", fmt.Errorf("%d of %d changes were undone; the next could not be: %v", len(ids)-1-i, len(ids), err), "/chat")
			return
		}
	}
	s.tell(w, r, outcome{Title: "Undone", Text: fmt.Sprintf("All %d changes of that reply are taken back.", len(ids))}, "/chat")
}
