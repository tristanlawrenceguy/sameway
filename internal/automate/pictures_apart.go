package automate

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Photos picked together from a phone's gallery are not always one
// thing: twelve from a day out are, and so are a letter's pages, but five
// receipts for expenses or three different letters are each their own.
// Once they are read, the model says which belong together and names
// each set. The note takes the name of the set its first photo is in;
// each other set becomes a note of its own, sorted and suggested apart,
// saying what it was shared with. Undoing the sorting puts them back.

const groupSystem = `A person shared several photos together from their phone's gallery. From what each shows, say which belong together as one thing and which are separate things.
One thing: the pages of one letter or document, photos of one day, one place, one event or one object. Separate things: different receipts, different letters or bills, unrelated photos.
Answer with JSON only: {"groups": [{"photos": [1, 2], "name": "what that set is, a short title as the person would name it, in the photos' language (\"Day at Brighton beach\", \"Receipt from Wickes\")"}]}, every photo in exactly one group, the groups in the order the photos came. Give one group when they are all one thing.`

type pictureGroup struct {
	Photos []any  `json:"photos"`
	Name   string `json:"name"`
}

// groupPictures names the photos a record links to as a whole, and for
// a share of separate things, moves each other set to a note of its own.
func (s *Service) groupPictures(ctx context.Context, t *schema.Type, rec *store.Record, ids []string) []records.Op {
	var q strings.Builder
	titles := map[string]string{}
	for k, id := range ids {
		f, err := s.Store.Get(records.FileType, id)
		if err != nil {
			return nil
		}
		titles[id] = str(f.Fields["title"])
		fmt.Fprintf(&q, "Photo %d, named %s: %s\n%s\n\n", k+1, titles[id], str(f.Fields["description"]), clipEnds(str(f.Fields["text"]), 300))
	}
	var got struct {
		Groups []pictureGroup `json:"groups"`
	}
	if s.askJSON(ctx, groupSystem, q.String(), &got) != nil {
		return nil
	}
	sets := groupsOf(got.Groups, len(ids))
	if len(sets) == 0 {
		return nil
	}
	var ops []records.Op
	first := sets[0]
	if name := strings.TrimSpace(first.name); name != "" && t.Title != "" && str(rec.Fields[t.Title]) == titles[ids[0]] {
		ops = append(ops, records.Op{Type: t.Name, ID: rec.ID, After: map[string]any{t.Title: name}})
	}
	if len(sets) > 1 && hasTag(rec, "shared") {
		left := map[string]any{}
		for _, f := range t.Fields {
			if v, ok := rec.Fields[f.Name].(string); ok && f.Name != t.Title {
				left[f.Name] = v
			}
		}
		body := bodyField(t)
		for _, set := range sets[1:] {
			var links []string
			for _, k := range set.photos {
				id := ids[k]
				links = append(links, "["+titles[id]+"](/t/"+records.FileType+"/"+id+")")
				for name, v := range left {
					left[name] = strings.TrimSpace(regexp.MustCompile(`\[[^\]]*\]\(/t/`+records.FileType+`/`+id+`\)\s*`).ReplaceAllString(v.(string), ""))
				}
			}
			name := strings.TrimSpace(set.name)
			if name == "" {
				name = titles[ids[set.photos[0]]]
			}
			fields := map[string]any{t.Title: name, "tags": rec.Fields["tags"]}
			if body != "" {
				fields[body] = strings.Join(links, "\n\n") + "\n\nShared together with [" + str(rec.Fields[t.Title]) + "](/t/" + t.Name + "/" + rec.ID + ")."
			}
			ops = append(ops, records.Op{Type: t.Name, After: fields})
		}
		if len(ops) > 0 && ops[0].ID == rec.ID {
			for k, v := range left {
				ops[0].After[k] = v
			}
		} else {
			ops = append([]records.Op{{Type: t.Name, ID: rec.ID, After: left}}, ops...)
		}
	}
	if len(ops) == 0 {
		return nil
	}
	done, err := s.Apply(ops...)
	if err != nil {
		return nil
	}
	for k, v := range ops[0].After {
		rec.Fields[k] = v
	}
	return done
}

type photoSet struct {
	photos []int // from 0
	name   string
}

// groupsOf is the model's groups when every photo is in exactly one,
// else none: a grouping that loses or doubles a photo is not followed.
func groupsOf(groups []pictureGroup, n int) []photoSet {
	seen := make([]bool, n)
	var sets []photoSet
	for _, g := range groups {
		var set photoSet
		for _, p := range g.Photos {
			k, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(p)))
			if err != nil || k < 1 || k > n || seen[k-1] {
				return nil
			}
			seen[k-1] = true
			set.photos = append(set.photos, k-1)
		}
		if len(set.photos) > 0 {
			set.name = g.Name
			sets = append(sets, set)
		}
	}
	for _, ok := range seen {
		if !ok {
			return nil
		}
	}
	// The record keeps the set its first photo is in.
	for i, set := range sets {
		if set.photos[0] == 0 && i > 0 {
			sets[0], sets[i] = sets[i], sets[0]
		}
	}
	return sets
}

// bodyField is where a kind keeps its words: the first long text field.
func bodyField(t *schema.Type) string {
	for _, f := range t.Fields {
		if f.Name != t.Title && (f.Type == "text" || f.Type == "markdown") {
			return f.Name
		}
	}
	return ""
}
