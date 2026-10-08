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

// Kind is what Sameway knows of one component beyond its manifest, in
// one place: for a data-bound one (a chart, a list), how its props
// resolve to what its template draws and what it shows in a few words
// ("3 habits", "How many tasks by Status: 3 groups"); for any, what a
// person calls it, how tall it usually stands, and which prop holds its
// heading. A resolved block that cannot be shown says why in
// out["problem"], in the words its page shows. The table is kinds.go.
type Kind struct {
	// Resolve is nil for a component that reads no records.
	Resolve func(w *Workspace, props map[string]any, at Place) map[string]any
	// Shows is nil for a kind with nothing to count.
	Shows func(w *Workspace, props, out map[string]any) string
	// Noun is what people call it when not its name: a list, not a
	// collection. Empty, its name in words.
	Noun string
	// Height is how tall it usually stands beside others, Tall or Short,
	// from its props; nil or "" is in between.
	Height func(props map[string]any) string
	// Heading is the prop holding the heading it puts in the page outline.
	Heading string
}

// Of is a component's kind, when it is a data-bound one.
func Of(component string) (Kind, bool) {
	k, ok := kinds[component]
	return k, ok && k.Resolve != nil
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
