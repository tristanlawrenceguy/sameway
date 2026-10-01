package server

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A meeting is an event with its recording: its page plays the recording
// with its transcript, and until it is written up offers to have the
// assistant write it up (chat/meeting.go). A recording with a transcript
// that no meeting has offers the same. Both are parts of the page
// (parts.go), recording and write-up, off until the assistant sees a
// reason or the person keeps them on.

// meetingExtras is what an event's page shows of its recording.
func (s *Server) meetingExtras(r *http.Request, rec *store.Record) string {
	id, _ := rec.Fields["recording"].(string)
	if id == "" {
		return ""
	}
	file, err := s.app.Store.Get(FileType, id)
	if err != nil || !isRecording(file) {
		return ""
	}
	_, here := s.shown(r)
	page := "/t/" + chat.EventType + "/" + rec.ID
	var b strings.Builder
	if summary, _ := rec.Fields["summary"].(string); strings.TrimSpace(summary) == "" && len(s.heard(file)) > 0 && changes(r) && s.showing(r, WriteUpPart) {
		b.WriteString(writeUpOffer("Write up this meeting (/t/"+chat.EventType+"/"+rec.ID+") from its recording (/t/"+FileType+"/"+id+")", "Ask the assistant to write it up", s))
		b.WriteString(s.fewer(page, WriteUpPart, "the offer to write it up", here))
	}
	if s.showing(r, RecordingPart) {
		b.WriteString(string(s.component("media", s.recordingOf(file))))
		b.WriteString(s.fewer(page, RecordingPart, "the recording", here))
	}
	return b.String()
}

// recordingOffer is a recording's page offering a write-up, when it has a
// transcript and no meeting has it yet.
func (s *Server) recordingOffer(r *http.Request, file *store.Record) string {
	if !changes(r) || !s.showing(r, WriteUpPart) || len(s.heard(file)) == 0 {
		return ""
	}
	t, ok := s.app.Types.Get(chat.EventType)
	if !ok {
		return ""
	}
	if _, has := t.Field("recording"); !has {
		return ""
	}
	if had, _ := query.Filter(s.app.Store, t, []string{"recording=" + file.ID}, "", 1, time.Now()); len(had) > 0 {
		return ""
	}
	return writeUpOffer("Write up the meeting in this recording (/t/"+FileType+"/"+file.ID+")", "Write up the meeting", s)
}

func writeUpOffer(ask, label string, s *Server) string {
	return `<p class="sw-muted">The assistant writes up a meeting from what was said: a summary, what was decided and the tasks that came up, each linked to where it was said, for you to check. One Undo takes it back.</p><p>` +
		string(s.component("link", map[string]any{"href": "/chat?prompt=" + url.QueryEscape(ask+": a short summary, what was decided and the tasks that came up."), "label": label, "look": "button"})) + `</p>`
}

// changes says whether whoever asked may change the workspace.
func changes(r *http.Request) bool {
	a := chat.VisitorOf(r.Context()).Access
	return a != chat.View && a != chat.Public
}
