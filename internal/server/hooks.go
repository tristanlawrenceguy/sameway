package server

import "github.com/tristanlawrenceguy/sameway/internal/records"

// hooks are what the rest of the app asks of the server: a link to a
// record in a reply reads as the record's name, the assistant can look at
// a page, and it is shown a picture when its model can see (pictures.go).
// A block is checked when written by internal/blocks, which needs no
// server.
func (s *Server) hooks() {
	s.app.Registry.LinkTitle = s.linkTitle
	s.app.Chat.Look = s.lookFor
	s.app.Chat.Picture = s.media.PictureFor
	s.app.Automate.Pictures = s.media.PicturesFor // a photo or a scan of a letter is read when sorted
	s.app.Chat.Home = s                           // recordings and workspaces; see home.go
	s.app.Chat.Published = func() map[string]bool { return s.Published().Types }
	// What a change says, the same for every way in: the API's writes as
	// the assistant's tools (chat/notice.go).
	s.app.Chat.SayNotes()
	// A record arriving from another computer made out for this one's
	// owner tells them (foryou.go).
	s.app.Store.AfterSync = s.forYou
	// An agent waiting on /api/changes hears a change logged here at once
	// (api/changes.go).
	records.Listen(s.app.Store, func(_ string, c records.Change) {
		if c.Activity != "" {
			s.api.Logged()
		}
	})
	// A request from anyone, however it came, tells the owner it waits
	// (chat/requests_made.go).
	s.app.Chat.Requested = func(summary string) {
		if s.notify != nil {
			go s.notify("A request waits for your yes", summary, s.linkTo("/today"))
		}
		go s.telegramAsk(summary) // telegram.go: with Make it and No to press
	}
	s.app.Chat.Channel = func(v records.Visitor, text string) bool { return s.telegramTell(v, text) || s.callTell(v, text) } // telegram.go, calls.go
	// The assistant writes from the mailbox Email in reads
	// (chat/send_email.go).
	s.app.Chat.MailFrom, s.app.Chat.Mail = s.inbox.MailFrom, s.inbox.SendMail
	// What an automation did on its own is told like a ring (ring.go).
	s.app.Automate.Tell = func(title, text, url string) {
		if s.notify != nil {
			go s.notify(title, text, s.linkTo(url))
		}
	}
}
