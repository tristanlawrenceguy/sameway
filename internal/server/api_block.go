package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// shownRecord is a record as the API answers a write of it: the record
// with its title, and for a block what it shows, so the one who wrote it has something to notice.
type shownRecord struct {
	titled
	Shows string `json:"shows,omitempty"`
	// Layout is how the block's tab reads now; see chat.LayoutNow.
	Layout string `json:"layout,omitempty"`
}

// blockWrite holds a block written through the API to what the
// assistant's are held to: a component there is, props that fit it, and
// something it can show. fields is the body; was is the block before, for
// a change. It answers 422 and says so when the block may not be written;
// otherwise shows is what it would show.
func (s *Server) blockWrite(w http.ResponseWriter, r *http.Request, fields map[string]any, was *store.Record) (shows string, refused bool) {
	if r.PathValue("type") != chat.BlockType {
		return "", false
	}
	_, newProps := fields["props"]
	_, newComponent := fields["component"]
	if !newProps && !newComponent {
		return "", false // moved or restyled: what it shows is as it was
	}
	name, _ := fields["component"].(string)
	props, _ := fields["props"].(map[string]any)
	if was != nil {
		if !newComponent {
			name, _ = was.Fields["component"].(string)
		}
		if !newProps {
			props, _ = was.Fields["props"].(map[string]any)
		}
	}
	c, ok := s.app.Registry.Get(name)
	if !ok {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "invalid",
			Message: fmt.Sprintf("no component %q; the components are %s", name, strings.Join(s.app.Registry.Names(), ", "))}})
		return "", true
	}
	if props == nil {
		props = map[string]any{}
	}
	if _, err := c.Validate(props); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "invalid", Message: chat.PropsTrouble(err)}})
		return "", true
	}
	shows, err := s.app.Chat.CheckBlock(name, props)
	var cannot *chat.CannotShow
	if errors.As(err, &cannot) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "cannot_show", Message: err.Error()}})
		return "", true
	}
	return shows, false
}
