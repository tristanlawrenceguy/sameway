package render

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
)

// An Arrangement is the thought behind a whole page, written once: which
// blocks a job wants, where each sits and how wide, in what order, with
// words the assistant fills in. A component is what a thing is; an
// arrangement is what a page for a job is. The assistant applies one in
// one call instead of composing a layout from scratch every time.
type Arrangement struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Use         *Use            `json:"use,omitempty"`
	Blocks      []ArrangedBlock `json:"blocks"`
	Source      string          `json:"source"`
}

// ArrangedBlock is one block of an arrangement: the component, its
// starting props, and how it sits.
type ArrangedBlock struct {
	// Key names the block within the arrangement, so fills can address it.
	Key       string         `json:"key"`
	Component string         `json:"component"`
	Props     map[string]any `json:"props"`
	Span      int            `json:"span,omitempty"`
	Frame     string         `json:"frame,omitempty"`
	Tone      string         `json:"tone,omitempty"`
	Region    string         `json:"region,omitempty"`
}

// LoadArrangementsFS loads every *.json arrangement directly under root.
// Each is checked against the components it uses, so an arrangement that
// could not be added is refused when it is loaded, not when it is asked for.
func (r *Registry) LoadArrangementsFS(fsys fs.FS, root, source string) error {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return err
	}
	if r.arrangements == nil {
		r.arrangements = map[string]*Arrangement{}
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := fs.ReadFile(fsys, path.Join(root, e.Name()))
		if err != nil {
			return err
		}
		a := &Arrangement{Source: source}
		if err := json.Unmarshal(raw, a); err != nil {
			return fmt.Errorf("arrangement %s: %w", e.Name(), err)
		}
		if a.Name == "" || a.Name+".json" != e.Name() {
			return fmt.Errorf("arrangement %s: name %q must match the file name", e.Name(), a.Name)
		}
		if err := r.checkArrangement(a); err != nil {
			return fmt.Errorf("arrangement %s: %w", a.Name, err)
		}
		r.arrangements[a.Name] = a
	}
	return nil
}

// LoadArrangementsDir loads a workspace's own arrangements. A missing
// directory is not an error.
func (r *Registry) LoadArrangementsDir(dir, source string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	return r.LoadArrangementsFS(os.DirFS(dir), ".", source)
}

func (r *Registry) checkArrangement(a *Arrangement) error {
	if len(a.Blocks) == 0 {
		return fmt.Errorf("has no blocks")
	}
	if a.Use == nil || a.Use.When == "" {
		return fmt.Errorf("needs use.when: the thought behind it")
	}
	keys := map[string]bool{}
	for i, b := range a.Blocks {
		if b.Key == "" || keys[b.Key] {
			return fmt.Errorf("block %d needs a key of its own", i+1)
		}
		keys[b.Key] = true
		c, ok := r.Get(b.Component)
		if !ok {
			return fmt.Errorf("block %s uses unknown component %q", b.Key, b.Component)
		}
		if _, err := c.Validate(b.Props); err != nil {
			return fmt.Errorf("block %s: %w", b.Key, err)
		}
	}
	return nil
}

// Arrangements lists every loaded arrangement by name.
func (r *Registry) Arrangements() []*Arrangement {
	out := make([]*Arrangement, 0, len(r.arrangements))
	for _, a := range r.arrangements {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Arrangement finds one by name.
func (r *Registry) Arrangement(name string) (*Arrangement, bool) {
	a, ok := r.arrangements[name]
	return a, ok
}
