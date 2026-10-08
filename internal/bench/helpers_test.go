package bench

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

func mustCreate(t *testing.T, a *app.App, typ string, f map[string]any) {
	if _, err := a.Store.Create(typ, f); err != nil {
		t.Fatal(err)
	}
}

func list(a *app.App, typ string) []*store.Record {
	r, _ := a.Store.List(typ, store.ListOptions{})
	return r
}

// find is the first record of a type whose field holds the words.
func find(a *app.App, typ, field, words string) *store.Record {
	for _, r := range list(a, typ) {
		if has(r, field, words) {
			return r
		}
	}
	return nil
}

func has(r *store.Record, field, words string) bool {
	return r != nil && strings.Contains(strings.ToLower(fmt.Sprint(r.Fields[field])), words)
}

// block says whether a block of the component (any, when "") whose props
// hold the words is on a canvas.
func block(a *app.App, component, words string) string {
	for _, b := range list(a, records.BlockType) {
		if b.Fields["component"] == "chat" {
			continue
		}
		if (component == "" || b.Fields["component"] == component) && strings.Contains(jsonOf(b.Fields["props"]), words) {
			return ""
		}
	}
	return fmt.Sprintf("no %s block with %s", component, words)
}

func jsonOf(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// next is the coming day with that weekday, as a person means it.
func next(wd time.Weekday) string {
	now := time.Now()
	d := (int(wd) - int(now.Weekday()) + 7) % 7
	if d == 0 {
		d = 7
	}
	return now.AddDate(0, 0, d).Format("2006-01-02")
}

func onDay(r *store.Record, day string) bool {
	for _, f := range []string{"due", "at", "starts"} {
		if v, ok := r.Fields[f].(string); ok && v != "" {
			if t, _, ok := when.Parse(v, time.Now()); ok && t.In(time.Local).Format("2006-01-02") == day {
				return true
			}
		}
	}
	return false
}

func hourIs(r *store.Record, field string, hour int) bool {
	v, _ := r.Fields[field].(string)
	t, _, ok := when.Parse(v, time.Now())
	return ok && t.In(time.Local).Hour() == hour
}

func want(ok bool, why string) string {
	if ok {
		return ""
	}
	return why
}

func all(whys ...string) string {
	var out []string
	for _, w := range whys {
		if w != "" {
			out = append(out, w)
		}
	}
	return strings.Join(out, "; ")
}

// day is a date the coming days are counted from: the weekday's next
// date, plus days.
func day(wd time.Weekday, plus int) string {
	d, _ := time.Parse("2006-01-02", next(wd))
	return d.AddDate(0, 0, plus).Format("2006-01-02")
}

// at is a time on that day as Sameway reads one without a zone.
func at(date string, hour, minute int) string {
	return fmt.Sprintf("%s %02d:%02d", date, hour, minute)
}

// timeIs says whether the field holds that hour and minute.
func timeIs(r *store.Record, field string, hour, minute int) bool {
	v, _ := r.Fields[field].(string)
	t, _, ok := when.Parse(v, time.Now())
	return ok && t.In(time.Local).Hour() == hour && t.In(time.Local).Minute() == minute
}

func tagged(r *store.Record, tag string) bool {
	return r != nil && strings.Contains(strings.ToLower(jsonOf(r.Fields["tags"])), `"`+tag+`"`)
}

// on says whether a block on the canvas named holds the words in its
// component or props.
func on(a *app.App, canvasID, words string) bool {
	for _, b := range list(a, records.BlockType) {
		if b.Fields["canvas"] == canvasID && strings.Contains(strings.ToLower(fmt.Sprint(b.Fields["component"])+jsonOf(b.Fields["props"])), words) {
			return true
		}
	}
	return false
}

func blockOf(t *testing.T, a *app.App, component string, props map[string]any) string {
	r, err := a.Store.Create(records.BlockType, a.Chat.BlockFields(map[string]any{"component": component, "props": props}))
	if err != nil {
		t.Fatal(err)
	}
	return r.ID
}

func num(v any) int {
	n, _ := strconv.Atoi(fmt.Sprint(v))
	return n
}
