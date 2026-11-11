package automate

import (
	"context"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A picture is read once and its words kept: a photo of a letter shared
// as IMG_2041 was read each time it was sorted, and kept nothing, so the
// note stayed IMG_2041, search did not find the letter, and the picture
// had no words for someone who cannot see it. Read here, its text is the
// file's, its description too when it has none, and a note named after
// it takes the name of what it is.

const pictureSystem = `You read a picture a person kept: a photo of a letter, a bill, a notice, a receipt, a screen, or anything else.
Answer with JSON only: {"name": "what it is, a short title as the person would name it, in the picture's language (\"School trip letter\", \"Water bill for November\")", "description": "one or two sentences of what it shows, for someone who cannot see it", "text": "the words in it, as Markdown, in order; empty when it has none"}`

type pictureRead struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Text        string `json:"text"`
}

// readPictures reads the pictures a record links to that are not read
// yet, and renames the record when it is named after one. It answers
// with the renaming, for the change that logs it.
func (s *Service) readPictures(ctx context.Context, t *schema.Type, rec *store.Record) []records.Op {
	if s.Pictures == nil {
		return nil
	}
	var ops []records.Op
	for _, id := range linkedFiles(rec) {
		f, err := s.Store.Get(records.FileType, id)
		if err != nil || strings.TrimSpace(str(f.Fields["text"])+str(f.Fields["description"])) != "" {
			continue
		}
		pics := s.Pictures(id) // a picture, or a scan's pages
		var got pictureRead
		if len(pics) == 0 || s.ask(ctx, pictureSystem, "The picture, named "+str(f.Fields["title"])+"; each page of it when there are more.", pics, &got) != nil {
			continue // a model that cannot see: sorting asks from the words
		}
		kept := map[string]any{"text": strings.TrimSpace(got.Text)}
		if f.Fields["kind"] == "pdf" { // a scan: read now, not "nothing to read"
			kept["status"], kept["note"] = "ready", "Its pages are pictures; the assistant read their words."
		}
		if str(f.Fields["description"]) == "" && got.Description != "" {
			kept["description"] = strings.TrimSpace(got.Description)
		}
		records.ApplyOps(s.Store, records.Op{Type: records.FileType, ID: id, After: kept}) // a file read, as keeping it does
		if name := strings.TrimSpace(got.Name); name != "" && t.Title != "" && str(rec.Fields[t.Title]) == str(f.Fields["title"]) {
			if done, err := s.Apply(records.Op{Type: t.Name, ID: rec.ID, After: map[string]any{t.Title: name}}); err == nil {
				ops = append(ops, done...)
				rec.Fields[t.Title] = name
			}
		}
	}
	return ops
}

// linkedFiles are the files a record's words link to, up to three.
func linkedFiles(rec *store.Record) []string {
	var ids []string
	seen := map[string]bool{}
	for _, v := range rec.Fields {
		for _, m := range fileLink.FindAllStringSubmatch(str(v), -1) {
			if len(ids) < picturesSeen && !seen[m[1]] {
				seen[m[1]] = true
				ids = append(ids, m[1])
			}
		}
	}
	return ids
}

func str(v any) string { s, _ := v.(string); return s }
