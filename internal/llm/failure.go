package llm

import (
	"net/url"
	"regexp"
	"strings"
)

// A turn that failed showed what the provider said, "anthropic returned
// HTTP 400: credit balance is too low" or "model server returned 404 Not
// Found: model not found", which tells a person neither what happened nor
// what to do. Failure is what a provider's refusal was, in their words, by
// kind; Error is the sentence the conversation shows, with what to do next.

// FailKind is why a provider did not answer.
type FailKind int

const (
	FailOther  FailKind = iota
	FailKey             // the key is not accepted
	FailCredit          // the account has nothing left to spend
	FailRate            // too many messages just now
	FailBusy            // the provider is overloaded or down for a moment
	FailModel           // the model asked for is not there
	FailLong            // the conversation is too long for the model
)

// Failure is a provider's refusal, sorted.
type Failure struct {
	Kind   FailKind
	Status int
	// Who is the provider by name, "Anthropic", "OpenRouter", "Ollama";
	// Model the model asked for; Said what the provider said.
	Who, Model, Said string
}

// Classify sorts what a provider answered with an error status.
func Classify(status int, said, who, model string) *Failure {
	f := &Failure{Status: status, Who: who, Model: model, Said: strings.TrimSpace(said)}
	low := strings.ToLower(said)
	switch {
	case status == 401 || status == 403 || strings.Contains(low, "invalid api key") || strings.Contains(low, "invalid x-api-key"):
		f.Kind = FailKey
	case status == 402 || strings.Contains(low, "credit balance") || strings.Contains(low, "insufficient credits") || strings.Contains(low, "insufficient_quota"):
		f.Kind = FailCredit
	case strings.Contains(low, "context length") || strings.Contains(low, "context window") || strings.Contains(low, "too long") || strings.Contains(low, "maximum context"):
		f.Kind = FailLong
	case status == 404 || strings.Contains(low, "model") && strings.Contains(low, "not found"):
		f.Kind = FailModel
	case status == 429:
		f.Kind = FailRate
	case status == 500 || status == 502 || status == 503 || status == 529 || strings.Contains(low, "overloaded"):
		f.Kind = FailBusy
	}
	return f
}

// Error is the failure as the conversation says it.
func (f *Failure) Error() string {
	switch f.Kind {
	case FailKey:
		return f.Who + " did not accept the key saved on this computer. Paste a new one where the page asks for a model."
	case FailCredit:
		return f.Who + " says this account has no credit left. Add some on their website, or choose another model."
	case FailRate:
		return f.Who + " is getting more messages from this key than it allows just now. Wait a minute, then send it again."
	case FailBusy:
		return f.Who + " is busy just now and did not answer. Send it again in a moment."
	case FailModel:
		return "The model " + f.Model + " is not there any more. Choose another where the page asks for a model."
	case FailLong:
		return "This chat has grown too long for the model. Start a new chat; this one stays in the list."
	}
	if f.Said == "" {
		return f.Who + " did not answer."
	}
	return f.Who + " did not answer: " + f.Said
}

// Retry says whether sending the same again soon may well work.
func (f *Failure) Retry() bool { return f.Kind == FailRate || f.Kind == FailBusy }

// whoAt names the provider at an OpenAI-compatible address.
func whoAt(base string) string {
	switch {
	case IsOllama(base):
		return "Ollama"
	case strings.Contains(base, "openrouter.ai"):
		return "OpenRouter"
	case strings.Contains(base, "api.openai.com"):
		return "OpenAI"
	}
	if u, err := url.Parse(base); err == nil && u.Hostname() != "" {
		return "The model server at " + u.Hostname()
	}
	return "The model server"
}

// apiMessage is the message inside an API's error, which Anthropic's client
// gives with the method, the address and the whole JSON around it.
func apiMessage(s string) string {
	if m := apiMessageRe.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return s
}

var apiMessageRe = regexp.MustCompile(`"message"\s*:\s*"([^"]*)"`)
