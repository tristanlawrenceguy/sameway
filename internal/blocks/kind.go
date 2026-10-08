package blocks

import (
	"net/url"

	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Workspace is what a block is worked out from: the records, and the
// settings that say which lists are shown (ui.lists).
type Workspace struct {
	Store *store.Store
	// Settings is the workspace's, read as they are now; nil reads as the
	// defaults, as for a store opened on its own.
	Settings *workspace.Workspace
}

// Place is where a block is shown: its id, and the page it is on with
// the choices a person made there. A block not yet written has no id; a
// block where there is no page to come back to (a side pane, a block
// arriving live, a check) has no page.
type Place struct {
	Block string
	Page  *Page
}

// Page is the page a block is shown on: its path, its address's query
// as asked for, whether it is the block's own page, and whether the one
// looking may change the block.
type Page struct {
	Path   string
	Query  url.Values
	Own    bool
	Change bool
}

// Kind is one data-bound component: how its props resolve to what its
// template draws, and what it shows in a few words ("3 habits",
// "How many tasks by Status: 3 groups") once resolved. A resolved block
// that cannot be shown says why in out["problem"], in the words its page
// shows.
type Kind struct {
	Resolve func(w *Workspace, props map[string]any, at Place) map[string]any
	// Shows is nil for a kind with nothing to count.
	Shows func(w *Workspace, props, out map[string]any) string
}

// kinds are the data-bound components, by name.
var kinds = map[string]Kind{
	ChartComponent:   {Resolve: resolveChart, Shows: chartShows},
	TrackerComponent: {Resolve: resolveTracker, Shows: trackerShows},
	ClockComponent:   {Resolve: resolveClock},
	CollectionComponent: {Resolve: resolveCollection, Shows: func(w *Workspace, props, _ map[string]any) string {
		return w.collectionShows(props)
	}},
	CalendarComponent: {Resolve: resolveCalendar, Shows: (*Workspace).calendarShows},
}

// Of is a component's kind, when it is a data-bound one.
func Of(component string) (Kind, bool) {
	k, ok := kinds[component]
	return k, ok
}

// Resolve is a block's props as its template takes them: resolved by its
// kind, or as they are, copied, for a component that reads no records.
func Resolve(w *Workspace, component string, props map[string]any, at Place) map[string]any {
	if k, ok := kinds[component]; ok && k.Resolve != nil {
		return k.Resolve(w, props, at)
	}
	return copyProps(props)
}

// copyProps is props to fill in, leaving the stored block as it is.
func copyProps(props map[string]any) map[string]any {
	out := make(map[string]any, len(props))
	for k, v := range props {
		out[k] = v
	}
	return out
}

// listsAll is whether the workspace shows every list, ui.lists: all.
func (w *Workspace) listsAll() bool {
	return w.Settings != nil && w.Settings.Config.UI.Lists == "all"
}
