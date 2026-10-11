package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// What Sameway does in the background (internal/runner) is the owner's to
// see, on Help: each job, how often, when it last ran and how that went,
// and when it runs next, in words. An agent reads the same from
// GET /api/jobs.

// jobsSection is the help page's account of the background work, for the
// owner. With nothing running (a test, a command), it is left out.
func (s *Server) jobsSection() string {
	all := s.app.Jobs.Status()
	if len(all) == 0 {
		return ""
	}
	now := s.now()
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="help-jobs"><h2 id="help-jobs">Work in the background</h2>
<p>What Sameway does on its own while it runs, whether or not a page is open.</p><ul class="sw-limits">`)
	for _, st := range all {
		b.WriteString("<li>" + template.HTMLEscapeString(s.jobWords(st, now)) + "</li>")
	}
	b.WriteString(`</ul></section>`)
	return b.String()
}

// jobWords is one job in a sentence or three: "Reminders ring: every 5
// seconds. Last ran today at 2:05pm, and went well. Next today at 2:05pm."
func (s *Server) jobWords(st runner.Status, now time.Time) string {
	h24 := s.h24()
	says := st.Name + ": "
	if st.Every > 0 {
		says += "every " + everyWords(st.Every) + "."
	} else {
		says += "runs all the time."
	}
	switch {
	case st.LastRun.IsZero():
		says += " Not run yet."
	case st.Every == 0:
		says += " Started " + lower(when.Ago(st.LastRun, now, h24)) + "."
	case st.Running:
		says += " Running now, since " + lower(when.Ago(st.LastRun, now, h24)) + "."
	default:
		says += " Last ran " + lower(when.Ago(st.LastRun, now, h24))
		if st.LastErr == "" {
			says += ", and went well."
		} else {
			says += ", and went wrong: " + strings.TrimSuffix(st.LastErr, ".") + "."
		}
	}
	if !st.NextRun.IsZero() && !st.Running {
		says += " Next " + lower(when.At(st.NextRun, now, h24)) + "."
	}
	return says
}

// everyWords is an interval as said: "5 seconds", "minute", "hour".
func everyWords(d time.Duration) string {
	for _, u := range []struct {
		d    time.Duration
		name string
	}{{24 * time.Hour, "day"}, {time.Hour, "hour"}, {time.Minute, "minute"}, {time.Second, "second"}} {
		if d >= u.d && d%u.d == 0 {
			if n := int(d / u.d); n > 1 {
				return fmt.Sprintf("%d %ss", n, u.name)
			}
			return u.name
		}
	}
	return d.String()
}

// lower starts words that begin a sentence elsewhere ("Today at 2pm")
// in the middle of one.
func lower(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// apiJobs is the background work as JSON, the owner's like the section:
// each job's name, every (in seconds), runs, running, last_run,
// last_error and next_run.
func (s *Server) apiJobs(w http.ResponseWriter, r *http.Request) {
	type job struct {
		runner.Status
		Every float64 `json:"every_seconds,omitempty"`
	}
	out := []job{}
	for _, st := range s.app.Jobs.Status() {
		out = append(out, job{Status: st, Every: st.Every.Seconds()})
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": out})
}
