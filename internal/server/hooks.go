package server

import (
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// hooks are what the rest of the app asks of the server: a link to a
// record in a reply reads as the record's name, the assistant can look at
// a page, it is shown a picture when its model can see (pictures.go), and a
// block is checked when written as its page will resolve it (check.go).
func (s *Server) hooks() {
	s.app.Registry.LinkTitle = s.linkTitle
	s.app.Chat.Look = s.lookFor
	s.app.Chat.Picture = s.pictureFor
	s.app.Chat.Check = s.blockCheck
	// What a record says at a glance, glance.go, for one record at a time.
	s.app.Records.Glance = func(t *schema.Type, rec *store.Record) string { return s.glanceText(t, rec, nil) }
	s.app.Chat.Home = s // recordings and workspaces; see home.go
	// A record arriving from another computer made out for this one's
	// owner tells them (foryou.go).
	s.app.Store.AfterSync = s.forYou
	// What an automation did on its own is told like a ring (ring.go).
	s.app.Chat.Tell = func(title, text, url string) {
		if s.notify != nil {
			go s.notify(title, text, s.linkTo(url))
		}
	}
}
