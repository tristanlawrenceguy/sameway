package chat

import "strings"

// Similar is what adding a block says when the tab, or another, already
// shows the same records (tools_similar.go), for a caller that is not the
// conversation, such as POST /api/block: an agent adding a second list of
// tasks hears it as the assistant does. "" when there is nothing like it.
func (s *Service) Similar(canvas, component string, props map[string]any, id string) string {
	return strings.TrimSpace(s.similarOn(canvas, component, props, id))
}
