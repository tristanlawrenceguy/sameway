package server

import "github.com/tristanlawrenceguy/sameway/internal/blocks"

// What a block shows is worked out in internal/blocks, without HTTP; the
// pages draw it. These are its names as the pages use them.

const (
	chartComponent   = blocks.ChartComponent
	trackerComponent = blocks.TrackerComponent
	clockComponent   = blocks.ClockComponent
)

// Words the pages say as a block does (blocks/words.go).
var (
	display    = blocks.Display
	sizeWords  = blocks.SizeWords
	capitalize = blocks.Capitalize
	strs       = blocks.Strs
	andList    = blocks.AndList
)

// tracker is a tracker block's props resolved, for a page that shows the
// habits outside a canvas (the habits list, the review).
func (s *Server) tracker(props map[string]any) map[string]any {
	return blocks.Resolve(s.app.Blocks, trackerComponent, props, blocks.Place{})
}
