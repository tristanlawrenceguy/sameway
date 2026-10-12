package api

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// apiJobs is the background work as JSON, the owner's like the section:
// each job's name, every (in seconds), runs, running, last_run,
// last_error and next_run.
func (s *Service) apiJobs(w http.ResponseWriter, r *http.Request) {
	type job struct {
		runner.Status
		Every float64 `json:"every_seconds,omitempty"`
	}
	out := []job{}
	for _, st := range s.app.Jobs.Status() {
		out = append(out, job{Status: st, Every: st.Every.Seconds()})
	}
	web.WriteJSON(w, http.StatusOK, map[string]any{"jobs": out})
}
