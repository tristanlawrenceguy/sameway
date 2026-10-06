package llm

import "testing"

// A model is named for a person by what it is and where the conversation
// goes, never by the protocol and an address.
func TestAModelIsNamedForAPerson(t *testing.T) {
	for _, c := range []struct {
		p    Provider
		want string
	}{
		{&OpenAI{Model: "sameway-qwen3.5:latest", BaseURL: "http://127.0.0.1:11434/v1"}, "qwen3.5 on this computer"},
		{&OpenAI{Model: "openrouter/auto", BaseURL: "https://openrouter.ai/api/v1"}, "openrouter/auto through openrouter.ai"},
		{&OpenAI{Model: "gpt-5", BaseURL: "https://api.openai.com/v1"}, "gpt-5 through openai.com"},
		{&Anthropic{Model: "claude-sonnet-5"}, "Claude (claude-sonnet-5)"},
		{&Command{Label: "Claude Code"}, "Claude Code"},
	} {
		if got := Words(c.p); got != c.want {
			t.Errorf("%s reads as %q, want %q", c.p.Name(), got, c.want)
		}
	}
}
