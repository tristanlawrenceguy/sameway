package server

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Scan to the list: on the phone, a photo of something running out (the
// product, its label or its barcode) and the assistant puts it on the
// shopping list. The photo is kept as a file and goes with a message to
// the assistant, which sees it when its model can; the person is taken
// to the chat to watch it happen. Pages are plain http on the home
// Wi-Fi, so there is no live camera: the file field opens the camera. It
// is held on the owner's phone app icon (manifest shortcuts).

// scanAsk is what the assistant is asked with the photo.
const scanAsk = "Add what this photo shows to my shopping list. Read the product from its label or barcode: its name, with the brand and size when they help to buy the right one. Put it where I keep my shopping; if I keep none yet, start it the way you would a shopping list. If you cannot tell what it is, say so rather than guess. Then say in one line what you added."

// scanPage is the camera field and its one button.
func (s *Server) scanPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	if p := s.app.Chat.Provider; p != nil {
		if sees, known := chat.Sees(p.Name()); known && !sees {
			b.WriteString(`<p class="sw-muted">The assistant's model cannot see pictures, so it cannot read a photo. This needs a model that sees pictures: Claude, or llava or qwen2.5-vl on this computer.</p>`)
		}
	}
	b.WriteString(string(s.form(ui.Form{Action: "/scan", Enctype: "multipart/form-data", Class: "sw-stack",
		Body:   `<div class="sw-field"><label class="sw-field__label" for="scan-photo">Photo</label><p class="sw-field__hint" id="scan-photo-hint">Of the product, its label or its barcode. On a phone this opens the camera.</p><input class="sw-field__input" id="scan-photo" type="file" name="file" accept="image/*" capture="environment" aria-describedby="scan-photo-hint"></div>`,
		Button: &ui.Button{Label: "Add to the shopping list"}})))
	s.page(w, r, "Scan to the list", template.HTML(b.String()), pageOptions{Lede: "Take a photo of something you need, and the assistant puts it on your shopping list."})
}

// scanSave keeps the photo and asks the assistant to add what it shows.
func (s *Server) scanSave(w http.ResponseWriter, r *http.Request) {
	none := errors.New("no photo came: take a photo of the product, its label or its barcode")
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		s.failed(w, r, "Not added", none, "/scan")
		return
	}
	if err := s.chatFor(r).Available(); err != nil {
		s.failed(w, r, "Not added", err, "/scan")
		return
	}
	file, err := s.media.StoreUpload(r)
	if err == http.ErrMissingFile {
		err = none
	}
	if err != nil {
		s.failed(w, r, "Not added", err, "/scan")
		return
	}
	s.startTurn(r, "", scanAsk, file.ID, "/chat")
	s.tellAt(w, r, outcome{Title: "Photo sent", Text: "The assistant is adding what it shows to your shopping list."}, "/chat")
}
