package chat

import (
	"context"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// How automations hear what happens (records.Listen) and the worker that
// runs them, a job on the app's runner (internal/runner); what they do
// with it is automate.go.

// AutomateJob listens for what happens to records and is the worker that
// runs the actions waiting for it, one at a time, until the runner stops.
func (s *Service) AutomateJob() runner.Job {
	s.listen()
	return runner.Job{Name: "Actions run when something happens", Run: func(ctx context.Context, _ time.Time) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case j := <-s.auto.jobs:
				s.runAutomation(ctx, j)
			}
		}
	}}
}

// StartAutomating is AutomateJob for a test, whose worker lasts as long
// as the test program does.
func (s *Service) StartAutomating() {
	if s.auto != nil {
		return
	}
	job := s.AutomateJob() // listening from here on
	go job.Run(context.Background(), time.Time{})
}

// listen has what happens to records heard, once.
func (s *Service) listen() {
	if s.auto != nil {
		return
	}
	s.auto = &automation{jobs: make(chan automationJob, 256), last: map[string]time.Time{}}
	records.Listen(s.Store, s.heard)
}

// heard is one change, logged or the system's own (records/listen.go):
// each record it wrote, as it was and as it is now, goes to the actions
// waiting for it and to the tags an action gave (judgement.go). Every way
// a record is written is heard here, from a page, the API, the
// assistant, an import or the content folder, since each goes through
// records.
func (s *Service) heard(_ string, c records.Change) {
	for _, op := range c.Ops {
		t, ok := s.Store.Types().Get(op.Type)
		if !ok {
			continue // a setting, or a type since gone
		}
		was, now := opRecord(op, op.Before), opRecord(op, op.After)
		s.recordChanged(t, was, now)
		s.tagsChanged(t, was, now)
	}
}

func opRecord(op records.Op, fields map[string]any) *store.Record {
	if fields == nil {
		return nil
	}
	return &store.Record{ID: op.ID, Type: op.Type, Fields: fields}
}
