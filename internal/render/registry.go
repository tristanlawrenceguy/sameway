// Package render loads component folders and renders them to HTML.
//
// A component is a folder with manifest.json, template.html, style.css, and
// examples/. Built-in components come from the embedded design system;
// workspace components come from <workspace>/components and override
// built-ins with the same name.
package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
)

// Manifest is the machine-readable contract of a component.
type Manifest struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Props       json.RawMessage `json:"props"`
	A11y        json.RawMessage `json:"a11y"`
	Machine     json.RawMessage `json:"machine"`
	Examples    []Example       `json:"examples"`
}

// Example is one named set of props with its golden output file.
type Example struct {
	Name  string         `json:"name"`
	Props map[string]any `json:"props"`
	File  string         `json:"file"`
}

// Component is a loaded, compiled component.
type Component struct {
	Manifest Manifest
	// Source is "builtin" or "workspace".
	Source string
	CSS    string

	fsys  fs.FS
	dir   string
	tmpl  *template.Template
	props *propSchema
}

// Registry holds every loaded component by name.
type Registry struct {
	byName map[string]*Component
	tokens string
	base   string
}

// New returns an empty registry.
func New() *Registry { return &Registry{byName: map[string]*Component{}} }

// SetBase records the token and base CSS that precede component CSS.
func (r *Registry) SetBase(tokensCSS, baseCSS string) { r.tokens, r.base = tokensCSS, baseCSS }

// LoadFS loads every component folder directly under root in fsys.
func (r *Registry) LoadFS(fsys fs.FS, root, source string) error {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		c, err := load(fsys, path.Join(root, e.Name()), source)
		if err != nil {
			return err
		}
		r.byName[c.Manifest.Name] = c
	}
	return nil
}

// LoadDir loads workspace components from a directory on disk. A missing
// directory is not an error.
func (r *Registry) LoadDir(dir, source string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	return r.LoadFS(os.DirFS(dir), ".", source)
}

func load(fsys fs.FS, dir, source string) (*Component, error) {
	raw, err := fs.ReadFile(fsys, path.Join(dir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("component %s: %w", dir, err)
	}
	c := &Component{Source: source, fsys: fsys, dir: dir}
	if err := json.Unmarshal(raw, &c.Manifest); err != nil {
		return nil, fmt.Errorf("component %s: manifest.json: %w", dir, err)
	}
	if c.Manifest.Name == "" || c.Manifest.Name != path.Base(dir) {
		return nil, fmt.Errorf("component %s: manifest name %q must match the folder name", dir, c.Manifest.Name)
	}
	if c.props, err = compileProps(c.Manifest.Name, c.Manifest.Props); err != nil {
		return nil, fmt.Errorf("component %s: props schema: %w", dir, err)
	}
	src, err := fs.ReadFile(fsys, path.Join(dir, "template.html"))
	if err != nil {
		return nil, fmt.Errorf("component %s: %w", dir, err)
	}
	c.tmpl, err = template.New(c.Manifest.Name).Funcs(Funcs).Option("missingkey=zero").Parse(strings.TrimRight(string(src), "\n"))
	if err != nil {
		return nil, fmt.Errorf("component %s: template.html: %w", dir, err)
	}
	if css, err := fs.ReadFile(fsys, path.Join(dir, "style.css")); err == nil {
		c.CSS = string(css)
	}
	return c, nil
}

// Get returns a component by name.
func (r *Registry) Get(name string) (*Component, bool) {
	c, ok := r.byName[name]
	return c, ok
}

// Components returns every component sorted by name.
func (r *Registry) Components() []*Component {
	out := make([]*Component, 0, len(r.byName))
	for _, c := range r.byName {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest.Name < out[j].Manifest.Name })
	return out
}

// Names returns component names sorted.
func (r *Registry) Names() []string {
	cs := r.Components()
	names := make([]string, len(cs))
	for i, c := range cs {
		names[i] = c.Manifest.Name
	}
	return names
}

// Render validates props against the component's schema, applies defaults,
// and executes the template.
func (r *Registry) Render(name string, props map[string]any) (template.HTML, error) {
	c, ok := r.byName[name]
	if !ok {
		return "", fmt.Errorf("unknown component %q (known: %s)", name, strings.Join(r.Names(), ", "))
	}
	return c.Render(props)
}

// Render validates and renders this component.
func (c *Component) Render(props map[string]any) (template.HTML, error) {
	clean, err := c.props.normalize(props)
	if err != nil {
		return "", fmt.Errorf("component %s: %w", c.Manifest.Name, err)
	}
	var buf bytes.Buffer
	if err := c.tmpl.Execute(&buf, clean); err != nil {
		return "", fmt.Errorf("component %s: %w", c.Manifest.Name, err)
	}
	return template.HTML(buf.String()), nil
}

// Validate checks props without rendering and returns the normalized props.
func (c *Component) Validate(props map[string]any) (map[string]any, error) {
	return c.props.normalize(props)
}

// ReadFile reads a file relative to the component folder, such as an example.
func (c *Component) ReadFile(rel string) ([]byte, error) {
	return fs.ReadFile(c.fsys, path.Join(c.dir, rel))
}

// Dir is the component folder path inside its filesystem.
func (c *Component) Dir() string { return c.dir }

// CSS concatenates tokens, base, and every component stylesheet in name order.
func (r *Registry) CSS() string {
	var b strings.Builder
	b.WriteString(r.tokens)
	b.WriteString("\n")
	b.WriteString(r.base)
	for _, c := range r.Components() {
		if c.CSS == "" {
			continue
		}
		fmt.Fprintf(&b, "\n/* component: %s */\n%s", c.Manifest.Name, c.CSS)
	}
	return b.String()
}
