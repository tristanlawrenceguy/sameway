package chat

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Longer writing is a piece made of parts, as a book is of chapters: each
// part says what it is part of, and the piece keeps their order, so moving
// one is one change. Beside the writing, never in it, is its material:
// guidelines, submission details, research, for the whole piece or for one
// part. A workspace made before these fields has them added the first time
// a piece is organised.

// Writing fields, by name, as the starter's note has them.
const (
	PartOf      = "part_of"
	PartsOrder  = "parts"
	MaterialFor = "material_for"
	Synopsis    = "synopsis"
	Aim         = "aim"
)

func writingFields(typeName string) []schema.Field {
	return []schema.Field{
		{Name: Synopsis, Type: "text", Description: "What it is about, in a line, for the outline."},
		{Name: PartOf, Type: "ref", To: typeName, Label: "Part of", Description: "The longer piece it is part of, as a chapter is of a book."},
		{Name: PartsOrder, Type: "list", Of: "string", Hidden: true},
		{Name: Aim, Type: "int", Label: "Words to aim for", Description: "How many words the whole piece should come to."},
		{Name: MaterialFor, Type: "ref", To: typeName, Label: "Material for", Description: "What it is background for, such as guidelines or research; never part of it."},
	}
}

// Organised says whether a type can be organised into pieces and parts.
func Organised(t *schema.Type) bool {
	f, ok := t.Field(PartOf)
	return ok && f.Type == "ref" && f.To == t.Name
}

type materialItem struct {
	ID  string `json:"id"`
	For string `json:"for"`
}

func (s *Service) organiseTools() []llm.Tool {
	if len(s.typeNames()) == 0 {
		return nil
	}
	return []llm.Tool{{Name: "organise_writing",
		Description: "Organise longer writing: a piece made of parts in order, as a book of chapters, and the material that goes with it (guidelines, submission details, research) for the whole or for one part. Give the piece and its parts in reading order; parts it has already and you leave out stay, after them. One change, undone in one go. Fields the type lacks for this are added first. Make the parts with create_record before.",
		Schema: obj(map[string]any{
			"type":  map[string]any{"type": "string", "description": "The content type; note when left out."},
			"piece": map[string]any{"type": "string", "description": "The id of the whole piece."},
			"parts": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Ids of its parts, in reading order."},
			"material": map[string]any{"type": "array", "description": "Records that are material for it.", "items": obj(map[string]any{
				"id":  map[string]any{"type": "string", "description": "The record's id."},
				"for": map[string]any{"type": "string", "description": "The part it is for; the whole piece when left out."},
			}, "id")},
		}, "piece")}}
}

func (s *Service) organiseWriting(typeName, pieceID string, parts []string, material []materialItem) toolResult {
	if typeName == "" {
		typeName = "note"
	}
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	piece, err := s.Store.Get(t.Name, pieceID)
	if err != nil {
		return fail("no %s %s to organise; find_records to get its id", t.Name, pieceID)
	}
	if t, err = s.writingType(t); err != nil {
		return fail("%v", err)
	}
	var batch []BatchItem
	write := func(id string, fields map[string]any) error {
		was, err := s.Store.Get(t.Name, id)
		if err != nil {
			return fmt.Errorf("no %s %s; find_records to get its id", t.Name, id)
		}
		if _, _, err := Write(s.Store, "updated", t.Name, id, fields); err != nil {
			return err
		}
		batch = append(batch, BatchItem{Type: t.Name, ID: id, Before: was.Fields})
		return nil
	}
	for _, id := range parts {
		if id == pieceID || slices.Contains(piecesAbove(s.Store, t, piece), id) {
			return fail("%s cannot be a part of %s, which is inside it", id, pieceID)
		}
	}
	order := append([]string{}, parts...)
	for _, p := range Parts(s.Store, t, piece) {
		if !slices.Contains(order, p.ID) {
			order = append(order, p.ID)
		}
	}
	for _, id := range parts {
		if err := write(id, map[string]any{PartOf: pieceID}); err != nil {
			return fail("nothing organised: %v", err)
		}
	}
	if err := write(pieceID, map[string]any{PartsOrder: order}); err != nil {
		return fail("nothing organised: %v", err)
	}
	for _, m := range material {
		to := m.For
		if to == "" {
			to = pieceID
		}
		if err := write(m.ID, map[string]any{MaterialFor: to}); err != nil {
			return fail("the parts are in order, but not the material: %v", err)
		}
	}
	title := Name(s.Store, t, piece)
	// The link opens what was just made, the outline and the material,
	// since that is what the person asked to see.
	href := "/t/" + t.Name + "/" + pieceID + "?show=outline"
	if len(material) > 0 {
		href += "&show=material"
	}
	c := Change{Action: "organised", Component: t.Name, ID: pieceID, Href: href,
		Detail: fmt.Sprintf("%s, %d parts and %d material", title, len(order), len(material)), Before: Batch(batch)}
	return toolResult{text: fmt.Sprintf("organised %s %s: %d parts in order and %d records of material; %s opens its page with the outline; send it, and say so.",
		t.Name, title, len(order), len(material), c.Href), change: &c}
}

// writingType is the type with every writing field, adding those it lacks.
func (s *Service) writingType(t *schema.Type) (*schema.Type, error) {
	for _, f := range writingFields(t.Name) {
		if _, has := t.Field(f.Name); has {
			continue
		}
		if s.AddField == nil {
			return nil, fmt.Errorf("%s has no field %s and this workspace cannot add one from here", t.Name, f.Name)
		}
		var err error
		if t, err = s.AddField(t.Name, f); err != nil {
			return nil, fmt.Errorf("could not add %s to %s: %v", f.Name, t.Name, err)
		}
	}
	return t, nil
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

// piecesAbove are the ids of the pieces a record is inside, nearest first.
func piecesAbove(st *store.Store, t *schema.Type, rec *store.Record) []string {
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
	for _, id := range piecesAbove(st, t, rec) {
		if m := of(id); len(m) > 0 {
			above[id] = m
		}
	}
	return mine, above
}

// WordCount counts the words in text.
func WordCount(text string) int { return len(strings.Fields(text)) }
