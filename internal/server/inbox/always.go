package inbox

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// suggestedAlways is Always on someone's request: made now, and the same
// kind of change from them from then on without asking (chat/request_always.go).
func (s *Service) suggestedAlways(w http.ResponseWriter, r *http.Request) {
	p, err := s.answered(r)
	if err == nil {
		err = s.app.Chat.Always(p.ID)
	}
	if err != nil {
		s.Failed(w, r, "Not made", err, "/today")
		return
	}
	s.TellAt(w, r, web.Outcome{Title: "Made", Text: "Made, and the same from them from now on without asking. Take it off their page to be asked again."}, "/today")
}
