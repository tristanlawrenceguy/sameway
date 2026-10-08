package blocks

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Two records with one title make two controls with one name: two boxes
// "Done Call plumber" in one list, two links "Call plumber" to different
// pages, two "Undo created task Call plumber" in the log. A person moving
// by links or controls, or an agent finding one by its role and name,
// cannot tell them apart (WCAG 2.4.6 and 2.4.9; a Playwright getByRole
// that matches two refuses to guess). So where a list shows records whose
// titles repeat, each of the repeated ones is told apart by the first fact
// that differs: the day it is due, then when it was added, to the minute
// and then the second, and last by where it comes: the first of 3. An id
// was once the last resort, and read to a person as a string of letters.
// Only when needed: a title that is its own stays as it is, and the words
// are read after the name, never shown twice.

// Apart gives each of names the words that tell it from the others with
// the same name, or "" where the name is its own. ways(i) are the words
// item i could be told apart by, best first; the first way whose words
// differ across every item sharing the name is used for all of them. A
// way may leave one item without words, which is then the plain one.
//
// rank, when given, orders the ones that nothing tells apart for their
// number, first to last: when each was added, so a note is "the first of
// 2" in its list and in the log alike, whichever order each shows them in.
func Apart(names []string, ways func(i int) []string, rank ...func(i int) string) []string {
	out := make([]string, len(names))
	groups := map[string][]int{}
	var order []string
	for i, n := range names {
		k := strings.ToLower(strings.Join(strings.Fields(n), " "))
		if k == "" {
			continue // nameless, or not shown with the others
		}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], i)
	}
	for _, k := range order {
		g := groups[k]
		if len(g) < 2 {
			continue
		}
		cands := make([][]string, len(g))
		most := 0
		for j, i := range g {
			cands[j] = ways(i)
			most = max(most, len(cands[j]))
		}
		found := false
		for w := 0; w < most && !found; w++ {
			seen, differ := map[string]bool{}, true
			for _, c := range cands {
				v := ""
				if w < len(c) {
					v = c[w]
				}
				if seen[v] {
					differ = false
					break
				}
				seen[v] = true
			}
			if !differ {
				continue
			}
			for j, i := range g {
				if w < len(cands[j]) {
					out[i] = cands[j][w]
				}
			}
			found = true
		}
		// Nothing about them differs that a person would say: they are told
		// apart by where each comes, as they are listed.
		if !found {
			ordered := append([]int(nil), g...)
			if len(rank) > 0 {
				sort.SliceStable(ordered, func(x, y int) bool { return rank[0](ordered[x]) < rank[0](ordered[y]) })
			}
			for j, i := range ordered {
				out[i] = ordinal(j+1) + " of " + fmt.Sprint(len(g))
			}
		}
	}
	return out
}

// RecordWays is what tells one record from another with its title: the
// day that matters to it (due tomorrow), when it was added, its id.
func RecordWays(t *schema.Type, rec *store.Record) []string {
	return []string{dayWords(t, rec), "added " + MomentWords(rec.CreatedAt), "added " + SecondWords(rec.CreatedAt)}
}

// SecondWords is a moment to the second, for two added in one minute.
func SecondWords(at time.Time) string {
	return shortDay(at.Local()) + ", " + at.Local().Format("15:04:05")
}

// ordinal is a place in a list in words: first, second, … 11th.
func ordinal(n int) string {
	words := []string{"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth"}
	if n < len(words) {
		return words[n]
	}
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		suffix = map[int]string{1: "st", 2: "nd", 3: "rd"}[n%10]
		if suffix == "" {
			suffix = "th"
		}
	}
	return fmt.Sprint(n) + suffix
}

// RecordsApart is apart for one type's records shown together, by id.
func (ws *Workspace) RecordsApart(t *schema.Type, recs []*store.Record) map[string]string {
	names := make([]string, len(recs))
	for i, rec := range recs {
		names[i] = ws.title(t, rec)
	}
	said := Apart(names, func(i int) []string { return RecordWays(t, recs[i]) }, func(i int) string { return AddedRank(recs[i]) })
	out := map[string]string{}
	for i, rec := range recs {
		if said[i] != "" {
			out[rec.ID] = said[i]
		}
	}
	return out
}

// WithContext is a name and the words that tell it apart: Call plumber
// (due Fri 25 Sep). In brackets, because a browser puts a space before
// words hidden in a span of their own, so a comma would be heard alone.
func WithContext(name, context string) string {
	if context == "" {
		return name
	}
	return name + " (" + context + ")"
}

// markApart names a list item's mark after its record and the words that
// tell it apart, so the box reads Done Call plumber, due Fri 25 Sep.
func markApart(actions []any, context string) {
	if context == "" {
		return
	}
	for _, a := range actions {
		if m, ok := a.(map[string]any); ok && m["component"] == "mark" {
			if p, ok := m["props"].(map[string]any); ok {
				p["context"] = WithContext(str(p["context"], ""), context)
			}
		}
	}
}

// eventsApart tells apart a calendar's events with one label, two called
// Call plumber, by their kind, their day and time, then by the record
// they lead to.
// Only those in the month shown are met together; month "" is them all.
func eventsApart(events []any, month string) []any {
	names := make([]string, len(events))
	for i, e := range events {
		ev, _ := e.(map[string]any)
		names[i] = str(ev["label"], "")
		if !strings.HasPrefix(str(ev["date"], ""), month) {
			names[i] = "" // not met here
		}
	}
	told := Apart(names, func(i int) []string {
		ev, _ := events[i].(map[string]any)
		on := ""
		if d, err := time.Parse("2006-01-02", str(ev["date"], "")); err == nil {
			on = "on " + shortDay(d)
			if at := str(ev["time"], ""); at != "" {
				on += ", " + at
			}
		}
		// On a calendar of several kinds a task and a reminder of one name
		// are told apart by their kind first, which is not said otherwise.
		kind := schema.Words(str(ev["kind"], ""))
		both := on
		if kind != "" {
			both = kind + ", " + on
		}
		return []string{kind, on, both}
	})
	for i, e := range events {
		if ev, ok := e.(map[string]any); ok && told[i] != "" {
			ev["context"] = told[i]
			actions, _ := ev["actions"].([]any)
			markApart(actions, told[i])
		}
	}
	return events
}

// AddedRank orders records by when they were added, for their number.
func AddedRank(rec *store.Record) string {
	return rec.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000000") + " " + rec.ID
}
