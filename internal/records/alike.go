package records

import (
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/fold"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Records alike: the same kind of thing, still open, called the same. A
// workspace fills with them when something is asked for again by someone
// who did not see it was there: six tasks Buy groceries, two habits Water.
// Whether two are one thing twice or two things is a judgement, so this
// only finds them; the assistant is told when it makes one (create_record),
// and a list offers to keep one of those nothing tells apart (Twins).

// NameKey is a name as two names are compared: in any case and form, its
// spaces one (fold.Key). A list tells such names apart (blocks.Apart) and
// this finds them.
func NameKey(s string) string { return fold.Key(s) }

// alikeKinds says whether a type's records can be one thing twice. An
// email, a file or a device is a copy of something elsewhere, and two
// entries are two times a habit was done: none of them is ever merged.
func alikeKinds(t *schema.Type) bool {
	switch t.Name {
	case EmailType, FileType, "device", EntryType:
		return false
	}
	return t.Content() && t.Title != ""
}

// nameOf is a record's own name, from its title field: a record with
// none is like nothing.
func nameOf(t *schema.Type, rec *store.Record) string {
	s, _ := rec.Fields[t.Title].(string)
	return NameKey(s)
}

// Alike is every open record of t called as rec is, other than rec,
// newest first. A done one is not alike: buying groceries again next week
// is not buying them twice.
func Alike(st *store.Store, t *schema.Type, rec *store.Record) []*store.Record {
	if !alikeKinds(t) || nameOf(t, rec) == "" {
		return nil
	}
	var out []*store.Record
	for _, g := range alikeGroups(st, t, false) {
		if nameOf(t, g[0]) != nameOf(t, rec) {
			continue
		}
		for _, r := range g {
			if r.ID != rec.ID {
				out = append(out, r)
			}
		}
	}
	return out
}

// Twins is t's open records that share a name and nothing else tells
// apart: no day, or the same day. Each group is newest first, the largest
// group first. These are the ones a list offers to keep one of.
func Twins(st *store.Store, t *schema.Type) [][]*store.Record {
	if !alikeKinds(t) {
		return nil
	}
	return alikeGroups(st, t, true)
}

// alikeGroups groups t's open records by name, and with sameDay by their
// day too, keeping the groups of two or more.
func alikeGroups(st *store.Store, t *schema.Type, sameDay bool) [][]*store.Record {
	recs, err := st.List(t.Name, store.ListOptions{OrderBy: "created_at", Desc: true})
	if err != nil {
		return nil
	}
	day := t.DayField()
	groups := map[string][]*store.Record{}
	var order []string
	for _, r := range recs {
		name := nameOf(t, r)
		if name == "" || t.Done(r.Fields) {
			continue
		}
		key := name
		if sameDay && day != "" {
			d, _ := r.Fields[day].(string)
			key += "\x00" + strings.SplitN(d, "T", 2)[0]
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], r)
	}
	var out [][]*store.Record
	for _, k := range order {
		if len(groups[k]) > 1 {
			out = append(out, groups[k])
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}

// TwinsOf is the others in rec's group of Twins, newest first: what a
// list merges into rec when it keeps rec.
func TwinsOf(st *store.Store, t *schema.Type, rec *store.Record) []*store.Record {
	for _, g := range Twins(st, t) {
		for _, r := range g {
			if r.ID != rec.ID {
				continue
			}
			var others []*store.Record
			for _, o := range g {
				if o.ID != rec.ID {
					others = append(others, o)
				}
			}
			return others
		}
	}
	return nil
}
