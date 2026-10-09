package web

import (
	"html/template"
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Deps is the server as a feature's handlers see it: the app, and the few
// things every page and action does the same way. It is kept narrow; a
// feature that needs more of the server says so with a seam of its own.
type Deps interface {
	// App is the workspace being served.
	App() *app.App
	// Page renders a body inside the site layout with the shared
	// navigation.
	Page(w http.ResponseWriter, r *http.Request, title string, body template.HTML, opts PageOptions)
	// Component renders a component from props or, if that fails, a
	// visible error, so a broken block never silently disappears.
	Component(name string, props map[string]any) template.HTML
	// Part renders a component from its builder in internal/ui.
	Part(p ui.Part) template.HTML
	// Form renders a form that does one action (ui.Form).
	Form(f ui.Form) template.HTML
	// Tell returns the person to the page they were on, or fallback, with
	// the outcome to show there.
	Tell(w http.ResponseWriter, r *http.Request, o Outcome, fallback string)
	// TellAt sends the person to one page with the outcome.
	TellAt(w http.ResponseWriter, r *http.Request, o Outcome, to string)
	// Failed tells a failure where the person is, in words they can act on.
	Failed(w http.ResponseWriter, r *http.Request, title string, err error, fallback string)
	// Fail reports an unexpected error as a page.
	Fail(w http.ResponseWriter, err error)
	// Who is who made a request's change.
	Who(r *http.Request) records.Who
	// Record logs a change made by a request, and returns its entry's id.
	Record(r *http.Request, c records.Change) string
	// Changed says something changed that open pages should follow.
	Changed()
	// Now is the time it is for the person, from the app's clock.
	Now() time.Time
	// Showing says whether a part of a page that is off until there is a
	// reason is on, by the address (?show=) or the workspace (ui.show).
	Showing(r *http.Request, key string) bool
}
