package chat

import (
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// history returns the recent user and assistant turns as model messages.
// Error notices are shown to the person but not sent to the model.
func (s *Service) history() ([]llm.Message, error) {
	recs, err := s.Messages()
	if err != nil {
		return nil, err
	}
	if s.HistoryLimit > 0 && len(recs) > s.HistoryLimit {
		recs = recs[len(recs)-s.HistoryLimit:]
	}
	var out []llm.Message
	for i := range recs {
		role, _ := recs[i].Fields["role"].(string)
		content, _ := recs[i].Fields["content"].(string)
		switch role {
		case "user":
			if fileID, _ := recs[i].Fields["file"].(string); fileID != "" {
				content += s.attachment(fileID)
			}
			out = append(out, llm.Message{Role: llm.RoleUser, Content: content})
		case "assistant":
			out = append(out, replay(recs[i].Fields["tools"])...)
			out = append(out, llm.Message{Role: llm.RoleAssistant, Content: content})
		}
	}
	for len(out) > 0 && out[0].Role != llm.RoleUser {
		out = out[1:]
	}
	return out, nil
}

// fail stores an error notice in the conversation and returns it with the error.
func (s *Service) fail(err error) (*store.Record, error) {
	Record(s.Store, "system", Change{Action: "failed", Detail: truncate(err.Error(), 200)})
	rec, storeErr := s.message(map[string]any{"role": "error", "content": err.Error()})
	if storeErr != nil {
		return nil, storeErr
	}
	return rec, err
}
