package server

import "html/template"

// tasksSection renders a collection of all tasks belonging to this project,
// ordered by most recently updated first. Empty projects still show an empty
// state so the section is present but not absent.
func (s *Server) tasksSection(projectID string) template.HTML {
	props := s.resolveCollection(map[string]any{
		"type":  "task",
		"where": []string{"project=" + projectID},
		"order": "-updated_at",
		"label": "Tasks",
		"id":    "tasks-" + projectID,
	}, "")
	delete(props, "summary") // let the template show "Nothing here yet." for empty projects
	return s.component(collectionComponent, props)
}
