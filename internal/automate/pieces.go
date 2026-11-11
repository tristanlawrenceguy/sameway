package automate

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Pieces shared one after another are one thing: a letter photographed
// page by page and sent as each was taken, a long message pasted in
// parts, screenshots of one conversation, the front and back of a card.
// Each came as its own note, read, named and suggested from alone, so
// page two's amount never met page one's deadline. When something is
// shared, the model is shown what was shared in the half hour before;
// when it says this continues one, the earlier note's words come first
// in this one and the earlier note goes, undone with the sorting.

const piecesWithin = 30 * time.Minute

const piecesSystem = `A person shares things from their phone one at a time, as they come to hand. Say whether what they shared just now is a further piece of something they shared shortly before, so that together they are one thing: the next page of the same letter or document, the rest of a message cut into parts, another screenshot of the same conversation, the back of the same card or receipt, more photos of the same thing, the rest of the same list or recipe.
Things that are each whole on their own are apart, even on one subject or from one sender: two bills, two letters, two links, a photo and an unrelated note.
Answer with JSON only: {"continues": the number of the earlier one it continues, or 0 when none, "why": "a few words"}`

// joinPieces joins a share to the earlier one it continues, as the model
// judges, and answers with the joining, for the change that logs it.
func (s *Service) joinPieces(ctx context.Context, t *schema.Type, rec *store.Record) ([]records.Op, string) {
	if !hasTag(rec, "shared") {
		return nil, ""
	}
	earlier := s.earlierShares(t, rec)
	if len(earlier) == 0 {
		return nil, ""
	}
	var q strings.Builder
	q.WriteString("Shared just now:\n" + clipEnds(recordWords(s.Store, t, rec), 1500) + "\n\nShared before it, newest first:")
	for i, e := range earlier {
		fmt.Fprintf(&q, "\n\n%d. Shared %s before:\n%s", i+1, ago(rec.CreatedAt.Sub(e.CreatedAt)), clipEnds(recordWords(s.Store, t, e), 800))
	}
	var got struct {
		Continues any    `json:"continues"`
		Why       string `json:"why"`
	}
	if s.askJSON(ctx, piecesSystem, q.String(), &got) != nil {
		return nil, ""
	}
	n, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(got.Continues)))
	if n < 1 || n > len(earlier) {
		return nil, ""
	}
	a := earlier[n-1]
	joined := map[string]any{}
	for _, f := range t.Fields {
		before, _ := a.Fields[f.Name].(string)
		now, _ := rec.Fields[f.Name].(string)
		switch {
		case f.Name == t.Title && before != "":
			joined[f.Name] = before // named for the whole when its first piece was
		case f.Name != t.Title && (f.Type == "text" || f.Type == "markdown") && strings.TrimSpace(before) != "":
			joined[f.Name] = strings.TrimSpace(before + "\n\n" + now)
		}
	}
	ops := []records.Op{{Type: t.Name, ID: rec.ID, After: joined}, {Type: t.Name, ID: a.ID}}
	// What was suggested from the first piece alone is asked again from
	// the whole.
	props, _ := s.Store.List(records.ProposalType, store.ListOptions{})
	for _, p := range props {
		if p.Fields["from"] == t.Name+"/"+a.ID && p.Fields["state"] == "pending" {
			ops = append(ops, records.Op{Type: records.ProposalType, ID: p.ID, After: map[string]any{"state": "dismissed"}})
		}
	}
	name := records.Name(s.Store, t, a)
	done, err := s.Apply(ops...)
	if err != nil {
		return nil, ""
	}
	for k, v := range joined {
		rec.Fields[k] = v
	}
	return done, "joined to " + name + ", shared before it (" + strings.TrimSpace(got.Why) + ")"
}

// earlierShares are the shares made in the half hour before this one,
// newest first, a few at most.
func (s *Service) earlierShares(t *schema.Type, rec *store.Record) []*store.Record {
	all, _ := s.Store.List(t.Name, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 50})
	var out []*store.Record
	for _, e := range all {
		gap := rec.CreatedAt.Sub(e.CreatedAt)
		if e.CreatedAt.Before(rec.CreatedAt) && gap <= piecesWithin && hasTag(e, "shared") && len(out) < 5 {
			out = append(out, e)
		}
	}
	return out
}

func hasTag(rec *store.Record, tag string) bool {
	for _, t := range records.StringList(rec.Fields["tags"]) {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func ago(d time.Duration) string {
	if m := int(d.Round(time.Minute) / time.Minute); m > 1 {
		return fmt.Sprintf("%d minutes", m)
	}
	return "a minute or less"
}
