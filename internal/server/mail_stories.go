package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/mailin"
)

// Connecting email is a hundred little stories, one per mail service, and
// a person only ever reads theirs, once. So the page asks for the address
// first and then tells that service's story alone: its own pages, its own
// words, its own way round what it cannot do. They need not look alike;
// the services do not. Afterwards there is nothing to remember: the
// address to send to is a contact to add, and the folder is made for them.

// mailStory is one service's way to connect.
type mailStory struct {
	Service string
	// Steps are said in order; each may hold a link to the service's page.
	Steps []string
	// Plus says the service delivers you+sameway@ to the inbox; without it,
	// mail is moved to the Sameway folder, which connecting makes.
	Plus bool
	// Cannot is why this service cannot be read with a password, and what
	// to do instead; the rest is not shown.
	Cannot string
	// AskHost is a service Sameway does not know, whose server is asked.
	AskHost bool
}

var htmlEsc = template.HTMLEscapeString

func outLink(href, words string) string {
	return `<a class="sw-link" href="` + href + `">` + words + `</a>`
}

// storyFor is the story for an address.
func storyFor(user string) mailStory {
	_, domain, _ := strings.Cut(strings.ToLower(strings.TrimSpace(user)), "@")
	switch domain {
	case "gmail.com", "googlemail.com":
		return mailStory{Service: "Gmail", Plus: true, Steps: []string{
			"Open " + outLink("https://myaccount.google.com/apppasswords", "Google's app passwords page") + ". If Google says app passwords are not available, " + outLink("https://myaccount.google.com/signinoptions/twosv", "turn on 2-Step Verification") + " first, then open it again.",
			"Type Sameway as the name and press Create.",
			"Copy the 16 letters Google shows and paste them below. The spaces do not matter.",
		}}
	case "icloud.com", "me.com", "mac.com":
		return mailStory{Service: "iCloud Mail", Steps: []string{
			"Open " + outLink("https://account.apple.com/account/manage", "your Apple Account") + ", choose Sign-In and Security, then App-Specific Passwords.",
			"Add one named Sameway, and paste the password Apple shows below.",
		}}
	case "yahoo.com", "yahoo.co.uk", "ymail.com", "rocketmail.com":
		return mailStory{Service: "Yahoo Mail", Steps: []string{
			"Open " + outLink("https://login.yahoo.com/account/security", "Yahoo's account security page") + " and choose Generate app password.",
			"Name it Sameway, and paste the password Yahoo shows below.",
		}}
	case "fastmail.com", "fastmail.fm":
		return mailStory{Service: "Fastmail", Plus: true, Steps: []string{
			"Open " + outLink("https://app.fastmail.com/settings/security/integrations", "Fastmail's Privacy and Security settings") + " and add an app password named Sameway, with access to mail (IMAP).",
			"Paste the password Fastmail shows below.",
		}}
	case "outlook.com", "hotmail.com", "live.com", "msn.com", "outlook.co.uk", "hotmail.co.uk":
		return mailStory{Service: "Outlook.com", Cannot: "Outlook.com no longer lets programs read mail with a password, so Sameway cannot read it. Instead, on your phone open the email in the Outlook app, choose Share, then Sameway: it is saved and sorted the same way. On a computer, " + outLink("/share", "Save to Sameway") + " does the same from your browser."}
	case "proton.me", "protonmail.com", "pm.me":
		return mailStory{Service: "Proton Mail", Cannot: "Proton Mail keeps mail encrypted, so only Proton's own Bridge app can read it, and Bridge needs a paid plan. With Bridge, choose Other below and give its address. Otherwise, share an email to Sameway from the Proton app on your phone."}
	}
	return mailStory{Service: domain, Plus: true, AskHost: true, Steps: []string{
		"In your mail account's security settings, make an app password for Sameway; if it has none, your usual password works.",
		"Check the mail server below: it is your service's IMAP address, often imap. and your domain, port 993.",
	}}
}

// mailStoryPage is the address asked, then that service's story alone.
func (s *Server) mailStoryPage(r *http.Request) string {
	esc := htmlEsc
	user := strings.TrimSpace(r.URL.Query().Get("user"))
	var b strings.Builder
	if !strings.Contains(user, "@") {
		b.WriteString(`<p>Anything you forward to Sameway comes in as a note and is sorted for you on Today: a booking, a bill, a note to self. First, which email do you use?</p>`)
		b.WriteString(`<form method="get" action="/mail" class="sw-stack">`)
		b.WriteString(string(s.component("text-field", map[string]any{"label": "Your email address", "name": "user", "type": "email", "required": true, "autocomplete": "email", "spellcheck": false})))
		b.WriteString(string(s.component("button", map[string]any{"label": "Next", "type": "submit"})) + `</form>`)
		return b.String()
	}
	st := storyFor(user)
	if st.Cannot != "" {
		b.WriteString(`<p>` + st.Cannot + `</p><p>` + outLink("/mail", "Use another address") + `</p>`)
		return b.String()
	}
	b.WriteString(`<p>` + esc(st.Service) + ` makes a password just for Sameway, so your own is never given. It takes a minute:</p><ol class="sw-stack">`)
	for _, step := range st.Steps {
		b.WriteString(`<li>` + step + `</li>`)
	}
	b.WriteString(`</ol><form method="post" action="/mail/connect" class="sw-stack"><input type="hidden" name="user" value="` + esc(user) + `">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Password for Sameway", "name": "password", "type": "password", "required": true, "autocomplete": "off", "spellcheck": false, "hint": "Kept on this computer only, outside the workspace."})))
	if st.AskHost {
		b.WriteString(string(s.component("text-field", map[string]any{"label": "Mail server", "name": "host", "value": mailin.Host(user), "spellcheck": false, "hint": "Its IMAP address and port, such as imap.example.com:993."})))
	}
	b.WriteString(string(s.component("button", map[string]any{"label": "Connect " + st.Service, "type": "submit"})) + `</form>`)
	b.WriteString(`<p class="sw-small">` + outLink("/mail", "Not "+esc(user)+"?") + `</p>`)
	return b.String()
}

// sendTo says, once connected, the one thing to know: where to send.
func sendTo(user string) string {
	if storyFor(user).Plus {
		return `Forward or send anything to <strong>` + htmlEsc(mailin.PlusAddress(user)) + `</strong>, or move it to the folder called ` + mailin.Folder + `. ` +
			outLink("/mail/contact.vcf", "Add Sameway to your contacts") + `, and it is there when you type Sameway as who to send to.`
	}
	return `Move or copy any email to the folder called ` + mailin.Folder + `, which Sameway made in your mailbox. On a phone, swipe the email and choose Move.`
}

// mailContact is a contact card for the address to send to, so the
// person need never remember it.
func (s *Server) mailContact(w http.ResponseWriter, r *http.Request) {
	a, ok := s.mailAccount()
	if !ok || !storyFor(a.User).Plus {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/vcard; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="Sameway.vcf"`)
	w.Write([]byte("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Sameway\r\nN:;Sameway;;;\r\nEMAIL;TYPE=INTERNET:" + mailin.PlusAddress(a.User) + "\r\nNOTE:Send or forward anything here and it comes in to your workspace.\r\nEND:VCARD\r\n"))
}
