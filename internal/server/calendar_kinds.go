package server

import (
	"net/url"
	"slices"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A calendar of several kinds (type: all, or types) narrowed to one: a row of
// links, All, Tasks, Reminders, Events, each with how many it has in the
// month shown (the filters component as links, one choice among a few).
// It is offered only when the month has more than one kind. The choice is
// in the page's address, named after the block (c-<block>-type=task), so
// two calendars keep their own, and the months and days either side keep
// it. It only narrows: a calendar of one type is never offered another.

// calendarKinds narrows a calendar of everything to the kind picked in the
// page's address and fills its filter: the kinds in the month with their
// counts, All first. Each event carries its kind, as everyEvent sets it.
func calendarKinds(out map[string]any, at *collectionPlace, block string) {
	if at == nil || at.Path == "" || block == "" {
		return
	}
	events, _ := out["events"].([]any)
	month, _ := out["month"].(string)
	param := "c-" + block + "-type"
	counts := map[string]int{}
	inMonth := 0
	for _, e := range events {
		ev, _ := e.(map[string]any)
		kind, _ := ev["kind"].(string)
		date, _ := ev["date"].(string)
		if kind == "" || !strings.HasPrefix(date, month) {
			continue
		}
		counts[kind]++
		inMonth++
	}
	picked := at.Query.Get(param)
	if hasKind := slices.ContainsFunc(events, func(e any) bool {
		ev, _ := e.(map[string]any)
		return picked != "" && ev["kind"] == picked
	}); !hasKind {
		picked = "" // a kind the calendar does not have narrows nothing
	}
	if picked != "" {
		kept := make([]any, 0, len(events))
		for _, e := range events {
			if ev, _ := e.(map[string]any); ev["kind"] == picked {
				kept = append(kept, e)
			}
		}
		out["events"] = kept
		// What is shown is that kind alone, so what it goes out as is too.
		out["types"] = []any{picked}
		keepKind(out, param, picked)
	}
	if len(counts) < 2 && picked == "" {
		return
	}
	var names []string
	for name := range counts {
		if name != picked {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	if picked != "" {
		names = append([]string{picked}, names...)
	}
	anchor := ""
	if !at.Own {
		anchor = "#block-" + block
	}
	opts := []any{map[string]any{"label": "All", "count": inMonth, "href": withKind(at, param, "") + anchor, "selected": picked == ""}}
	for _, name := range names {
		opts = append(opts, map[string]any{"label": capitalize(schema.Plural(name)), "count": counts[name], "href": withKind(at, param, name) + anchor, "selected": name == picked})
	}
	label := "Kinds of event"
	if c, _ := out["caption"].(string); c != "" {
		label += ": " + c
	}
	out["filter"] = map[string]any{"shape": "links", "label": label + ", " + when.Month(month), "choices": []any{map[string]any{"label": "Kind", "options": opts}}}
}

// withKind is the page's address with this calendar's kind set, or taken
// away for All, the rest as it is.
func withKind(at *collectionPlace, param, kind string) string {
	q := url.Values{}
	for k, v := range at.Query {
		q[k] = v
	}
	q.Del(param)
	if kind != "" {
		q.Set(param, kind)
	}
	if len(q) == 0 {
		return at.Path
	}
	return at.Path + "?" + q.Encode()
}

// keepKind carries the kind into the links to the months and days either
// side, and into each day number, so moving through time keeps it.
func keepKind(out map[string]any, param, kind string) {
	add := func(href string) string {
		u, err := url.Parse(href)
		if err != nil {
			return href
		}
		q := u.Query()
		q.Set(param, kind)
		u.RawQuery = q.Encode()
		return u.String()
	}
	if nav, ok := out["nav"].(map[string]any); ok {
		for _, link := range nav {
			if l, ok := link.(map[string]any); ok {
				if h, _ := l["href"].(string); h != "" {
					l["href"] = add(h)
				}
			}
		}
	}
	if base, _ := out["dayBase"].(string); strings.HasSuffix(base, "?day=") {
		out["dayBase"] = strings.TrimSuffix(base, "day=") + url.QueryEscape(param) + "=" + url.QueryEscape(kind) + "&day="
	}
}
