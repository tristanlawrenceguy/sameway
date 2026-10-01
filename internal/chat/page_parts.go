package chat

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/prose"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A record's page is the record and nothing else at rest. What more it can
// show is said here, with the record, so whoever reads it decides whether
// the person would be served by it and hands over the address that opens
// it, /t/<type>/<id>?show=<key>; set_setting ui.show +<key> keeps one on.
// Only what this record has is listed: a piece with no parts offers no
// outline.

// PagePart is something a record's page can be asked to show.
type PagePart struct {
	Key   string `json:"key"`
	Shows string `json:"shows"`
}

// PageParts are the parts this record's page has to show.
func PageParts(st *store.Store, t *schema.Type, rec *store.Record) []PagePart {
	var out []PagePart
	add := func(key, shows string) { out = append(out, PagePart{key, shows}) }
	text := ""
	for _, f := range t.Fields {
		if f.Type == "markdown" {
			text, _ = rec.Fields[f.Name].(string)
			break
		}
	}
	if len(prose.Headings(text, 3, "h")) >= 3 {
		add("contents", "its headings at the top, each leading to its place")
	}
	if WordCount(text) >= 30 && !t.Internal {
		add("writing-help", "the kinds of help with its writing: spelling, tightening, feedback on structure")
	}
	if Organised(t) {
		if PieceOf(st, t, rec) != nil {
			add("place", "where it is in its piece, with the parts either side")
		}
		if len(Parts(st, t, rec)) > 0 {
			add("outline", "its parts in order with what each is about, its words against its aim, moving them, and reading it all")
		}
		if mine, above := Material(st, t, rec); len(mine)+len(above) > 0 {
			add("material", "the guidelines, details and research kept with it")
		}
	}
	if t.Name == EventType {
		if id, _ := rec.Fields["recording"].(string); id != "" {
			add("recording", "its recording, played with its transcript")
			if s, _ := rec.Fields["summary"].(string); strings.TrimSpace(s) == "" {
				add("write-up", "the offer to have it written up from its recording")
			}
		}
	}
	if t.Name == FileType && (rec.Fields["kind"] == "audio" || rec.Fields["kind"] == "video") && rec.Fields["text"] != "" {
		if et, ok := st.Types().Get(EventType); ok {
			if _, has := et.Field("recording"); has {
				if had, _ := query.Filter(st, et, []string{"recording=" + rec.ID}, "", 1, time.Now()); len(had) == 0 {
					add("write-up", "the offer to write up the meeting in it")
				}
			}
		}
	}
	return out
}
