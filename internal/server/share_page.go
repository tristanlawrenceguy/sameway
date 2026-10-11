package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// sharePage is the form, filled with whatever the bookmark sent, and how
// to save from a browser and from a phone. The form comes first: from the
// bookmark it is all the person came for, one press away. The bookmark
// opens this page in a new tab, so the page being read stays where it
// was; it never saves on its own, since any site could open this address.
func (s *Server) sharePage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(string(s.form(ui.Form{Action: "/share", Enctype: "multipart/form-data", Class: "sw-stack",
		Body: s.part(ui.TextField{Label: "Title", Name: "title", Value: q.Get("title"), Hint: "Left empty, the page's own title is used."}) +
			s.part(ui.TextField{Label: "Link", Name: "url", ID: "field-url", Type: ui.URL, Value: q.Get("url"), Spellcheck: ui.Bool(false), Hint: "A web page's words are read and kept with it, so it can be found and read later even if the page goes."}) +
			s.component("textarea", map[string]any{"label": "Words", "name": "text", "value": q.Get("text"), "rows": 4}),
		Button: &ui.Button{Label: "Save as a note"}})))
	bookmark := bookmarklet(s.origin(r))
	b.WriteString(`<h2>From your browser</h2><p>Drag this link to your bookmarks bar: <a class="sw-link" href="` + esc(bookmark) + `">Save to Sameway</a>. On any page, press it to save the page here, with any words you selected.</p>`)
	b.WriteString(`<p>Without a mouse, or on an iPhone or iPad: bookmark any page, then edit that bookmark and paste this as its address.</p>`)
	b.WriteString(string(s.part(ui.TextField{Label: "The bookmark's address", Name: "bookmark", Value: bookmark, Spellcheck: ui.Bool(false)})))
	b.WriteString(`<h2>From the Share menu</h2><p>On an Android phone, or a computer with Chrome or Edge, install Sameway as an app (<a class="sw-link" href="/help#help-app">how</a>): it is then in the Share menu of every app, so a page, a photo or some words shared to Sameway are saved as a note. An iPhone does not let a web app into its Share menu, so there use the bookmark above in Safari.</p>`)
	s.page(w, r, "Save to Sameway", template.HTML(b.String()), pageOptions{Lede: "Keep a page, a photo or a few words the moment you see them."})
}

// bookmarklet is the bookmark that saves a page here: its address, its
// title and any words selected, in a new tab that has no hold on the
// page it came from (noopener).
func bookmarklet(origin string) string {
	return "javascript:(()=>{const e=encodeURIComponent;open('" + origin + "/share?url='+e(location.href)+'&title='+e(document.title)+'&text='+e(getSelection().toString()),'_blank','noopener')})()"
}

// origin is how the one asking reached this server, for a link that
// comes back to it.
func (s *Server) origin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
