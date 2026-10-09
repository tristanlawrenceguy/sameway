package server

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// What a block shows is worked out in internal/blocks, without HTTP; the
// pages draw it. These are its names as the pages use them.

const (
	chartComponent      = blocks.ChartComponent
	trackerComponent    = blocks.TrackerComponent
	clockComponent      = blocks.ClockComponent
	collectionComponent = blocks.CollectionComponent
	calendarComponent   = blocks.CalendarComponent
)

// Words the pages say as a block does (blocks/words.go).
var (
	sizeWords  = blocks.SizeWords
	capitalize = blocks.Capitalize
	strs       = blocks.Strs
	andList    = blocks.AndList
	isText     = blocks.IsText
	textOf     = blocks.TextOf
	orderWords = blocks.OrderWords
)

// Telling records with one name apart, as a block's list does
// (blocks/apart.go).
var (
	apart       = blocks.Apart
	withContext = blocks.WithContext
	addedRank   = blocks.AddedRank
)

// resolve is a block's props as its template takes them, where it is
// shown: by its kind (internal/blocks), or as they are.
func (s *Server) resolve(component string, props map[string]any, block string, page *blocks.Page) map[string]any {
	return blocks.Resolve(s.app.Blocks, component, props, blocks.Place{Block: block, Page: page})
}

// markOf is the yes-or-no fact a record offers to change (blocks/marks.go).
func (s *Server) markOf(t *schema.Type, rec *store.Record) (map[string]any, bool) {
	return s.app.Blocks.MarkOf(t, rec)
}

// markActions is the mark as a component item's actions.
func (s *Server) markActions(t *schema.Type, rec *store.Record) []any {
	return s.app.Blocks.MarkActions(t, rec)
}

// recordsApart tells one type's records shown together apart, by id.
func (s *Server) recordsApart(t *schema.Type, recs []*store.Record) map[string]string {
	return s.app.Blocks.RecordsApart(t, recs)
}

// tracker is a tracker block's props resolved, for a page that shows the
// habits outside a canvas (the habits list, the review).
func (s *Server) tracker(props map[string]any) map[string]any {
	return blocks.Resolve(s.app.Blocks, trackerComponent, props, blocks.Place{})
}

// blockName is what a block's own controls are named after: Remove Up
// next, Expand Up next, from the label the block shows, and the plain
// name of its component only when it has none. Two lists would otherwise
// both be Remove collection.
func blockName(component string, props map[string]any) string {
	if said := records.Summarise(component, props); said != "" {
		return said
	}
	for _, k := range []string{"label", "caption", "title"} {
		if v, _ := props[k].(string); strings.TrimSpace(v) != "" {
			return trim.Title(v)
		}
	}
	return component
}

// onCanvas is the place of a collection block on a canvas: a person
// narrows and sorts it where it has room, on the canvas itself at full
// size, not in a pane or a strip.
func onCanvas(b *store.Record, convo *conversation) *blocks.Page {
	if convo == nil || str(b.Fields["region"], "main") != "main" || str(b.Fields["size"], "full") != "full" {
		return nil
	}
	return &blocks.Page{Path: convo.Path, Query: convo.Query, Change: !convo.LookOnly}
}

// withMonth is the block's props with another month shown, the rest as
// they are; the stored block is not touched. An empty day is the month
// again.
func withMonth(props map[string]any, month, day string) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	if month != "" {
		out["month"] = month
	}
	if day != "" {
		out["day"] = day
	} else {
		delete(out, "day")
	}
	return out
}

// noType says a type is not there, and what is (blocks.NoType).
func (s *Server) noType(name string) string { return s.app.Blocks.NoType(name) }
