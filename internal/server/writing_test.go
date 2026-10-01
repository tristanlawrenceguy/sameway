package server_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A piece made of parts reads as one: its page has the outline in order
// with words against its aim, each part says where it is with the parts
// either side and the piece's material, a part moves as one change, and
// Read it all gives the whole, as a page, Markdown or Word.
func TestLongerWritingIsOrganisedOnItsPages(t *testing.T) {
	a, h := newApp(t)
	mk := func(fields map[string]any) string {
		r, err := a.Store.Create("note", fields)
		if err != nil {
			t.Fatal(err)
		}
		return r.ID
	}
	book := mk(map[string]any{"title": "The Pond", "body": "How we made a pond."})
	one := mk(map[string]any{"title": "Digging", "body": "We dug for a week."})
	two := mk(map[string]any{"title": "Lining", "body": "The liner came late."})
	three := mk(map[string]any{"title": "Filling", "body": "Rain did the rest."})
	guide := mk(map[string]any{"title": "Submission guidelines", "body": "Under 5,000 words."})
	args, _ := json.Marshal(map[string]any{"piece": book, "parts": []string{one, two, three}, "material": []any{map[string]any{"id": guide}}})
	if text, isErr := a.Chat.Call("organise_writing", args); isErr {
		t.Fatal(text)
	}
	// A note has none of this until it is organised; then it has it all.
	if _, err := a.Store.Update("note", book, map[string]any{"aim": 20}); err != nil {
		t.Fatal(err)
	}
	a.Store.Update("note", one, map[string]any{"synopsis": "The hole."})

	if rest := get(t, h, "/t/note/"+book).Body.String(); strings.Contains(rest, `id="parts"`) || strings.Contains(rest, `id="material"`) {
		t.Error("at rest the piece's page is the piece: its outline and material are parts, off until asked")
	}
	page := get(t, h, "/t/note/"+book+"?show=outline&show=material").Body.String()
	for _, want := range []string{`<h2 id="parts">Parts</h2>`, "The hole.", "18 of 20 words", "Read it all", "Submission guidelines"} {
		if !strings.Contains(page, want) {
			t.Errorf("the piece's page should have %q", want)
		}
	}
	if strings.Index(page, "Digging") > strings.Index(page, "Lining") {
		t.Error("the outline is in reading order")
	}
	part := get(t, h, "/t/note/"+two+"?show=place&show=material").Body.String()
	for _, want := range []string{"Part 2 of 3 of", "Before: Digging", "Next: Filling", "For all of The Pond"} {
		if !strings.Contains(part, want) {
			t.Errorf("a part's page should have %q", want)
		}
	}

	postForm(t, h, "/t/note/"+book+"/parts/move", url.Values{"part": {three}, "dir": {"up"}})
	nt, _ := a.Types.Get("note")
	piece, _ := a.Store.Get("note", book)
	if parts := chat.Parts(a.Store, nt, piece); parts[1].ID != three {
		t.Errorf("Filling moved up to second")
	}
	log, _ := a.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if log[0].Fields["action"] != "updated" || log[0].Fields["target_id"] != book {
		t.Errorf("a move is one change to the piece, to undo like any other: %v", log[0].Fields)
	}

	whole := get(t, h, "/t/note/"+book+"/whole").Body.String()
	if !strings.Contains(whole, "How we made a pond.") || !strings.Contains(whole, "Rain did the rest.") {
		t.Errorf("Read it all is the piece then its parts: %.2000s", whole)
	}
	if strings.Index(whole, "Rain did") > strings.Index(whole, "The liner came") {
		t.Error("in their order")
	}
	if !strings.Contains(whole, "whole?as=docx") {
		t.Error("to take away too")
	}
	if md := get(t, h, "/t/note/"+book+"/whole?as=md").Body.String(); !strings.HasPrefix(md, "# The Pond") || !strings.Contains(md, "## Digging") {
		t.Errorf("as Markdown, each part under its title: %q", md)
	}
	if res := get(t, h, "/t/note/"+book+"/whole?as=docx"); !strings.Contains(res.Header().Get("Content-Type"), "wordprocessingml") {
		t.Errorf("as Word: %s", res.Header().Get("Content-Type"))
	}
	if strings.Contains(get(t, h, "/t/note/"+guide+"?show=place").Body.String(), "Part 1 of") {
		t.Error("material is never a part")
	}
}

// A long piece lists its headings at the top, each leading to its place.
func TestALongPieceHasItsContents(t *testing.T) {
	a, h := newApp(t)
	n, _ := a.Store.Create("note", map[string]any{"title": "Guide", "body": "## Start\n\nText.\n\n## Middle\n\nMore.\n\n## End\n\nDone."})
	if strings.Contains(get(t, h, "/t/note/"+n.ID).Body.String(), `id="contents"`) {
		t.Error("contents are a part, off at rest")
	}
	page := get(t, h, "/t/note/"+n.ID+"?show=contents").Body.String()
	if !strings.Contains(page, `<h2 id="contents">Contents</h2>`) || !strings.Contains(page, `href="#h-`+n.ID+`-2">Middle</a>`) || !strings.Contains(page, `id="h-`+n.ID+`-2"`) {
		t.Errorf("contents lead to the headings: %.3000s", page)
	}
	short, _ := a.Store.Create("note", map[string]any{"title": "Short", "body": "## One\n\nText."})
	if strings.Contains(get(t, h, "/t/note/"+short.ID+"?show=contents").Body.String(), `id="contents"`) {
		t.Error("a short piece has no contents")
	}
}

// A type without the fields for it gets them the first time it is organised.
func TestOrganisingAddsTheFieldsATypeLacks(t *testing.T) {
	a, _ := newApp(t)
	big, _ := a.Store.Create("task", map[string]any{"title": "Move house"})
	small, _ := a.Store.Create("task", map[string]any{"title": "Pack books"})
	args, _ := json.Marshal(map[string]any{"type": "task", "piece": big.ID, "parts": []string{small.ID}})
	if text, isErr := a.Chat.Call("organise_writing", args); isErr {
		t.Fatal(text)
	}
	tt, _ := a.Types.Get("task")
	if !chat.Organised(tt) {
		t.Fatal("task can be organised now")
	}
	if got, _ := a.Store.Get("task", small.ID); got.Fields["part_of"] != big.ID {
		t.Error("and the part is in its piece")
	}
}

// What a record's page can show is said with the record, only what it has,
// so the assistant can decide and hand over the address.
func TestARecordSaysWhatItsPageCanShow(t *testing.T) {
	a, h := newApp(t)
	piece, _ := a.Store.Create("note", map[string]any{"title": "Book"})
	part, _ := a.Store.Create("note", map[string]any{"title": "One"})
	args, _ := json.Marshal(map[string]any{"piece": piece.ID, "parts": []string{part.ID}})
	if text, isErr := a.Chat.Call("organise_writing", args); isErr {
		t.Fatal(text)
	}
	var got struct {
		Parts []struct{ Key string } `json:"parts"`
		Open  string                 `json:"open"`
	}
	json.Unmarshal(get(t, h, "/api/note/"+piece.ID).Body.Bytes(), &got)
	if len(got.Parts) != 1 || got.Parts[0].Key != "outline" || got.Open == "" {
		t.Errorf("a piece with parts says it has an outline, and how to open it: %+v", got)
	}
	json.Unmarshal(get(t, h, "/api/note/"+part.ID).Body.Bytes(), &got)
	if len(got.Parts) != 1 || got.Parts[0].Key != "place" {
		t.Errorf("a part says where it is: %+v", got)
	}
}

// Parts come in reading order, those named first, then the rest as they
// were; material is for the whole; a piece cannot go inside its own part;
// and one Undo takes an organising back.
func TestOrganisingKeepsOrderAndUndoes(t *testing.T) {
	a, _ := newApp(t)
	note := func(title string) string {
		r, err := a.Store.Create("note", map[string]any{"title": title})
		if err != nil {
			t.Fatal(err)
		}
		return r.ID
	}
	call := func(args map[string]any) (string, bool) {
		raw, _ := json.Marshal(args)
		return a.Chat.Call("organise_writing", raw)
	}
	book, one, two, three, guide := note("The Pond"), note("Digging"), note("Lining"), note("Filling"), note("Guidelines")
	call(map[string]any{"piece": book, "parts": []string{two, one}, "material": []any{map[string]any{"id": guide}}})
	call(map[string]any{"piece": book, "parts": []string{three}})
	nt, _ := a.Types.Get("note")
	piece, _ := a.Store.Get("note", book)
	var titles []string
	for _, p := range chat.Parts(a.Store, nt, piece) {
		titles = append(titles, p.Fields["title"].(string))
	}
	if strings.Join(titles, ",") != "Filling,Lining,Digging" {
		t.Errorf("parts named come first, in order, then the rest: %v", titles)
	}
	part, _ := a.Store.Get("note", one)
	if mine, above := chat.Material(a.Store, nt, part); len(mine) != 0 || len(above[book]) != 1 {
		t.Errorf("a part has the whole piece's material: %v %v", mine, above)
	}
	if text, isErr := call(map[string]any{"piece": one, "parts": []string{book}}); !isErr || !strings.Contains(text, "inside it") {
		t.Errorf("a piece cannot go inside its own part: %s", text)
	}
	entries, _ := a.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
	var last string
	for _, e := range entries {
		if e.Fields["action"] == "organised" {
			last = e.ID
			break
		}
	}
	undo, _ := json.Marshal(map[string]any{"id": last})
	if text, isErr := a.Chat.Call("undo_change", undo); isErr {
		t.Fatal(text)
	}
	if p, _ := a.Store.Get("note", three); p.Fields["part_of"] == book {
		t.Error("undo takes the second organising back")
	}
	if p, _ := a.Store.Get("note", two); p.Fields["part_of"] != book {
		t.Error("and leaves the first")
	}
}
