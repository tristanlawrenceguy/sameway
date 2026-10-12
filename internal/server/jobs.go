package server

import (
	"context"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/runner"
)

// BackgroundJobs are the server's work while it runs, with or without a
// page open, for the app's runner (internal/runner) to do. Each keeps the
// interval it had on its own loop, or comes at the time set for it.
func (s *Server) BackgroundJobs() []runner.Job {
	jobs := []runner.Job{
		{Name: "Reminders ring", Every: 5 * time.Second, Run: func(_ context.Context, now time.Time) error {
			s.Ring(now) // ring.go
			return nil
		}},
		// Transcripts from Teams and Zoom: looked for every few seconds,
		// fetched at most every five minutes (media/meeting_fetch.go).
		{Name: "Meeting transcripts from Teams and Zoom", Every: 5 * time.Second, Run: func(_ context.Context, now time.Time) error {
			return s.FetchMeetings(now)
		}},
		{Name: "Calendars kept in step", Every: time.Hour, Run: s.keepCalendars},         // calendar_links.go
		{Name: "Daily copy in the cloud folder", Every: time.Hour, Run: s.keepCloudCopy}, // cloud_backup.go
		{Name: "Changes passed through the cloud folder", Every: 15 * time.Second, Wait: true, Run: func(context.Context, time.Time) error {
			return s.syncRound() // cloud_sync.go
		}},
		// At the times set on Help, read again as they change.
		{Name: "Morning brief", Next: s.briefNext, Run: func(_ context.Context, now time.Time) error {
			s.briefIfDue(now) // brief.go
			return nil
		}},
		{Name: "Weekly review ready", Next: s.reviewNext, Run: func(_ context.Context, now time.Time) error {
			s.reviewIfDue(now) // review_day.go
			return nil
		}},
		{Name: "Search by meaning kept current", Every: 2 * time.Minute, Run: func(ctx context.Context, _ time.Time) error {
			return s.refreshMeaning(ctx) // search_meaning.go: when an embedding model is here
		}},
		{Name: "Email read from the mailbox", Every: 5 * time.Minute, Wait: true, Run: func(ctx context.Context, _ time.Time) error {
			c, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			_, err := s.readMail(c) // mail_in.go: when connected
			return err
		}},
	}
	return append(jobs, s.WriteDownJobs()...) // media/hostwrite.go
}
