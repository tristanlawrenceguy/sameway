package server

import (
	"errors"
	"fmt"
	"strings"
)

// The machine's side of the assistant's tools (chat/home_tools.go): the
// same as a recording's Write it down and the workspaces page's buttons,
// said in words for the assistant to pass on.

// MakeWorkspace makes a workspace beside this one and starts it.
func (s *Server) MakeWorkspace(name string, copy bool) (string, error) {
	did := "made"
	if copy {
		did = "copied this workspace to"
	}
	dir, url, notStarted, err := s.makeWorkspace(name, copy)
	if err != nil {
		return "", err
	}
	if notStarted != nil {
		return fmt.Sprintf("%s %s at %s, but it could not be started from here: %v", did, name, dir, notStarted), nil
	}
	return fmt.Sprintf("%s %s at %s; it is open at %s", did, name, dir, url), nil
}

// makeWorkspace makes a workspace beside this one, blank or as a copy,
// and starts it: the one way, for the workspaces page and the assistant.
func (s *Server) makeWorkspace(name string, copy bool) (dir, url string, notStarted, err error) {
	build := s.blank
	if copy {
		build = s.copy
	}
	if dir, err = build(strings.TrimSpace(name)); err != nil {
		return "", "", nil, err
	}
	url, notStarted = s.start(dir)
	return dir, url, notStarted, nil
}

// OpenWorkspace starts another known workspace, by its name.
func (s *Server) OpenWorkspace(name string) (string, error) {
	var names []string
	for _, o := range s.others() {
		if strings.EqualFold(o.Name, strings.TrimSpace(name)) {
			if o.Running {
				return o.Name + " is open at " + o.URL, nil
			}
			url, err := s.start(o.Dir)
			if err != nil {
				return "", err
			}
			return o.Name + " is open at " + url, nil
		}
		names = append(names, o.Name)
	}
	if len(names) == 0 {
		return "", errors.New("this is the only workspace on this computer")
	}
	return "", fmt.Errorf("no workspace called %q; there are: %s", name, strings.Join(names, ", "))
}

// RestoreWorkspace puts a deleted workspace back from the trash.
func (s *Server) RestoreWorkspace(name string) (string, error) {
	var names []string
	for _, t := range s.machine().TrashedWorkspaces() {
		if strings.EqualFold(t.Name, strings.TrimSpace(name)) {
			back, err := s.machine().Untrash(t.Now)
			if err != nil {
				return "", err
			}
			return back.Name + " is back at " + back.From + "; open_workspace opens it", nil
		}
		names = append(names, t.Name)
	}
	if len(names) == 0 {
		return "", errors.New("the trash has no workspaces in it")
	}
	return "", fmt.Errorf("no deleted workspace called %q; the trash has: %s", name, strings.Join(names, ", "))
}
