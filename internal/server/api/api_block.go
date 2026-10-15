package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// shownRecord is a record as the API answers a write of it: the record
// with its title, and for a block what it shows, so the one who wrote it has something to notice.
type shownRecord struct {
	titled
	Shows string `json:"shows,omitempty"`
	// Layout is how the block's tab reads now; see chat.LayoutNow.
	Layout string `json:"layout,omitempty"`
	// Notes is what is worth saying about the change, the same for every
	// way in (records/notes.go): a record called as open ones are, a block
	// over records another already shows.
	Notes []string `json:"notes,omitempty"`
	// Undo is how to take the write back (api_undo_said.go).
	Undo *undoSaid `json:"undo,omitempty"`
}

// blockWrite holds a block written through the API to what the
// assistant's are held to: a component there is, props that fit it, and
// something it can show. fields is the body; was is the block before, for
// a change. It answers 422 and says so when the block may not be written;
// otherwise shows is what it would show.
func (s *Service) blockWrite(w http.ResponseWriter, r *http.Request, fields map[string]any, was *store.Record) (shows string, refused bool) {
	if r.PathValue("type") != records.BlockType {
		return "", false
	}
	shows, refusal := s.blockCheck(fields, was)
	if refusal != nil {
		web.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": *refusal})
		return "", true
	}
	return shows, false
}

// blockCheck is blockWrite's check without the answer: what the block
// would show, or why it may not be written.
func (s *Service) blockCheck(fields map[string]any, was *store.Record) (shows string, refusal *web.APIError) {
	_, newProps := fields["props"]
	_, newComponent := fields["component"]
	if !newProps && !newComponent {
		return "", nil // moved or restyled: what it shows is as it was
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
		return "", &web.APIError{Code: "invalid", Message: fmt.Sprintf("no component %q; the components are %s", name, strings.Join(s.app.Registry.Names(), ", "))}
	}
	if props == nil {
		props = map[string]any{}
	}
	if _, err := c.Validate(props); err != nil {
		return "", &web.APIError{Code: "invalid", Message: chat.PropsTrouble(err)}
	}
	shows, err := s.app.Chat.CheckBlock(name, props)
	var cannot *chat.CannotShow
	if errors.As(err, &cannot) {
		return "", &web.APIError{Code: "cannot_show", Message: err.Error()}
	}
	return shows, nil
}
