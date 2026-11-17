package server

import (
	"html/template"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// A choice a message carried (chat/offers.go), at its own address, for
// whoever got the message: what pressing it does, in Sameway's words, and
// one button that does it. A link alone changes nothing, so a mail
// service that opens links to check them takes no choice. Once one of the
// message's choices is taken, each says which.

const (
	offerPrefix = "/offer/"
	offerGet    = "GET /offer/{token}"
	offerPost   = "POST /offer/{token}"
)

func (s *Server) offerPage(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	p := s.app.Chat.OfferByToken(token)
	if p == nil {
		http.NotFound(w, r)
		return
	}
	label, _ := p.Fields["yes"].(string)
	what, _ := p.Fields["asked"].(string)
	state, _ := p.Fields["state"].(string)
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"choice": label, "does": what, "state": offerWords(state)})
		return
	}
	body := template.HTML(`<p>This would ` + template.HTMLEscapeString(what) + `.</p>`)
	if state == "offered" {
		body += s.form(ui.Form{Action: offerPrefix + token, Button: &ui.Button{Label: label}})
	} else {
		body += s.part(ui.Status{Message: offerWords(state)})
	}
	s.page(w, r, label, body, pageOptions{})
}

func (s *Server) offerTake(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if s.app.Chat.OfferByToken(token) == nil {
		http.NotFound(w, r)
		return
	}
	if err := s.app.Chat.Take(token); err != nil {
		if wantsJSON(r) {
			writeJSON(w, http.StatusConflict, map[string]any{"said": err.Error()})
			return
		}
		s.page(w, r, "Not done", s.part(ui.Status{Message: err.Error()}), pageOptions{Status: http.StatusConflict})
		return
	}
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"state": "chosen", "said": offerWords("accepted")})
		return
	}
	http.Redirect(w, r, offerPrefix+token, http.StatusSeeOther)
}

// offerWords is what became of a choice, in a sentence.
func offerWords(state string) string {
	switch state {
	case "accepted":
		return "Done: this was your choice, and they have been told."
	case "dismissed":
		return "Another choice was taken, so this one is closed."
	}
	return "Not chosen yet."
}
