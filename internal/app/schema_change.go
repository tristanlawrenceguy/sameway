package app

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A content type can change after it is made, by asking: a pick-list gets
// another choice, a field or a choice is called something else, a field
// or a whole type is hidden (kept, off the pages) or shown again, and, when
// the person says yes to it, deleted. Each is written to the type's file
// the way adding is, and stamped so the other computers that host the
// workspace change the same way.

// rewrite changes a type's file by edit, reads it back, and takes what it
// now says: the type's fields, whether it is hidden.
func (a *App) rewrite(t *schema.Type, edit func([]byte) ([]byte, error)) (*schema.Type, error) {
	path, src := t.File, []byte(nil)
	var err error
	if path == "" {
		path = filepath.Join(a.Workspace.Dir, "schema", t.Name+".yaml")
		src, err = schema.TypeYAML(t)
	} else {
		src, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	next, err := edit(src)
	if err != nil {
		return nil, err
	}
	parsed, err := schema.Parse(next)
	if err != nil {
		return nil, err
	}
	if err := a.writeSchema(path, next); err != nil {
		return nil, err
	}
	t.Fields, t.Hidden, t.File = parsed.Fields, parsed.Hidden, path
	if err := a.Store.Migrate(); err != nil {
		return nil, err
	}
	a.Store.StampSchema(t)
	return t, nil
}

// changeField rewrites one field as change makes it.
func (a *App) changeField(typeName, field string, change func(*schema.Field) error) (*schema.Type, error) {
	t, ok := a.Types.Get(typeName)
	if !ok {
		return nil, fmt.Errorf("there is no content type %q", typeName)
	}
	have, ok := t.Field(field)
	if !ok {
		return nil, fmt.Errorf("%s has no field called %s", t.Name, field)
	}
	f := *have
	f.Labels = maps.Clone(have.Labels)
	if err := change(&f); err != nil {
		return nil, err
	}
	return a.rewrite(t, func(src []byte) ([]byte, error) { return schema.ReplaceFieldYAML(src, f) })
}

// AddChoice gives a pick-list one more choice, with how it reads.
func (a *App) AddChoice(typeName, field, value, label string) (*schema.Type, error) {
	return a.changeField(typeName, field, func(f *schema.Field) error {
		if f.Type != "enum" {
			return fmt.Errorf("%s is not a pick-list, so it has no choices", field)
		}
		if slices.Contains(f.Values, value) {
			return fmt.Errorf("%s already offers %s", field, value)
		}
		f.Values = append(f.Values, value)
		if label != "" {
			if f.Labels == nil {
				f.Labels = map[string]string{}
			}
			f.Labels[value] = label
		}
		return nil
	})
}

// Relabel changes what a field is called on the pages, or, given a
// choice, what that choice is called. Its name, and what is stored, stay.
func (a *App) Relabel(typeName, field, choice, label string) (*schema.Type, error) {
	return a.changeField(typeName, field, func(f *schema.Field) error {
		if choice == "" {
			f.Label = label
			return nil
		}
		if !slices.Contains(f.Values, choice) {
			return fmt.Errorf("%s offers no choice %s", field, choice)
		}
		if f.Labels == nil {
			f.Labels = map[string]string{}
		}
		f.Labels[choice] = label
		if label == "" {
			delete(f.Labels, choice)
		}
		return nil
	})
}

// SetHidden hides a field, or with field "" a whole type, or shows it
// again. Nothing it holds is lost either way.
func (a *App) SetHidden(typeName, field string, hidden bool) (*schema.Type, error) {
	if field != "" {
		return a.changeField(typeName, field, func(f *schema.Field) error { f.Hidden = hidden; return nil })
	}
	t, ok := a.Types.Get(typeName)
	if !ok {
		return nil, fmt.Errorf("there is no content type %q", typeName)
	}
	return a.rewrite(t, func(src []byte) ([]byte, error) { return schema.SetTypeYAML(src, "hidden", hidden) })
}

// kept refuses what the workspace itself depends on: the system's own
// types, the fields Sameway keeps, and a type's title.
func kept(t *schema.Type, field string) error {
	if t.Internal {
		return fmt.Errorf("%s is part of Sameway itself; it can be hidden but not deleted", t.Name)
	}
	if field == "" {
		if t.Provided {
			return fmt.Errorf("%s comes with Sameway and other parts rely on it; hide it instead", t.Name)
		}
		return nil
	}
	f, _ := t.Field(field)
	if f.ReadOnly || field == t.Title {
		return fmt.Errorf("%s is kept by Sameway or names each %s; hide it instead", field, t.Name)
	}
	return nil
}

// RemoveField deletes a field: what every record held in it is cleared,
// on every computer, then the field leaves the type.
func (a *App) RemoveField(typeName, field string) (*schema.Type, error) {
	t, ok := a.Types.Get(typeName)
	if !ok {
		return nil, fmt.Errorf("there is no content type %q", typeName)
	}
	if _, has := t.Field(field); !has {
		return nil, fmt.Errorf("%s has no field called %s", t.Name, field)
	}
	if err := kept(t, field); err != nil {
		return nil, err
	}
	recs, err := a.Store.List(t.Name, store.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, r := range recs {
		if r.Fields[field] != nil {
			a.Store.Update(t.Name, r.ID, map[string]any{field: nil})
		}
	}
	a.Store.StampDeleted(t.Name, field)
	return a.rewrite(t, func(src []byte) ([]byte, error) { return schema.RemoveFieldYAML(src, field) })
}

// RemoveType deletes a type and every record of it, on every computer.
func (a *App) RemoveType(typeName string) error {
	t, ok := a.Types.Get(typeName)
	if !ok {
		return fmt.Errorf("there is no content type %q", typeName)
	}
	if err := kept(t, ""); err != nil {
		return err
	}
	if err := a.Store.DeleteAll(t.Name); err != nil {
		return err
	}
	a.Store.StampDeleted(t.Name, "")
	if t.File != "" {
		if err := os.Remove(t.File); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	a.Types.Remove(t.Name)
	return nil
}
