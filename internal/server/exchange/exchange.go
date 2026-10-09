// Package exchange is what comes into a workspace and goes out of it:
// records read from a file (a CSV, contacts, a calendar, a mailbox),
// bringing a person's things from another app, and any list, record,
// block or the whole workspace going out again as a spreadsheet,
// calendar, document or archive. The server holds one Service and adds
// its Routes to the one route table.
package exchange

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server/media"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Deps is what exchange needs of the server: what every handler does
// (web.Deps), and what its pages and documents say as the server's do.
type Deps interface {
	web.Deps
	// Title is a record's name, as its page says it; RefTitle the name of
	// the record a ref field points at.
	Title(t *schema.Type, rec *store.Record) string
	RefTitle(f schema.Field, id string) string
	// Crumbs and DotOf are the way back to a list above a page's heading,
	// and the list's colour.
	Crumbs(listHref, listLabel, here string, dot int) template.HTML
	DotOf(typeName string) int
	// Listed says whether a list is in the sidebar, so in an export of
	// everything.
	Listed(t *schema.Type) bool
	// H24 is whether times are read on the 24-hour clock.
	H24() bool
	// CSS is the site's stylesheet, which a record's web page carries.
	CSS() []byte
	// Request is a page of the server as it answers one request, and
	// ForReaders that page as the internet is sent it (server/public_clean.go).
	Request(method, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder
	ForReaders(src []byte, allowed func(path string) bool) []byte
	// Media is files and recordings: an imported file is kept as one.
	Media() *media.Service
}

// Service is exchange for one workspace.
type Service struct {
	Deps
	app *app.App
}

// New is exchange for the workspace the server serves.
func New(d Deps) *Service { return &Service{Deps: d, app: d.App()} }
