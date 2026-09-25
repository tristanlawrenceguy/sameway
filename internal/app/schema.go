package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// AddField gives a content type one more field while the workspace runs:
// the type's YAML gets the field (a provided type that had no file of its
// own gets one first), the table gets its column, and every page, tool
// and route sees it at once. What the file says is validated the way it
// would be at start-up, so a bad field never lands.
func (a *App) AddField(typeName string, f schema.Field) (*schema.Type, error) {
	t, ok := a.Types.Get(typeName)
	if !ok {
		return nil, fmt.Errorf("there is no content type %q", typeName)
	}
	if _, has := t.Field(f.Name); has {
		return nil, fmt.Errorf("%s already has a field called %s", t.Name, f.Name)
	}
	path := t.File
	var src []byte
	if path == "" {
		path = filepath.Join(a.Workspace.Dir, "schema", t.Name+".yaml")
		var err error
		if src, err = schema.TypeYAML(t); err != nil {
			return nil, err
		}
	} else {
		var err error
		if src, err = os.ReadFile(path); err != nil {
			return nil, err
		}
	}
	next, err := schema.AddFieldYAML(src, f)
	if err != nil {
		return nil, err
	}
	parsed, err := schema.Parse(next)
	if err != nil {
		return nil, err
	}
	if f.Type == "ref" {
		if _, ok := a.Types.Get(f.To); !ok {
			return nil, fmt.Errorf("a ref needs a type to point at; %q is not one here", f.To)
		}
	}
	if err := a.writeSchema(path, next); err != nil {
		return nil, err
	}
	t.Fields = parsed.Fields
	t.File = path
	if err := a.Store.Migrate(); err != nil {
		return nil, err
	}
	a.Store.StampSchema(t)
	return t, nil
}

// AddType makes a new content type while the workspace runs: its file in
// schema/, its table, and its place in every catalogue.
func (a *App) AddType(t *schema.Type) (*schema.Type, error) {
	if _, exists := a.Types.Get(t.Name); exists {
		return nil, fmt.Errorf("there is already a content type called %s", t.Name)
	}
	src, err := schema.TypeYAML(t)
	if err != nil {
		return nil, err
	}
	parsed, err := schema.Parse(src)
	if err != nil {
		return nil, err
	}
	for _, f := range parsed.Fields {
		if f.Type == "ref" {
			if _, ok := a.Types.Get(f.To); !ok && f.To != t.Name {
				return nil, fmt.Errorf("field %s points at %q, which is not a content type here", f.Name, f.To)
			}
		}
	}
	path := filepath.Join(a.Workspace.Dir, "schema", t.Name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return nil, errors.New("there is already a file at " + path)
	}
	if err := a.writeSchema(path, src); err != nil {
		return nil, err
	}
	parsed.File = path
	a.Types.Put(parsed)
	if err := a.Store.Migrate(); err != nil {
		return nil, err
	}
	a.Store.StampSchema(parsed)
	return parsed, nil
}

func (a *App) writeSchema(path string, src []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, src, 0o644)
}
