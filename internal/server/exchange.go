package server

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server/exchange"
	"github.com/tristanlawrenceguy/sameway/internal/server/media"
)

// exchangeRoutes are imports and exports (internal/server/exchange).
func exchangeRoutes() []route {
	return served(exchange.Routes, func(s *Server) *exchange.Service { return s.exchange })
}

// What exchange needs of the server beyond web.Deps (exchange.Deps).
var _ exchange.Deps = face{}

func (f face) Crumbs(listHref, listLabel, here string, dot int) template.HTML {
	return f.crumbs(listHref, listLabel, here, dot)
}
func (f face) DotOf(typeName string) int  { return f.dotOf(typeName) }
func (f face) Listed(t *schema.Type) bool { return f.listed(t) }
func (f face) H24() bool                  { return f.h24() }
func (f face) CSS() []byte                { return f.css }
func (f face) Request(method, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	return f.request(method, path, form, cookies...)
}
func (f face) ForReaders(src []byte, allowed func(path string) bool) []byte {
	return forReaders(src, allowed)
}
func (f face) Media() *media.Service { return f.media }
