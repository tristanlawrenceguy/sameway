package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Tracking: the Log press, and the habit's own page. The tracker block
// is internal/blocks' (blocks/track.go); the arithmetic is in
// internal/track.

// noGoal says a habit's goal is none: its schema has the goal empty for
// none, and a 0 there, as a form or an agent may leave it, means the same,
// so a habit's page does not say Goal 0.
func noGoal(typ, field string, v any) bool {
	return typ == HabitType && field == "goal" && blocks.Number(v) == 0
}

// habitLog is the Log press: an entry for the amount given or one, now,
// or on the day given when it was done before.
func (s *Server) habitLog(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(HabitType, r.PathValue("id"))
	if err != nil {
		s.failed(w, r, "Not logged", err, "/")
		return
	}
	r.ParseForm()
	h := blocks.HabitOf(rec)
	amount := 1.0
	if v := strings.TrimSpace(r.PostForm.Get("amount")); v != "" {
		if amount, err = strconv.ParseFloat(strings.ReplaceAll(v, ",", "."), 64); err != nil || amount <= 0 {
			s.tell(w, r, outcome{Failed: true, Title: h.Name + " not logged", Problems: []problem{{Field: "log-" + h.ID, Text: "An amount to log is a number above zero, such as 1 or 2.5."}}}, "/")
			return
		}
	}
	at := when.Store(time.Now(), false)
	if v := strings.TrimSpace(r.PostForm.Get("on")); v != "" {
		now := time.Now()
		t, day, ok := when.Parse(v, now)
		if !ok {
			s.tell(w, r, outcome{Failed: true, Title: h.Name + " not logged", Problems: []problem{{Field: "on-" + h.ID, Text: "The day it was done reads as a date, such as " + now.AddDate(0, 0, -1).Format("2006-01-02") + " or yesterday."}}}, "/")
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
		s.failed(w, r, "Not logged", err, "/")
		return
	}
	undo := s.record(r, records.Change{Action: "logged", Component: HabitType, ID: h.ID, Detail: h.Name + ": " + track.Amount(amount, h.Unit), Href: "/t/" + HabitType + "/" + h.ID, Before: map[string]any{"entry": entry.ID}})
	// Said, with where it stands now and its Undo: "Water: 1 glass logged.
	// Now 6 of 8 glasses."
	sum := track.Summarise(track.Normal(h), blocks.EntriesOf(s.app.Store, h.ID), time.Now(), 1)
	s.tellAt(w, r, outcome{Title: h.Name + ": " + track.Amount(amount, h.Unit) + " logged.", Text: "Now " + track.Progress(track.Normal(h), sum) + ".", Undo: undo, Of: h.Name + " " + track.Amount(amount, h.Unit)}, backFrom(r))
}

// habitSection is what a habit's own page adds: where it stands, its own
// tracker row with a day to log for, and the last periods as a chart with
// the target (or the limit) drawn.
func (s *Server) habitSection(rec *store.Record) template.HTML {
	now := time.Now()
	h := track.Normal(blocks.HabitOf(rec))
	item := blocks.Standing(s.app.Store, rec, now)
	item["dated"] = true
	var b strings.Builder
	// The heading names the part of the page; the tracker under it is the
	// region, called Keeping up, and shows no heading of its own.
	b.WriteString(`<div class="sw-stack sw-habit"><h2 id="habit-standing">Keeping up</h2>`)
	b.WriteString(string(s.component(trackerComponent, map[string]any{"habits": []any{item}})))
	best, _ := item["best"].(int)
	if best > 0 {
		fmt.Fprintf(&b, `<p class="sw-muted sw-small">Best run: %s.</p>`, template.HTMLEscapeString(track.StreakWords(best, h)))
	}
	n := map[string]int{"day": 30, "week": 12, "month": 12, "year": 5}[h.Cadence]
	sum := track.Summarise(h, blocks.EntriesOf(s.app.Store, h.ID), now, n)
	series := make([]any, 0, len(sum.Last))
	for _, p := range sum.Last {
		series = append(series, map[string]any{"label": track.PeriodLabel(p.Start, h.Cadence), "value": p.Amount})
	}
	kind := "bar"
	if h.Combine != track.Sum {
		kind = "line"
	}
	// The numbers table heads its columns with what they are: the period,
	// and the unit or Amount.
	value := "Amount"
	if h.Unit != "" {
		value = capitalize(h.Unit)
	}
	props := map[string]any{"caption": fmt.Sprintf("The last %d %ss", n, h.Cadence), "series": series, "kind": kind, "groupLabel": capitalize(h.Cadence), "valueLabel": value}
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
	b.WriteString(`</div>`)
	return template.HTML(b.String())
}
