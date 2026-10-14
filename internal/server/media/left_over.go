package media

import (
	"context"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A file is "Being read" only while this Sameway reads it: the converter,
// the parts of a recording and the background browser are all work in
// this program, and none of it outlives it. A file left converting when
// Sameway closed said "Being read" and "Its text appears here when the
// converter answers" for good, a state named for a moment shown days
// after it, as "Ringing" was on a reminder (crew finding 0727). When the
// workspace opens, each is put right: a recording waits to be written
// down again, which the sweep does once speech-to-text is here; any other
// file says it was not read and how to have it read.

// leftOverJob puts right, once at the start, the files left converting
// when Sameway last closed: those last changed before opened.
func (s *Service) leftOverJob(opened time.Time) runner.Job {
	return runner.Job{Name: "Files left half read when Sameway closed", Run: func(context.Context, time.Time) error {
		s.leftOver(opened)
		return nil
	}}
}

// leftOver is that, for files last changed before opened.
func (s *Service) leftOver(opened time.Time) {
	recs, err := s.app.Store.List(records.FileType, store.ListOptions{})
	if err != nil {
		return
	}
	put := false
	for _, rec := range recs {
		if rec.Fields["status"] != "converting" || !rec.UpdatedAt.Before(opened) {
			continue
		}
		if isRecording(rec) {
			s.fileSays(rec.ID, map[string]any{"status": "ready", "note": ""})
		} else {
			s.fileSays(rec.ID, map[string]any{"status": "failed", "note": "Sameway closed before it was read. Add the file again to read it."})
		}
		put = true
	}
	if put {
		s.Changed()
	}
}
