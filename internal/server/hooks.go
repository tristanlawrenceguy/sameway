package server

// hooks are what the rest of the app asks of the server: a link to a
// record in a reply reads as the record's name, the assistant can look at
// a page, and it is shown a picture when its model can see (pictures.go).
// A block is checked when written by internal/blocks, which needs no
// server.
func (s *Server) hooks() {
	s.app.Registry.LinkTitle = s.linkTitle
	s.app.Chat.Look = s.lookFor
	s.app.Chat.Picture = s.pictureFor
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
