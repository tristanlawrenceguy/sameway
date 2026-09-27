package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// lookAtComponent renders one component from props and reads the fragment,
// so an agent sees what a block would be before adding it, and what a
// screen reader would be stuck on.
func (s *Server) lookAtComponent(w http.ResponseWriter, name string, props map[string]any) {
	c, ok := s.app.Registry.Get(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": apiError{Code: "not_found",
			Message: fmt.Sprintf("no component %q; the components are %s", name, strings.Join(s.app.Registry.Names(), ", "))}})
		return
	}
	if props == nil {
		props = map[string]any{}
	}
	if _, err := c.Validate(props); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "invalid", Message: err.Error()}})
		return
	}
	// Transform raw setting-change props to human-readable form.
	if name == "message" {
		if changesAny, ok := props["changes"].([]any); ok {
			cleanedChanges := make([]any, 0, len(changesAny))
			for _, item := range changesAny {
				chg, ok := item.(map[string]any)
				if !ok {
					continue
				}
				copied := map[string]any{}
				for k, v := range chg {
					copied[k] = v
				}
				chat.CleanSettingChange(copied)
				cleanedChanges = append(cleanedChanges, copied)
			}
			props["changes"] = cleanedChanges
		}
	}
	html, err := s.app.Registry.Render(name, props)
	if err != nil {
		writeError(w, err)
		return
	}
	outline, err := look.Fragment(string(html))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"component": name, "html": string(html), "outline": outline})
}
