package server

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Tracking: the tracker block, the Log press, and the habit's own page.
// The arithmetic is in internal/track; this reads the habit and entry
// records and puts the numbers where a person looks.

const (
	trackerComponent = "tracker"
	HabitType        = "habit"
	EntryType        = "entry"
)

// habitOf reads a habit record.
func habitOf(rec *store.Record) track.Habit {
	h := track.Habit{ID: rec.ID}
	h.Name, _ = rec.Fields["name"].(string)
	h.Unit, _ = rec.Fields["unit"].(string)
	h.Cadence, _ = rec.Fields["cadence"].(string)
	h.Aim, _ = rec.Fields["aim"].(string)
	h.Combine, _ = rec.Fields["combine"].(string)
	h.Target = number(rec.Fields["target"])
	h.Goal = number(rec.Fields["goal"])
	return h
}

func number(v any) float64 {
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

// entriesOf reads the entries logged against a habit.
func (s *Server) entriesOf(habitID string) []track.Entry {
	recs, err := s.app.Store.List(EntryType, store.ListOptions{})
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
		amount := number(rec.Fields["amount"])
		if amount == 0 && rec.Fields["amount"] == nil {
			amount = 1
		}
		out = append(out, track.Entry{At: ts, Amount: amount})
	}
	return out
}

// standing is a habit as the tracker shows it.
func (s *Server) standing(rec *store.Record, now time.Time) map[string]any {
	h := track.Normal(habitOf(rec))
	sum := track.Summarise(h, s.entriesOf(h.ID), now, map[string]int{"day": 7, "week": 4, "month": 6, "year": 3}[h.Cadence])
	pct := 0
	if sum.Target > 0 {
		pct = int(math.Min(100, math.Round(sum.Now/sum.Target*100)))
	}
	item := map[string]any{
		"id": h.ID, "name": h.Name, "href": "/t/" + HabitType + "/" + h.ID, "unit": h.Unit, "period": h.Cadence, "aim": h.Aim,
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
	last := make([]any, 0, len(sum.Last))
	for _, p := range sum.Last {
		last = append(last, map[string]any{"date": track.PeriodLabel(p.Start, h.Cadence), "met": p.Met, "amount": p.Amount, "words": periodWords(h, p), "logged": p.Logged})
	}
	item["last"] = last
	return item
}

// periodWords is how a period went, for a reader of the dots: met or not
// met, within or over a limit, or what was recorded.
func periodWords(h track.Habit, p track.Period) string {
	switch h.Aim {
	case track.Record:
		if !p.Logged {
			return "nothing logged"
		}
		return track.Amount(p.Amount, h.Unit)
	case track.Limit:
		if p.Met {
			return track.Amount(p.Amount, h.Unit) + ", within the limit"
		}
		if p.Amount > h.Target {
			return track.Amount(p.Amount, h.Unit) + ", over the limit"
		}
		return "not tracked yet"
	}
	return ""
}

func firstOf(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// resolveTracker fills a tracker block from the habits, all that are not
// archived, or those with one of the tags asked for.
func (s *Server) resolveTracker(props map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	tags := strs(props["tags"])
	habits := []any{}
	if _, ok := s.app.Types.Get(HabitType); ok {
		recs, _ := s.app.Store.List(HabitType, store.ListOptions{OrderBy: "created_at"})
		now := time.Now()
		for _, rec := range recs {
			if archived, _ := rec.Fields["archived"].(bool); archived {
				continue
			}
			if len(tags) > 0 && !hasTag(rec, tags) {
				continue
			}
			habits = append(habits, s.standing(rec, now))
		}
	}
	out["habits"] = habits
	return out
}

func hasTag(rec *store.Record, tags []string) bool {
	have, _ := rec.Fields["tags"].([]any)
	for _, t := range have {
		for _, want := range tags {
			if s, _ := t.(string); strings.EqualFold(s, want) {
				return true
			}
		}
	}
	return false
}

// habitLog is the Log press: an entry for the amount given or one, now,
// or on the day given when it was done before.
func (s *Server) habitLog(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(HabitType, r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	r.ParseForm()
	h := habitOf(rec)
	amount := 1.0
	if v := strings.TrimSpace(r.PostForm.Get("amount")); v != "" {
		if amount, err = strconv.ParseFloat(strings.ReplaceAll(v, ",", "."), 64); err != nil || amount <= 0 {
			s.app.Chat.Notice("An amount to log is a number above zero, such as 1 or 2.5.")
			http.Redirect(w, r, backFrom(r), http.StatusSeeOther)
			return
		}
	}
	at := when.Store(time.Now(), false)
	if v := strings.TrimSpace(r.PostForm.Get("on")); v != "" {
		now := time.Now()
		t, day, ok := when.Parse(v, now)
		if !ok {
			s.app.Chat.Notice("The day it was done reads as a date, such as " + now.AddDate(0, 0, -1).Format("2006-01-02") + " or yesterday.")
			http.Redirect(w, r, backFrom(r), http.StatusSeeOther)
			return
		}
		if !day || t.Format("2006-01-02") != now.Format("2006-01-02") {
			at = when.Store(t, day)
		}
	}
	fields := map[string]any{"habit": h.ID, "at": at, "amount": amount}
	if note := strings.TrimSpace(r.PostForm.Get("note")); note != "" {
		fields["note"] = note
	}
	entry, err := s.app.Store.Create(EntryType, fields)
	if err != nil {
		s.app.Chat.Notice(err.Error())
	} else {
		s.record(r, chat.Change{Action: "logged", Component: HabitType, ID: h.ID, Detail: h.Name + ": " + track.Amount(amount, h.Unit), Href: "/t/" + HabitType + "/" + h.ID, Before: map[string]any{"entry": entry.ID}})
	}
	http.Redirect(w, r, backFrom(r), http.StatusSeeOther)
}

// habitSection is what a habit's own page adds: where it stands, its own
// tracker row with a day to log for, and the last periods as a chart with
// the target (or the limit) drawn.
func (s *Server) habitSection(rec *store.Record) template.HTML {
	now := time.Now()
	h := track.Normal(habitOf(rec))
	item := s.standing(rec, now)
	item["dated"] = true
	var b strings.Builder
	b.WriteString(`<section class="sw-stack sw-habit" aria-labelledby="habit-standing"><h2 id="habit-standing">Keeping up</h2>`)
	b.WriteString(string(s.component(trackerComponent, map[string]any{"label": h.Name, "habits": []any{item}})))
	best, _ := item["best"].(int)
	if best > 0 {
		fmt.Fprintf(&b, `<p class="sw-muted sw-small">Best run: %s.</p>`, template.HTMLEscapeString(track.StreakWords(best, h)))
	}
	n := map[string]int{"day": 30, "week": 12, "month": 12, "year": 5}[h.Cadence]
	sum := track.Summarise(h, s.entriesOf(h.ID), now, n)
	series := make([]any, 0, len(sum.Last))
	for _, p := range sum.Last {
		series = append(series, map[string]any{"label": track.PeriodLabel(p.Start, h.Cadence), "value": p.Amount})
	}
	kind := "bar"
	if h.Combine != track.Sum {
		kind = "line"
	}
	props := map[string]any{"caption": fmt.Sprintf("The last %d %ss", n, h.Cadence), "series": series, "kind": kind}
	if h.Aim != track.Record {
		props["target"] = sum.Target
	}
	if h.Aim == track.Limit {
		props["targetLabel"] = "limit"
	}
	if h.Unit != "" {
		props["unit"] = h.Unit
	}
	b.WriteString(string(s.component("chart", props)))
	b.WriteString(`</section>`)
	return template.HTML(b.String())
}

// dayFormat writes a day as a person would: the year only when it is not
// this one.
func dayFormat(t, now time.Time) string {
	if t.Year() == now.Year() {
		return "2 Jan"
	}
	return "2 Jan 2006"
}
