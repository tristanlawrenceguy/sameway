package search

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A search covers everything, and what it found can be narrowed to one
// kind: the page does it with a row of links, an agent with a type. Both
// check the kind here and count from here, so a person and an agent
// narrowing the same words get the same things, the same counts, and the
// same no to a kind there is none of.

// Searchable says whether a search can be narrowed to the kind name: one a
// person has a page of. What the system keeps for itself, a message or a
// canvas block, is not, and one the person has hidden is not either.
func Searchable(types *schema.Set, name string) (*schema.Type, bool) {
	t, ok := types.Get(name)
	if !ok || t.Internal || t.Hidden || Skip[name] {
		return nil, false
	}
	return t, true
}

// Kinds is every kind a search can be narrowed to, by name.
func Kinds(types *schema.Set) []string {
	var names []string
	for _, t := range types.Types {
		if _, ok := Searchable(types, t.Name); ok {
			names = append(names, t.Name)
		}
	}
	sort.Strings(names)
	return names
}

// Refusal is what an agent is told when it narrows to a kind it cannot:
// said as a no, never as nothing found, so it does not conclude a thing
// is not there. It names the kinds it can use.
func Refusal(only string, kinds []string) string {
	can := "there are none to narrow to"
	if len(kinds) > 0 {
		can = "the kinds are: " + strings.Join(kinds, ", ")
	}
	return fmt.Sprintf("There is no kind of thing called “%s” to search in; %s. Leave out type to search everything.", only, can)
}

// Result is one search as an agent has it: how many were found of each
// kind, the kind shown if one was asked for, and one page of those.
type Result struct {
	Query string `json:"query"`
	Type  string `json:"type,omitempty"`
	// Total is everything found; Counts is the same by kind.
	Total  int            `json:"total"`
	Counts map[string]int `json:"counts"`
	// Found is how many of the kind shown, or Total when none was asked for.
	Found int `json:"found"`
	Page  int `json:"page"`
	Pages int `json:"pages"`
	// From is where this page starts among those found, from 0.
	From int   `json:"-"`
	Hits []Hit `json:"hits"`
}

// Narrow is the hits of everything, counted by kind, narrowed to only when
// named, and the page asked for of Limit each. A page past the end is the
// last, as on the search page.
func Narrow(all []Hit, q, only string, page int) Result {
	hits := all
	if only != "" {
		hits = Of(all, only)
	}
	pages := max((len(hits)+Limit-1)/Limit, 1)
	page = min(max(page, 1), pages)
	lo, hi := (page-1)*Limit, min(page*Limit, len(hits))
	shown := append([]Hit{}, hits[lo:hi]...)
	return Result{Query: strings.TrimSpace(q), Type: only, Total: len(all), Counts: Counts(all),
		Found: len(hits), Page: page, Pages: pages, From: lo, Hits: shown}
}

// Said is what the search found, in words: 12 found: 7 notes, 3 tasks,
// 2 blocks; or, narrowed, Showing tasks only: 3 of 12 found (7 notes,
// 2 blocks elsewhere); then, when there is more than one page, which of
// them these are and how to ask for the next.
func (r Result) Said() string {
	var b strings.Builder
	switch {
	case r.Total == 0:
		return "Nothing found"
	case r.Type == "":
		fmt.Fprintf(&b, "%d found: %s", r.Total, counted(r.Counts, ""))
	default:
		fmt.Fprintf(&b, "Showing %s only: %d of %d found", label(r.Type, 2), r.Found, r.Total)
		if rest := counted(r.Counts, r.Type); rest != "" {
			fmt.Fprintf(&b, " (%s elsewhere)", rest)
		}
	}
	if r.Pages > 1 {
		fmt.Fprintf(&b, ". Showing %d–%d of %d", r.From+1, r.From+len(r.Hits), r.Found)
		if r.Page < r.Pages {
			fmt.Fprintf(&b, "; ask for page %d for more", r.Page+1)
		} else {
			fmt.Fprintf(&b, ", the last of %d pages", r.Pages)
		}
	}
	return b.String()
}

// counted is how many of each kind, the most first, leaving out except:
// 7 notes, 3 tasks, 1 block.
func counted(counts map[string]int, except string) string {
	var names []string
	for name := range counts {
		if name != except {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool {
		if counts[names[i]] != counts[names[j]] {
			return counts[names[i]] > counts[names[j]]
		}
		return names[i] < names[j]
	})
	parts := make([]string, len(names))
	for i, name := range names {
		parts[i] = fmt.Sprintf("%d %s", counts[name], label(name, counts[name]))
	}
	return strings.Join(parts, ", ")
}

// label is a kind as a person reads it, one or many: note, notes.
func label(name string, n int) string {
	name = strings.ReplaceAll(name, "_", " ")
	if n == 1 {
		return name
	}
	return schema.Plural(name)
}
