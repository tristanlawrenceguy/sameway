package api

import (
	"fmt"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Several blocks in one request. An agent building a page made its
// records in one call and then each block in a call of its own, since
// blocks were refused as a list; POST /api/block now takes a list as
// records do, each block held to what one alone is (blockCheck), all
// written or none as one change, and PATCH /api/block the same change to
// several. One refused is named by its place, and nothing is written.

// blocksChecked checks each block of a batch, marks it the agent's, and
// gives what each would show; or answers 422 naming the one refused.
func (s *Service) blocksChecked(w http.ResponseWriter, r *http.Request, writes []records.Writing) (shows []string, refused bool) {
	agent := Agent(r)
	shows = make([]string, len(writes))
	for i, wr := range writes {
		var was *store.Record
		if wr.ID != "" {
			was, _ = s.app.Store.Get(records.BlockType, wr.ID)
		}
		said, refusal := s.blockCheck(wr.Fields, was)
		if refusal != nil {
			refusal.Message = fmt.Sprintf("nothing was written: block [%d] was refused: %s", i, refusal.Message)
			web.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": *refusal})
			return nil, true
		}
		shows[i] = said
		byAgent(wr.Fields, agent, wr.ID == "")
		if c, _ := wr.Fields["canvas"].(string); wr.ID == "" && s.app.Chat.HasCanvas(c) {
			s.SeedChat(c)
		}
	}
	return shows, false
}
