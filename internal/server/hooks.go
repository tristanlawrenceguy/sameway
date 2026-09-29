package server

// hooks are what the rest of the app asks of the server: a link to a
// record in a reply reads as the record's name, the assistant can look at
// a page, it is shown a picture when its model can see (pictures.go), and a
// block is checked when written as its page will resolve it (check.go).
func (s *Server) hooks() {
	s.app.Registry.LinkTitle = s.linkTitle
	s.app.Chat.Look = s.lookFor
	s.app.Chat.Picture = s.pictureFor
	s.app.Chat.Check = s.blockCheck
	s.app.Chat.Home = s // recordings and workspaces; see home.go
}
