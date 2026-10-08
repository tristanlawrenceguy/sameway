package chat

import (
	"context"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A first message to a model on this computer waited sixteen seconds
// before the model began: four to load it, twelve to read Sameway's
// instructions and tools, some seventeen thousand tokens. A second
// message waited under half a second, because Ollama keeps what it has
// read and only reads what is new. So when a page with the conversation
// opens, Sameway sends the same instructions, tools and conversation
// ahead, asking for a single word, and asks Ollama to keep the model for
// half an hour: the person's first message starts where a second one
// would. A model elsewhere is not asked: it is quick, and costs money.

var warmed sync.Map // model -> time.Time

// warmEvery is how seldom a model is readied: a page opened again soon
// after finds it ready still.
const warmEvery = 4 * time.Minute

// Warm readies a model on this computer for this conversation's next
// message, at most every few minutes, and says whether it did.
func (s *Service) Warm(ctx context.Context) bool {
	o, ok := s.Provider.(*llm.OpenAI)
	if !ok || !llm.IsOllama(o.BaseURL) {
		return false
	}
	if at, ok := warmed.Load(o.Model); ok && time.Since(at.(time.Time)) < warmEvery {
		return false
	}
	warmed.Store(o.Model, time.Now())
	history, err := s.history()
	if err != nil {
		return false
	}
	// The person's next message comes after all of this; a placeholder
	// stands where it will be, so the model reads everything before it.
	msgs := append(history, llm.Message{Role: llm.RoleUser, Content: "."})
	// The tools a turn is given (toolset.go), so what is read ahead is the
	// start the next turn shares: the core, and what the chat has used.
	req := llm.Request{System: s.systemPrompt(), Messages: msgs, Tools: s.toolsFor(s.Tools(), msgs)}
	one := *o
	one.MaxTokens = 1
	one.Complete(ctx, req)
	llm.OllamaKeep(ctx, o.Model, "30m")
	return true
}
