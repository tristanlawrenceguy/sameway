package server

import (
	"html/template"
	"strings"
)

// Connecting the assistant to a model is a handful of little stories, one
// for what the person already has: a Claude subscription, nothing at all,
// a key from Anthropic or OpenRouter, a ChatGPT subscription. The card put
// every way on show at once, a key field for two companies among them, and
// left the person to work out which was theirs. Now it asks what they have,
// each answer a story of its own, opened alone, in that service's own
// words and steps; the person reads only theirs, once.

// modelStory is one answer to what the person has, and its steps.
type modelStory struct {
	have  string // the question's answer, as they would say it
	steps []string
	key   string // the key it begins with, when the story ends in pasting one
}

func modelStories() []modelStory {
	return []modelStory{
		{have: "A Claude subscription (Pro or Max)", steps: []string{
			"Install " + storyLink("https://claude.com/claude-code", "Claude Code") + ", Anthropic's program for your subscription.",
			"Open it once and sign in with your Claude account.",
			"Come back here: Sameway finds it on its own and offers it above. Your conversations go to Claude through your subscription, at no extra cost.",
		}},
		{have: "Nothing yet, and I want it free and on this computer", steps: []string{
			storyLink(ollamaDownload(), "Download Ollama") + ", which runs AI models on this computer, and install it.",
			"Come back here: Sameway sees it and offers to fetch a free model, a few gigabytes, once. Nothing you write leaves this computer.",
			"It is slower than Claude and gets more wrong; you can change to Claude later without losing anything.",
		}},
		{have: "An Anthropic API key, or I will make one", key: "sk-ant-", steps: []string{
			"Open " + storyLink("https://console.anthropic.com/settings/keys", "Anthropic's keys page") + " and sign in, or make an account.",
			"Press Create Key and name it Sameway. A new account needs a little credit first, under Billing; each message costs a fraction of a cent.",
			"Copy the key, which begins sk-ant-, and paste it here.",
		}},
		{have: "An OpenRouter key", key: "sk-or-", steps: []string{
			"Open " + storyLink("https://openrouter.ai/keys", "OpenRouter's keys page") + " and press Create Key.",
			"Copy the key, which begins sk-or-, and paste it here. OpenRouter picks a model for each message; it reaches Claude, OpenAI's and others.",
		}},
		{have: "A ChatGPT subscription", steps: []string{
			"A ChatGPT subscription cannot be used by other programs: OpenAI sells that separately.",
			"Choose the free model on this computer instead, or an OpenRouter key, which reaches OpenAI's models too.",
		}},
	}
}

// modelStoriesHTML is the question and its stories, each a disclosure
// opened alone.
func (s *Server) modelStoriesHTML(hidden string) string {
	var b strings.Builder
	b.WriteString(`<p>Which of these do you have?</p>`)
	for _, st := range modelStories() {
		var inner strings.Builder
		inner.WriteString(`<ol class="sw-stack">`)
		for _, step := range st.steps {
			inner.WriteString(`<li>` + step + `</li>`)
		}
		inner.WriteString(`</ol>`)
		if st.key != "" {
			inner.WriteString(string(s.keyForm(hidden, st.key))) // model_key.go
		}
		if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": st.have}, template.HTML(inner.String())); err == nil {
			b.WriteString(string(body))
		}
	}
	return b.String()
}

// storyLink is a link in a story's step; one to another site opens in a
// new tab (prose.Outward).
func storyLink(href, words string) string {
	return `<a class="sw-link" href="` + href + `">` + words + `</a>`
}
