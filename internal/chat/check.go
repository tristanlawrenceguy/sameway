package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// CannotShow is a block refused when written: it could only have said on
// the page that it is set up wrong. The words are the page's own, which
// say what is wrong and what to do instead.
type CannotShow struct {
	Component string
	Problem   string
}

func (e *CannotShow) Error() string {
	return fmt.Sprintf("this %s could not be shown: %s", e.Component, e.Problem)
}

// CheckBlock is what a block must pass to be written, whichever way it
// comes: the assistant's tools, an agent's over MCP, the API. It resolves
// the block the way its page will; shows says what it would show, for
// the one who wrote it to notice ("3 tasks, not done, by due"), and a
// *CannotShow error says why it would not show anything, so nothing is
// written. Its props are for the caller to have validated first.
func (s *Service) CheckBlock(component string, props map[string]any) (shows string, err error) {
	if s.Check == nil {
		return "", nil
	}
	shows, problem := s.Check(component, props)
	if problem != "" {
		return "", &CannotShow{Component: component, Problem: problem}
	}
	return shows, nil
}

// showing is a tool's line with what the block shows after it, when that
// can be said.
func showing(line, shows string) string {
	if shows == "" {
		return line
	}
	return line + "; it shows " + shows
}

// writable is whether a block's props may be written by a tool: they fit
// the component, and the block can be shown. shows is what it would show;
// bad is the tool's error when it may not, and nothing should be written.
func (s *Service) writable(tool string, c *render.Component, props map[string]any) (shows string, bad *toolResult) {
	if _, err := c.Validate(props); err != nil {
		r := fail("%s Fix the props and call %s again.", PropsTrouble(err), tool)
		return "", &r
	}
	// Checked now, not only when drawn: a block that could only say it is
	// set up wrong is not written, and the caller is told why and what to
	// do instead.
	shows, err := s.CheckBlock(c.Manifest.Name, props)
	if err != nil {
		r := fail("not saved: %v. Change the props and call %s again.", err, tool)
		return "", &r
	}
	return shows, nil
}
