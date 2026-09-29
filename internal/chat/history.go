package chat

import "github.com/tristanlawrenceguy/sameway/internal/llm"

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
	var ids [][]string // the pictures each of out comes with
	for i := range recs {
		role, _ := recs[i].Fields["role"].(string)
		content, _ := recs[i].Fields["content"].(string)
		switch role {
		case "user":
			// A file that came with the message comes with it to the model too.
			if fileID, _ := recs[i].Fields["file"].(string); fileID != "" {
				content += s.attachment(fileID)
			}
			file, _ := recs[i].Fields["file"].(string)
			ids = append(ids, s.pictureIDs(content, file))
			out = append(out, llm.Message{Role: llm.RoleUser, Content: content})
		case "assistant":
			// The tools this reply used come first, as the turn they were.
			for _, m := range replay(recs[i].Fields["tools"]) {
				out, ids = append(out, m), append(ids, nil)
			}
			out, ids = append(out, llm.Message{Role: llm.RoleAssistant, Content: content}), append(ids, nil)
		}
	}
	s.withPictures(out, ids)
	// Providers require the conversation to start with a user turn.
	for len(out) > 0 && out[0].Role != llm.RoleUser {
		out = out[1:]
	}
	return out, nil
}
