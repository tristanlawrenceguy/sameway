package server

import (
	"html/template"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// What this computer and its model cannot do is said before a person
// finds out by trying: a picture sent to a model that cannot see, a
// recording with no speech-to-text to write it down. Each line says what
// happens instead, and what would do it.

// limitsSection is the help page's account of them.
func (s *Server) limitsSection(owner bool) string {
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="help-limits"><h2 id="help-limits">What this computer can do</h2><ul class="sw-limits">`)
	line := func(text string) { b.WriteString("<li>" + text + "</li>") }

	if p := s.app.Chat.Provider; p == nil || s.app.Chat.ProviderErr != nil {
		line("The assistant has no model to answer with yet; Connect a model on the chat page.")
	} else {
		model := template.HTMLEscapeString(llm.Words(p))
		switch sees, known := chat.Sees(p.Name()); {
		case !known:
			line("Pictures: the assistant's model, " + model + ", has not been sent one yet. If it cannot see them it says so, and answers from a picture's description.")
		case sees:
			line("Pictures: the assistant's model, " + model + ", sees them. It can describe one for you to check.")
		default:
			line("Pictures: the assistant's model, " + model + ", cannot see them, so it answers from a picture's description. A model that sees pictures (Claude, or llava or qwen2.5-vl on this computer) would read them.")
		}
	}

	switch {
	case s.media.SpeechKit().Ready():
		line("Recordings: speech-to-text is on this computer, so a recording you add is written down.")
		switch {
		case s.media.Speakers().Ready():
			line("Speakers: told apart in a recording written down whole, as Speaker 1, Speaker 2; edit the text to name them. A call recorded with this computer's sound says you and them instead.")
		case owner:
			line(`Speakers: not told apart yet, so a transcript says who spoke only for a call recorded with this computer's sound. Telling them apart is a ` + sizeWords(speech.SpeakersSize()) + ` download, and recordings never leave this computer. ` +
				string(s.form(ui.Form{Action: "/speech/speakers/get", Button: &ui.Button{Label: "Get speaker separation", Variant: ui.Secondary}})))
		}
	case !speech.Supported() && !s.media.SpeechGiven():
		line("Recordings: there is no speech-to-text for this kind of computer, so a recording keeps a transcript only when one is written by hand.")
	case owner:
		line(`Recordings: speech-to-text is not on this computer yet, so a recording keeps a transcript only when one is written by hand. Getting it is a ` + sizeWords(speech.DownloadSize()) + ` download. ` +
			string(s.form(ui.Form{Action: "/speech/get", Button: &ui.Button{Label: "Get speech-to-text", Variant: ui.Secondary}})))
	default:
		line("Recordings: speech-to-text is not on this computer yet, so a recording keeps a transcript only when one is written by hand. The owner can get it.")
	}
	line(`Saving from elsewhere: a page, a photo or a few words, from your browser's bookmarks bar or your phone's Share menu, with <a class="sw-link" href="/share">Save to Sameway</a>.`)
	if owner {
		line(s.phoneLine())                // phone.go
		line(s.mailLine())                 // mail_in.go
		line(appsLine())                   // apps_page.go
		if l := s.meaningLine(); l != "" { // search_meaning.go
			line(l)
		}
		line(s.briefLine())
		line(s.reviewLine())               // review.go          // today.go
		lines, setup := s.media.AppLines() // meeting_fetch_help.go
		for _, l := range lines {
			line(l)
		}
		if len(setup) > 0 {
			inner := "<ul>"
			for _, l := range setup {
				inner += "<li>" + l + "</li>"
			}
			if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": "Transcripts from Teams or Zoom"}, template.HTML(inner+"</ul>")); err == nil {
				line(string(body))
			}
		}
	}
	b.WriteString(`</ul></section>`)
	return b.String()
}
