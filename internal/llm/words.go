package llm

import (
	"net/url"
	"strings"
)

// Words is a model as a person reads it under the chat and on the help
// page: "qwen3.5 on this computer", "Claude (claude-sonnet-5)", not
// "openai-compatible (sameway-qwen3.5:latest at http://127.0.0.1:11434/v1)",
// which is Name, for logs. Where the conversation goes is the part a
// person needs: here, or which company.
func Words(p Provider) string {
	switch p := p.(type) {
	case *OpenAI:
		model := strings.TrimSuffix(strings.TrimPrefix(p.Model, "sameway-"), ":latest")
		u, err := url.Parse(p.BaseURL)
		if err != nil || u.Hostname() == "" {
			return model
		}
		switch host := u.Hostname(); host {
		case "127.0.0.1", "localhost", "::1":
			return model + " on this computer"
		default:
			return model + " through " + strings.TrimPrefix(host, "api.")
		}
	case *Anthropic:
		return "Claude (" + p.Model + ")"
	}
	return p.Name()
}
