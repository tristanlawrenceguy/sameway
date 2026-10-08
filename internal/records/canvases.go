package records

import (
	"errors"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// HomePath is where the first canvas lives.
const HomePath = "/"

// CanvasPath is the page for a canvas: Home, or a tab by id.
func CanvasPath(id string) string {
	if id == "" {
		return HomePath
	}
	return "/c/" + id
}

// OnCanvas keeps the blocks that belong to one tab.
func OnCanvas(blocks []*store.Record, id string) []*store.Record {
	var out []*store.Record
	for _, b := range blocks {
		on, _ := b.Fields["canvas"].(string)
		if on == id {
			out = append(out, b)
		}
	}
	return out
}

// RemoveCanvas takes a tab away with the blocks on it, and says it as a
// change that keeps them all, so undoing it puts the whole tab back.
func RemoveCanvas(st *store.Store, id string) (Change, error) {
	if id == "" {
		return Change{}, errors.New("Home is the first canvas and stays; remove the blocks on it instead")
	}
	rec, err := st.Get(CanvasType, id)
	if err != nil {
		return Change{}, fmt.Errorf("no canvas with id %s; the tabs are listed in the prompt", id)
	}
	name, _ := rec.Fields["name"].(string)
	// The tab and its blocks go into the log together, so undoing this
	// puts the whole tab back.
	done, err := ApplyOps(st, Op{Type: CanvasType, ID: id})
	if err != nil {
		return Change{}, fmt.Errorf("could not remove the canvas: %v", err)
	}
	return Change{Action: "removed", Component: CanvasType, ID: id, Detail: name, Ops: done}, nil
}
