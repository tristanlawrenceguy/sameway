package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"

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
	// A message's changes are said as the chat says them, so an agent
	// that sends a change as it was stored (set, ui.text, large) sees the
	// line a person would: changed text size to Large.
	if name == "message" {
		props["changes"] = s.lookChanges(props["changes"])
	}
	if _, err := c.Validate(props); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "invalid", Message: chat.PropsTrouble(err)}})
		return
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
	// Read before it is added, a block that could not be shown says so,
	// as adding it would.
	if _, problem := blocks.Check(s.app.Blocks, name, props); problem != "" {
		outline.Problems = append(outline.Problems, "this "+name+" cannot be shown as it is set up: "+problem)
	}
	writeJSON(w, http.StatusOK, map[string]any{"component": name, "html": string(html), "outline": outline})
}

// lookChanges is a message's changes as event lines, through the one
// wording the chat and the log share. Anything else is left for the
// manifest to refuse.
func (s *Server) lookChanges(changes any) any {
	list, ok := changes.([]any)
	if !ok {
		return changes
	}
	out := make([]any, 0, len(list))
	for _, item := range list {
		c, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		props := map[string]any{}
		for _, k := range []string{"actor", "who", "person", "via", "undo"} {
			if v, ok := c[k]; ok {
				props[k] = v
			}
		}
		// A change as a reply stored it names its log entry; its Undo
		// posts there.
		if id, _ := c["activity"].(string); id != "" && props["undo"] == nil {
			props["undo"] = "/activity/" + id + "/undo"
		}
		href, _ := c["href"].(string)
		s.say(props, c, href)
		out = append(out, props)
	}
	return out
}
