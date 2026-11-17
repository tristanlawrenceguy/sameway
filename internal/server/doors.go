package server

import "strings"

// doors are the ways in the server keeps, and whether each is open, as
// the assistant's ways_in says them (chat/ways_in.go).
func (s *Server) doors() []string {
	pub := s.Published()
	open := func(on bool, yes, no string) string {
		if on {
			return yes
		}
		return no
	}
	address := ""
	if s.app.Chat.Address != nil {
		address = strings.TrimSuffix(s.app.Chat.Address(), "/")
	}
	mail := "Email: not connected (Email in)"
	if s.app.Chat.MailFrom != nil && s.app.Chat.MailFrom() != "" {
		mail = "Email: answered at " + s.app.Chat.MailFrom() + "; replies to what Sameway sends, and mail from people let in, are answered in their thread"
	}
	_, bot := s.bot()
	linked := s.app.Store.Meta(telegramMe) != ""
	return []string{
		mail,
		open(pub.Any(), "Published pages: "+s.publishedKinds(pub)+open(address != "", " at "+address, ""), "Published pages: nothing published"),
		open(pub.Book, "Booking link: on, at /book", "Booking link: off"),
		open(bot, "Telegram: @"+s.app.Store.Meta(telegramName)+open(linked, ", linked to you", ", not yet linked to you"), "Telegram: not connected (/telegram)"),
		open(s.keys().Get(stripeKey) != "", "Payments: your Stripe connected (/payments); each link waits for your press", "Payments: Stripe not connected (/payments)"),
		open(pub.Any(), "Your website: the ask box can go on it (the snippet is on /question)", "Your website: needs something published first"),
		open(pub.Any(), "Other agents: find you by /.well-known/agent-card.json and /llms.txt, and ask over A2A and MCP", "Other agents: nothing to find until something is published"),
	}
}
