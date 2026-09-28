package chat

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An action can run on its own: every hour, every day at a time, or every
// week on a day at a time. That is how a weather block stays current and
// how the assistant can plan a morning before the person sits down. The
// clock is the server's; each run is in the log under the system's name,
// and the action remembers when it last ran so a restart does not repeat
// it.

// Due says whether an action should run now, given when it last ran.
func Due(rec *store.Record, now time.Time) bool {
	every, _ := rec.Fields["every"].(string)
	last := lastRun(rec)
	switch every {
	case "hour":
		return now.Sub(last) >= time.Hour
	case "day", "week":
		at, _ := rec.Fields["at"].(string)
		h, m := clock(at)
		if every == "week" {
			on, _ := rec.Fields["on"].(string)
			if !strings.EqualFold(now.Weekday().String(), on) {
				return false
			}
		}
		if now.Hour() < h || (now.Hour() == h && now.Minute() < m) {
			return false
		}
		y1, m1, d1 := now.Date()
		y2, m2, d2 := last.In(now.Location()).Date()
		return !(y1 == y2 && m1 == m2 && d1 == d2)
	}
	return false
}

func lastRun(rec *store.Record) time.Time {
	s, _ := rec.Fields["last_run"].(string)
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// clock reads HH:MM; anything else means the start of the day.
func clock(at string) (int, int) {
	var h, m int
	if t, err := time.Parse("15:04", strings.TrimSpace(at)); err == nil {
		h, m = t.Hour(), t.Minute()
	}
	return h, m
}

// RunDue runs every action whose time has come, as the system, and
// records when. It returns the ids it ran.
func (s *Service) RunDue(ctx context.Context, now time.Time) []string {
	if _, ok := s.Store.Types().Get(ActionType); !ok {
		return nil
	}
	recs, err := s.Store.List(ActionType, store.ListOptions{})
	if err != nil {
		return nil
	}
	var ran []string
	for _, rec := range recs {
		if !Due(rec, now) {
			continue
		}
		// Stamp first, so a slow or failing run is not tried again every
		// minute; the log says what happened either way.
		s.Store.Update(ActionType, rec.ID, map[string]any{"last_run": now.UTC().Format(time.RFC3339)})
		if _, _, err := s.RunAs(ctx, "system", rec.ID, ""); err != nil {
			Record(s.Store, "system", Change{Action: "failed", Detail: "scheduled action: " + err.Error()})
		}
		ran = append(ran, rec.ID)
	}
	return ran
}

// StartSchedule checks once a minute, until ctx ends, for actions whose
// time has come. The server starts it; tests call RunDue directly.
func (s *Service) StartSchedule(ctx context.Context) {
	go func() {
		tick := time.NewTicker(time.Minute)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-tick.C:
				if ran := s.RunDue(ctx, now); len(ran) > 0 {
					log.Printf("schedule: ran %s", strings.Join(ran, ", "))
				}
			}
		}
	}()
}
