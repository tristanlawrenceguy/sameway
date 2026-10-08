package blocks

import "github.com/tristanlawrenceguy/sameway/internal/schema"

// kinds is every component Sameway knows something of beyond its
// manifest, by name. The page resolves a block through it, the check made
// when a block is written asks it what the block shows, and the assistant
// names a block and weighs the page's layout by it: once four switches,
// one in each, that a new component had to be added to apart.
var kinds = map[string]Kind{
	ChartComponent:   {Resolve: resolveChart, Shows: chartShows, Height: tallUnlessSmall},
	TrackerComponent: {Resolve: resolveTracker, Shows: trackerShows, Height: tallUnlessSmall},
	ClockComponent:   {Resolve: resolveClock, Height: always(Short)},
	CollectionComponent: {Resolve: resolveCollection, Shows: func(w *Workspace, props, _ map[string]any) string {
		return w.collectionShows(props)
	}, Noun: "list", Height: board, Heading: "label"},
	CalendarComponent: {Resolve: resolveCalendar, Shows: (*Workspace).calendarShows, Height: tallUnlessSmall},
	"text":            {Noun: "piece of text", Height: shortText},
	"record":          {Noun: "record"},
	"filters":         {Noun: "set of filters"},
	"chat":            {Height: always(Tall)},
	"table":           {Height: always(Tall)},
	"heading":         {Height: always(Short), Heading: "text"},
	"card":            {Height: always(Short), Heading: "title"},
	"alert":           {Height: always(Short), Heading: "title"},
	"list":            {Heading: "label"},
	"button":          {Height: always(Short)},
	"link":            {Height: always(Short)},
	"badge":           {Height: always(Short)},
	"status":          {Height: always(Short)},
	"search":          {Height: always(Short)},
}

// How tall a block usually stands: taller than a card, or a line or two.
const (
	Tall  = "tall"
	Short = "short"
)

// Noun is what people call a component: a list, a piece of text, or its
// name in words.
func Noun(component string) string {
	if n := kinds[component].Noun; n != "" {
		return n
	}
	return schema.Words(component)
}

// Height is how tall a block of the component usually stands, Tall or
// Short, or "" for in between.
func Height(component string, props map[string]any) string {
	if h := kinds[component].Height; h != nil {
		return h(props)
	}
	return ""
}

// Heading is the prop that holds the heading a component puts in the
// page outline, or "" for one that puts none there.
func Heading(component string) string { return kinds[component].Heading }

func always(h string) func(map[string]any) string {
	return func(map[string]any) string { return h }
}

// tallUnlessSmall is a block that is tall unless shown at a glance or
// in brief.
func tallUnlessSmall(props map[string]any) string {
	if d, _ := props["detail"].(string); d == "glance" || d == "brief" {
		return ""
	}
	return Tall
}

// board is a collection, tall as a board and in between as a list.
func board(props map[string]any) string {
	if as, _ := props["as"].(string); as == "board" {
		return Tall
	}
	return ""
}

// shortText is a text block, short while its words are few.
func shortText(props map[string]any) string {
	if s, _ := props["content"].(string); len(s) < 240 {
		return Short
	}
	return ""
}
