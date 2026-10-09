package server

import (
	"html/template"
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// What every handler shares is internal/web. The names the server's own
// handlers use most are kept here as its, so the many files that tell an
// outcome or list a route read as before; a feature package uses web's.
type (
	outcome     = web.Outcome
	problem     = web.Problem
	pageOptions = web.PageOptions
	reach       = web.Reach
	apiError    = web.APIError
)

const (
	people  = web.People
	owner   = web.Owner
	inward  = web.Inward
	outward = web.Outward
)

var (
	writeJSON  = web.WriteJSON
	writeError = web.WriteError
)

// face is the server as a feature's handlers are given it (web.Deps).
type face struct{ *Server }

var _ web.Deps = face{}

func (f face) App() *app.App { return f.app }
func (f face) Page(w http.ResponseWriter, r *http.Request, title string, body template.HTML, opts web.PageOptions) {
	f.page(w, r, title, body, opts)
}
func (f face) Component(name string, props map[string]any) template.HTML {
	return f.component(name, props)
}
func (f face) Part(p ui.Part) template.HTML  { return f.part(p) }
func (f face) Form(fm ui.Form) template.HTML { return f.form(fm) }
func (f face) Tell(w http.ResponseWriter, r *http.Request, o web.Outcome, fallback string) {
	f.tell(w, r, o, fallback)
}
func (f face) TellAt(w http.ResponseWriter, r *http.Request, o web.Outcome, to string) {
	f.tellAt(w, r, o, to)
}
func (f face) Failed(w http.ResponseWriter, r *http.Request, title string, err error, fallback string) {
	f.failed(w, r, title, err, fallback)
}
func (f face) Fail(w http.ResponseWriter, err error)           { f.fail(w, err) }
func (f face) Who(r *http.Request) records.Who                 { return f.who(r) }
func (f face) Record(r *http.Request, c records.Change) string { return f.record(r, c) }
func (f face) Now() time.Time                                  { return f.now() }
func (f face) Showing(r *http.Request, key string) bool        { return f.showing(r, key) }
