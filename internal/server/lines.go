package server

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// One change, one way of saying it. The activity log and the Changes made
// list under a reply both show a change as the event component, from the
// props line builds, and line says it through chat.Say: so a wording fixed
// here is fixed in both, and the same change reads the same in both.

// line is one log entry as the event component shows it: who, the words,
// where it leads, and Undo while it can still be undone. The page it is
// on gives from, where Undo returns: the event, or the message around it.
// canUndo false leaves Undo out whatever the entry.
func (s *Server) line(r *store.Record, canUndo bool) map[string]any {
	props := map[string]any{"actor": r.Fields["actor"]}
	// Where it was done from, when not here, as the log's own summary says.
	if via, _ := r.Fields["via"].(string); via != "" && !strings.HasPrefix(via, "through ") {
		props["via"] = "on " + via
	}
	if who, person := s.whoDid(r); who != "" {
		props["who"], props["person"] = who, person
	}
	// An agent by the name it gave and how it came in, as the summary says
	// it: Claude Code (through MCP).
	if r.Fields["actor"] == chat.ActorAgent {
		by, _ := r.Fields["by"].(string)
		via, _ := r.Fields["via"].(string)
		props["who"] = chat.AgentWho(by, via)
	}
	href := s.hrefFor(r)
	s.say(props, r.Fields, href)
	if canUndo && s.app.Chat.Undoable(r) {
		props["undo"] = "/activity/" + r.ID + "/undo"
	}
	return props
}

// say puts a change's words and its link into an event's props.
func (s *Server) say(props, fields map[string]any, href string) {
	w := chat.Say(fields, href != "")
	props["action"] = w.Action
	if w.Target != "" {
		props["target"] = schema.Words(w.Target)
	}
	detail := w.Detail
	// Old entry activity records stored a raw database ID in their detail
	// field (before recordTitle was fixed for EntryType). Resolve those to
	// readable titles like "Reading: 30 minutes".
	if w.Target == "entry" && w.Detail != "" {
		targetID, _ := fields["target_id"].(string)
		if targetID != "" && w.Detail == targetID {
			if et, ok := s.app.Types.Get("entry"); ok {
				if rec, err := s.app.Store.Get("entry", targetID); err == nil {
					if title := s.title(et, rec); title != "" {
						detail = title
					}
				}
			}
		}
	}
	if detail != "" {
		props["detail"] = detail
	}
	if href != "" {
		props["href"] = href
	}
}

// receipt is what a reply changed, as the message's Changes made list
// shows it: each change as its log entry says it, the same line as on
// /activity. Undo is offered only under the newest reply (latest), where
// undo is a moment; older receipts keep their links, and the log keeps
// the control. A change with no entry in the log, from before there was
// one, is said from what the reply stored.
func (s *Server) receipt(changes any, latest bool) []any {
	list, _ := changes.([]any)
	out := make([]any, 0, len(list))
	for _, item := range list {
		c, ok := item.(map[string]any)
		if !ok {
			continue
		}
		stored, _ := c["href"].(string)
		if id, _ := c["activity"].(string); id != "" {
			if entry, err := s.app.Store.Get(chat.ActivityType, id); err == nil {
				props := s.line(entry, latest)
				// A change to a kind, not one thing, such as a field added
				// to notes, has no id the log can follow; its page is the
				// one the reply kept.
				if tid, _ := entry.Fields["target_id"].(string); props["href"] == nil && tid == "" && stored != "" {
					s.say(props, entry.Fields, stored)
				}
				out = append(out, props)
				continue
			}
		}
		props := map[string]any{"actor": "assistant"}
		s.say(props, c, stored)
		out = append(out, props)
	}
	return out
}

// messageProps is one message as the chat shows it, whether the page was
// loaded or the message arrived live, so the two are the same HTML.
// latest is the newest message, whose receipt can offer Undo.
func (s *Server) messageProps(m *store.Record, from string, latest bool) map[string]any {
	props := map[string]any{
		"role": m.Fields["role"], "content": m.Fields["content"], "id": "msg-" + m.ID, "from": from,
		"time": messageTime(m.CreatedAt), "datetime": m.CreatedAt.UTC().Format(time.RFC3339),
		"changes": s.receipt(m.Fields["changes"], latest),
	}
	if fileID, _ := m.Fields["file"].(string); fileID != "" {
		props["attachment"] = s.attachment(fileID)
	}
	return props
}
