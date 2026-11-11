package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/server/media"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// photosSection is Photos you take on Workspaces: the folder a phone's
// photos come into on this computer, kept from as they arrive
// (media/folder_in.go). Off, it names the folder an app here copies them
// to, when one is found, ready to press; any other can be typed.
func (s *Server) photosSection(r *http.Request) string {
	if !s.chatFor(r).IsOwner() {
		return ""
	}
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-photos"><h2 id="ws-photos">Photos you take</h2>`)
	if dir := s.media.PhotosFrom(); dir != "" {
		b.WriteString(`<p>New photos and videos in ` + esc(dir) + ` come into your files by themselves, and are sorted like anything you share.</p>`)
		b.WriteString(string(s.form(ui.Form{Action: "/photos/from", Hidden: ui.Hidden("folder", ""), Button: &ui.Button{Label: "Stop keeping new photos", Variant: ui.Secondary}})) + `</section>`)
		return b.String()
	}
	found, value := media.PhotoFolders(), ""
	if len(found) > 0 {
		b.WriteString(`<p>Your phone's photos come to this computer through ` + esc(found[0][0]) + `. Sameway can keep each new one as it arrives, so they are here without sharing them one by one.</p>`)
		value = found[0][1]
	} else {
		b.WriteString(`<p>Sameway can keep each new photo from your phone as it arrives, through the app that copies them to this computer: iCloud for Windows, OneDrive or Google Drive. Name the folder they come into.</p>`)
	}
	b.WriteString(string(s.form(ui.Form{Action: "/photos/from", Class: "sw-stack",
		Body:   s.part(ui.TextField{Label: "Folder", Name: "folder", Value: value, Spellcheck: ui.Bool(false), Hint: "Photos already there stay out; any of them can be added with Save to Sameway."}),
		Button: &ui.Button{Label: "Keep new photos from this folder", Variant: ui.Secondary}})))
	b.WriteString(`</section>`)
	return b.String()
}

// photosFrom starts or stops keeping new photos from a folder.
func (s *Server) photosFrom(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	dir := strings.Trim(strings.TrimSpace(r.PostForm.Get("folder")), `"`)
	if err := s.media.KeepPhotosFrom(dir); err != nil {
		s.tell(w, r, outcome{Failed: true, Title: "Not changed", Problems: []problem{{Field: "folder", Text: err.Error() + ": " + dir}}}, "/workspaces#ws-photos")
		return
	}
	if dir == "" {
		s.tell(w, r, outcome{Title: "New photos are not kept", Text: "The photos already kept stay in your files."}, "/workspaces#ws-photos")
		return
	}
	s.tell(w, r, outcome{Title: "New photos are kept", Text: "Each new photo or video in " + dir + " comes into your files within a minute of arriving."}, "/workspaces#ws-photos")
}
