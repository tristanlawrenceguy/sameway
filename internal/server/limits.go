package server

import (
	"html/template"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
)

// What this computer and its model cannot do is said before a person
// finds out by trying: a picture sent to a model that cannot see, a
// recording with no speech-to-text to write it down. Each line says what
// happens instead, and what would do it.

// limitsSection is the help page's account of them.
func (s *Server) limitsSection(owner bool) string {
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="help-limits"><h2 id="help-limits">What this computer can do</h2><ul>`)
	line := func(text string) { b.WriteString("<li>" + text + "</li>") }

	if p := s.app.Chat.Provider; p == nil || s.app.Chat.ProviderErr != nil {
		line("The assistant has no model to answer with yet; Connect a model on the chat page.")
	} else {
		model := template.HTMLEscapeString(p.Name())
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
	case s.speechKit().Ready():
		line("Recordings: speech-to-text is on this computer, so a recording you add is written down.")
		switch {
		case s.speakers().Ready():
			line("Speakers: told apart in a recording written down whole, as Speaker 1, Speaker 2; edit the text to name them. A call recorded with this computer's sound says you and them instead.")
		case owner:
			line(`Speakers: not told apart yet, so a transcript says who spoke only for a call recorded with this computer's sound. Telling them apart is a ` + sizeWords(speech.SpeakersSize()) + ` download, and recordings never leave this computer. <form method="post" action="/speech/speakers/get">` +
				string(s.component("button", map[string]any{"label": "Get speaker separation", "type": "submit", "variant": "secondary"})) + `</form>`)
		}
	case !speech.Supported() && !s.speech.given:
		line("Recordings: there is no speech-to-text for this kind of computer, so a recording keeps a transcript only when one is written by hand.")
	case owner:
		line(`Recordings: speech-to-text is not on this computer yet, so a recording keeps a transcript only when one is written by hand. Getting it is a ` + sizeWords(speech.DownloadSize()) + ` download. <form method="post" action="/speech/get">` +
			string(s.component("button", map[string]any{"label": "Get speech-to-text", "type": "submit", "variant": "secondary"})) + `</form>`)
	default:
		line("Recordings: speech-to-text is not on this computer yet, so a recording keeps a transcript only when one is written by hand. The owner can get it.")
	}
	if owner {
		for _, l := range s.appLines() { // meeting_fetch_help.go
			line(l)
		}
	}
	b.WriteString(`</ul></section>`)
	return b.String()
}
