// Package media is files, recordings and meetings: a file kept and read,
// a picture and a recording on their pages, a recording written down on
// the computer that hosts the workspace and its speakers told apart, and
// transcripts brought from Teams and Zoom. The server holds one Service
// and adds its Routes to the one route table.
package media

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Deps is what media needs of the server: what every handler does
// (web.Deps), and a few things of its own.
type Deps interface {
	web.Deps
	// Title is a record's name, as its page says it.
	Title(t *schema.Type, rec *store.Record) string
	// Importable says whether records of a kind can be read from a file.
	Importable(t *schema.Type) bool
	// Shown and Fewer are the parts of a page that are off until there is
	// a reason (server/parts.go): which this view shows, and the way back
	// out of one the address opened.
	Shown(r *http.Request) (always, here []string)
	Fewer(page, key, what string, here []string) string
	// Handler is the whole server, for reading a recording's sound in a
	// browser on this computer with no page open (hostwrite.go).
	Handler() http.Handler
}

// Service is media for one workspace: what is being written down, how,
// and the meeting apps signed in to.
type Service struct {
	Deps
	app    *app.App
	speech speechState // speech-to-text on this computer; transcribe.go
	host   hostState   // recordings written down with no page; hostwrite.go
	apps   meetingApps // transcripts brought from Teams and Zoom; meeting_fetch.go
}

// New is media for the workspace the server serves.
func New(d Deps) *Service { return &Service{Deps: d, app: d.App()} }

// The parts of a meeting's page that are off until there is a reason
// (server/parts.go): its recording played on its page, and the offer to
// write it up from the transcript.
const (
	RecordingPart = "recording"
	WriteUpPart   = "write-up"
)

// SpeechGiven says whether speech-to-text was set by UseSpeech rather than
// downloaded, as a test does.
func (s *Service) SpeechGiven() bool { return s.speech.given }

// str is a field's text, or fallback.
func str(v any, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}
