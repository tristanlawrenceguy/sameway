package chat

import (
	"fmt"
	"slices"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Longer writing is a piece made of parts, as a book is of chapters: each
// part says what it is part of, and the piece keeps their order, so moving
// one is one change. Beside the writing, never in it, is its material:
// guidelines, submission details, research, for the whole piece or for one
// part. A workspace made before these fields has them added the first time
// a piece is organised.

func writingFields(typeName string) []schema.Field {
	return []schema.Field{
		{Name: records.Synopsis, Type: "text", Description: "What it is about, in a line, for the outline."},
		{Name: records.PartOf, Type: "ref", To: typeName, Label: "Part of", Description: "The longer piece it is part of, as a chapter is of a book."},
		{Name: records.PartsOrder, Type: "list", Of: "string", Hidden: true},
		{Name: records.Aim, Type: "int", Label: "Words to aim for", Description: "How many words the whole piece should come to."},
		{Name: records.MaterialFor, Type: "ref", To: typeName, Label: "Material for", Description: "What it is background for, such as guidelines or research; never part of it."},
	}
}

type materialItem struct {
	ID  string `json:"id"`
	For string `json:"for"`
}

var organiseOps = []Op{{Tool: llm.Tool{Name: "organise_writing",
	Description: "Organise longer writing: a piece made of parts in order, as a book of chapters, and the material that goes with it (guidelines, submission details, research) for the whole or for one part. Give the piece and its parts in reading order; parts it has already and you leave out stay, after them. One change, undone in one go. Fields the type lacks for this are added first. Make the parts with create_record before.",
	Schema: obj(map[string]any{
		"type":  map[string]any{"type": "string", "description": "The content type; note when left out."},
		"piece": map[string]any{"type": "string", "description": "The id of the whole piece."},
		"parts": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Ids of its parts, in reading order."},
		"material": map[string]any{"type": "array", "description": "Records that are material for it.", "items": obj(map[string]any{
			"id":  map[string]any{"type": "string", "description": "The record's id."},
			"for": map[string]any{"type": "string", "description": "The part it is for; the whole piece when left out."},
		}, "id")},
	}, "piece")}, Offered: func(s *Service, _ *llm.Tool) bool { return len(records.TypeNames(s.Store)) > 0 }}}

func (s *Service) organiseWriting(typeName, pieceID string, parts []string, material []materialItem) toolResult {
	if typeName == "" {
		typeName = "note"
	}
	t, err := records.ContentType(s.Store, typeName)
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
	var batch []records.BatchItem
	write := func(id string, fields map[string]any) error {
		was, err := s.Store.Get(t.Name, id)
		if err != nil {
			return fmt.Errorf("no %s %s; find_records to get its id", t.Name, id)
		}
		if _, _, err := records.Write(s.Store, "updated", t.Name, id, fields); err != nil {
			return err
		}
		batch = append(batch, records.BatchItem{Type: t.Name, ID: id, Before: was.Fields})
		return nil
	}
	for _, id := range parts {
		if id == pieceID || slices.Contains(records.PiecesAbove(s.Store, t, piece), id) {
			return fail("%s cannot be a part of %s, which is inside it", id, pieceID)
		}
	}
	order := append([]string{}, parts...)
	for _, p := range records.Parts(s.Store, t, piece) {
		if !slices.Contains(order, p.ID) {
			order = append(order, p.ID)
		}
	}
	for _, id := range parts {
		if err := write(id, map[string]any{records.PartOf: pieceID}); err != nil {
			return fail("nothing organised: %v", err)
		}
	}
	if err := write(pieceID, map[string]any{records.PartsOrder: order}); err != nil {
		return fail("nothing organised: %v", err)
	}
	for _, m := range material {
		to := m.For
		if to == "" {
			to = pieceID
		}
		if err := write(m.ID, map[string]any{records.MaterialFor: to}); err != nil {
			return fail("the parts are in order, but not the material: %v", err)
		}
	}
	title := records.Name(s.Store, t, piece)
	// The link opens what was just made, the outline and the material,
	// since that is what the person asked to see.
	href := "/t/" + t.Name + "/" + pieceID + "?show=outline"
	if len(material) > 0 {
		href += "&show=material"
	}
	c := records.Change{Action: "organised", Component: t.Name, ID: pieceID, Href: href,
		Detail: fmt.Sprintf("%s, %d parts and %d material", title, len(order), len(material)), Before: records.Batch(batch)}
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
