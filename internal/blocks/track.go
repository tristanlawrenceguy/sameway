package blocks

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// The tracker block: the habits, where each stands. The arithmetic is in
// internal/track; this reads the habit and entry records and puts the
// numbers where a person looks. The Log press and a habit's own page are
// the server's (server/track.go).

// TrackerComponent is the habits block.
const TrackerComponent = "tracker"

// HabitOf reads a habit record.
func HabitOf(rec *store.Record) track.Habit {
	h := track.Habit{ID: rec.ID}
	h.Name, _ = rec.Fields["name"].(string)
	h.Unit, _ = rec.Fields["unit"].(string)
	h.Cadence, _ = rec.Fields["cadence"].(string)
	h.Aim, _ = rec.Fields["aim"].(string)
	h.Combine, _ = rec.Fields["combine"].(string)
	h.Target = Number(rec.Fields["target"])
	h.Goal = Number(rec.Fields["goal"])
	return h
}

// Number reads a number from a field, whatever JSON or a form made of it.
func Number(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f
	}
	return 0
}

// EntriesOf reads the entries logged against a habit.
func EntriesOf(st *store.Store, habitID string) []track.Entry {
	recs, err := st.List(records.EntryType, store.ListOptions{})
	if err != nil {
		return nil
	}
	var out []track.Entry
	for _, rec := range recs {
		if h, _ := rec.Fields["habit"].(string); h != habitID {
			continue
		}
		v, _ := rec.Fields["at"].(string)
		ts, err := time.Parse(time.RFC3339, v)
		if err != nil {
			continue
		}
		if strings.HasSuffix(v, "T00:00:00Z") {
			// A whole day is kept as midnight UTC on its date; it is that
			// date here, wherever here is.
			ts = time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.Local)
		}
		amount := Number(rec.Fields["amount"])
		if amount == 0 && rec.Fields["amount"] == nil {
			amount = 1
		}
		out = append(out, track.Entry{At: ts, Amount: amount})
	}
	return out
}

// Standing is a habit as the tracker shows it.
func Standing(st *store.Store, rec *store.Record, now time.Time) map[string]any {
	h := track.Normal(HabitOf(rec))
	sum := track.Summarise(h, EntriesOf(st, h.ID), now, map[string]int{"day": 7, "week": 4, "month": 6, "year": 3}[h.Cadence])
	pct := 0
	if sum.Target > 0 {
		pct = int(math.Min(100, math.Round(sum.Now/sum.Target*100)))
	}
	item := map[string]any{
		"id": h.ID, "name": h.Name, "shortName": trim.Title(h.Name), "href": "/t/" + records.HabitType + "/" + h.ID, "unit": h.Unit, "period": h.Cadence, "aim": h.Aim,
		"amount": sum.Now, "target": sum.Target, "progress": track.Progress(h, sum), "pct": pct, "met": sum.Met,
		"streak": sum.Streak, "streakWords": track.StreakWords(sum.Streak, h), "best": sum.Best,
	}
	if h.Goal > 0 {
		goal := track.Amount(sum.Total, "") + " of " + track.Amount(h.Goal, h.Unit)
		if by, _ := rec.Fields["by"].(string); by != "" {
			if t, err := time.Parse(time.RFC3339, by); err == nil {
				goal += " by " + t.Local().Format(dayFormat(t, now))
			}
		}
		item["goal"] = goal
	}
	// A reading starts Log at the last one, so the same again is one press.
	if h.Aim == track.Record {
		for i := len(sum.Last) - 1; i >= 0; i-- {
			if sum.Last[i].Logged {
				item["log"] = strconv.FormatFloat(sum.Last[i].Amount, 'f', -1, 64)
				break
			}
		}
	}
	last := make([]any, 0, len(sum.Last))
	for i, p := range sum.Last {
		going := i == len(sum.Last)-1
		last = append(last, map[string]any{"date": periodName(p.Start, h.Cadence), "met": p.Met, "amount": p.Amount, "words": periodWords(h, p, going, sum), "logged": p.Logged, "going": going && !p.Met})
	}
	item["last"] = last
	item["when"] = periodWhen(now, h.Cadence)
	return item
}

// resolveTracker fills a tracker block from the habits: those named in
// habits, by name or id, in that order, or else all that are not
// archived; of them, those with one of the tags asked for.
func resolveTracker(w *Workspace, props map[string]any, _ Place) map[string]any {
	out := copyProps(props)
	tags := Strs(props["tags"])
	named := Strs(props["habits"]) // names or ids; objects are the server's own
	habits := []any{}
	// Today when every habit is daily; this period when they differ.
	out["when"] = "today"
	if _, ok := w.Store.Types().Get(records.HabitType); ok {
		recs, _ := w.Store.List(records.HabitType, store.ListOptions{OrderBy: "created_at"})
		if len(named) > 0 {
			var problem string
			if recs, problem = pickHabits(recs, named); problem != "" {
				out["habits"], out["problem"] = []any{}, problem
				return out
			}
		}
		now := time.Now()
		for _, rec := range recs {
			if archived, _ := rec.Fields["archived"].(bool); archived && len(named) == 0 {
				continue
			}
			if len(tags) > 0 && !hasTag(rec, tags) {
				continue
			}
			item := Standing(w.Store, rec, now)
			if item["period"] != "day" {
				out["when"] = "this period"
			}
			habits = append(habits, item)
		}
	}
	out["habits"] = habits
	if len(habits) == 0 && len(tags) > 0 {
		if have := habitTags(w.Store); len(have) > 0 {
			out["problem"] = "no habit is tagged " + strings.Join(tags, " or ") + "; the habits have " + strings.Join(have, ", ")
		}
	}
	return out
}

// dayFormat writes a day as a person would: the year only when it is not
// this one.
func dayFormat(t, now time.Time) string {
	if t.Year() == now.Year() {
		return "2 Jan"
	}
	return "2 Jan 2006"
}
