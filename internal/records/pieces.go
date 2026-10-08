package records

import (
	"slices"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Longer writing is a piece made of parts in order, as a book is of
// chapters, with the material kept for it: the fields that say so, and
// reading them back, for the page, the API and the assistant alike.

// Writing fields, by name, as the starter's note has them.
const (
	PartOf      = "part_of"
	PartsOrder  = "parts"
	MaterialFor = "material_for"
	Synopsis    = "synopsis"
	Aim         = "aim"
)

// Organised says whether a type can be organised into pieces and parts.
func Organised(t *schema.Type) bool {
	f, ok := t.Field(PartOf)
	return ok && f.Type == "ref" && f.To == t.Name
}

// Parts are a piece's parts in reading order: as the piece keeps them,
// then any it does not name yet, oldest first.
func Parts(st *store.Store, t *schema.Type, piece *store.Record) []*store.Record {
	if !Organised(t) {
		return nil
	}
	recs, _ := query.Filter(st, t, []string{PartOf + "=" + piece.ID}, "created_at", 0, time.Now())
	order, _ := piece.Fields[PartsOrder].([]any)
	rank := func(r *store.Record) int {
		for i, id := range order {
			if id == r.ID {
				return i
			}
		}
		return len(order)
	}
	slices.SortStableFunc(recs, func(a, b *store.Record) int { return rank(a) - rank(b) })
	return recs
}

// PieceOf is the piece a part belongs to, or nil.
func PieceOf(st *store.Store, t *schema.Type, rec *store.Record) *store.Record {
	if !Organised(t) {
		return nil
	}
	id, _ := rec.Fields[PartOf].(string)
	if id == "" || id == rec.ID {
		return nil
	}
	piece, err := st.Get(t.Name, id)
	if err != nil {
		return nil
	}
	return piece
}

// PiecesAbove are the ids of the pieces a record is inside, nearest first.
func PiecesAbove(st *store.Store, t *schema.Type, rec *store.Record) []string {
	var out []string
	for p := PieceOf(st, t, rec); p != nil && !slices.Contains(out, p.ID) && len(out) < 20; p = PieceOf(st, t, p) {
		out = append(out, p.ID)
	}
	return out
}

// Material is what is kept for a record (material_for points at it),
// then what is kept for each piece it is inside, each with what it is for.
func Material(st *store.Store, t *schema.Type, rec *store.Record) (mine []*store.Record, above map[string][]*store.Record) {
	if _, ok := t.Field(MaterialFor); !ok {
		return nil, nil
	}
	of := func(id string) []*store.Record {
		recs, _ := query.Filter(st, t, []string{MaterialFor + "=" + id}, "created_at", 0, time.Now())
		return recs
	}
	mine, above = of(rec.ID), map[string][]*store.Record{}
	for _, id := range PiecesAbove(st, t, rec) {
		if m := of(id); len(m) > 0 {
			above[id] = m
		}
	}
	return mine, above
}

// WordCount counts the words in text.
func WordCount(text string) int { return len(strings.Fields(text)) }
