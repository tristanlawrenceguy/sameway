package server

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The parts of a page that are off until there is a reason.
//
// A page at rest is what the person came for and nothing else. Not a
// form, and not a link to one either: a row of links is still a row of
// things to read past, and a page that offers eight ways on is a page
// about itself. So a part that is off leaves no trace — no count, no
// "more", no arrow.
//
// It is not gone. Two things turn one on, and both of them are somebody
// deciding it is wanted:
//
//   - the address, ?show=<key>, which opens it for that view and closes
//     again with a link; the assistant hands one over when it has a reason
//     ("you asked what else is in the garden — here it is on the page").
//   - the workspace, set_setting ui.show +<key>, which puts it on every
//     time, until -<key> takes it back.
//
// Which leaves the assistant as the way in, and that is the arrangement:
// it is handed every connection and every field whole (see api_record.go
// and get_record), so it knows what is there when the page does not say.
// The keys are the same in the address and in the setting, and the same
// as the connection keys from internal/relate: one vocabulary for
// everything a page can be asked to show.

const (
	// FieldsPart is a record's whole field list, including the ones its
	// title and the chips under it already say.
	FieldsPart = "fields"
	// RemindPart is the field for setting a reminder about this record.
	RemindPart = "remind"
	// AskPart is the way to the assistant with this record in the box.
	AskPart = "ask"
	// DayPart is the way to this record's day on the canvas calendar.
	DayPart = "day"
)

// shown is what this view has been asked to show: always, from the
// workspace, and here, from the address. They are kept apart because a
// part the address opened can be closed again by a link, and a part the
// workspace turned on is taken back by asking, not by a control on every
// page that carries it.
func (s *Server) shown(r *http.Request) (always, here []string) {
	for _, key := range strings.Split(s.app.Workspace.Config.UI.Show, ",") {
		if key = strings.TrimSpace(key); key != "" {
			always = append(always, key)
		}
	}
	return always, r.URL.Query()["show"]
}

// showing says whether one part is on, either way.
func (s *Server) showing(r *http.Request, key string) bool {
	always, here := s.shown(r)
	return has(always, key) || has(here, key)
}

// fewer is the way back out of a part the address opened, in the quiet
// layer: whoever opened it can close it. A part the workspace turned on
// carries none, because it is on until somebody asks for it to go, and a
// control for that on every page is the thing being avoided.
func (s *Server) fewer(page, key, what string, here []string) string {
	if !has(here, key) {
		return ""
	}
	return `<p class="sw-quiet sw-related__hide">` + string(s.component("link", map[string]any{
		"href": showURL(page, without(here, key)), "label": "Fewer", "context": what, "look": "button",
	})) + `</p>`
}

// headFields are the fields a record's page already says above its
// fields: its title as the heading, and the three the chips under it
// carry. Saying them again in the list below is the same page twice.
// These are exactly what lede and facts draw; keep them in step.
func headFields(t *schema.Type, rec *store.Record) map[string]bool {
	out := map[string]bool{}
	if t.Title != "" {
		out[t.Title] = true
	}
	for _, f := range t.Shown() {
		if f.Type == "bool" {
			out[f.Name] = true
			break
		}
	}
	for _, f := range t.Shown() {
		if f.Type == "enum" {
			out[f.Name] = true
			break
		}
	}
	// The day chip takes the first date that has something in it, so this
	// does too: an empty first date leaves the second one on the chip.
	for _, f := range t.Shown() {
		if f.Type != "datetime" {
			continue
		}
		if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
			out[f.Name] = true
			break
		}
	}
	return out
}
