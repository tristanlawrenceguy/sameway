package schema

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// LoadFS reads every *.yaml file in dir of fsys. It is how the content
// types that ship inside the binary are read. A missing dir yields an
// empty set.
func LoadFS(fsys fs.FS, dir string) (*Set, error) {
	return load(fsys, dir, func(name string) string { return path.Join(dir, name) })
}

// load is what Load and LoadFS share. pathOf names a file for error
// messages and for Type.File.
func load(fsys fs.FS, dir string, pathOf func(name string) string) (*Set, error) {
	set := &Set{byName: map[string]*Type{}}
	entries, err := fs.ReadDir(fsys, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return set, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		src, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		t, err := Parse(src)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", pathOf(e.Name()), err)
		}
		t.File = pathOf(e.Name())
		if _, dup := set.byName[t.Name]; dup {
			return nil, fmt.Errorf("%s: content type %q is defined twice", t.File, t.Name)
		}
		set.byName[t.Name] = t
		set.Types = append(set.Types, t)
	}
	sort.Slice(set.Types, func(i, j int) bool { return set.Types[i].Name < set.Types[j].Name })
	return set, nil
}

// Complete gives every internal type the fields its built-in definition
// has and the workspace copy lacks.
//
// Internal types (blocks, messages, activity) belong to the system:
// `sameway init` copies them into the workspace so people can read them,
// but a copy made before a field existed must not stop the assistant from
// using that field after an upgrade. Without this, a block placed in a
// side pane silently lost its region and landed in the main column.
//
// Fields the workspace already defines are kept as they are, in their
// order, with the missing ones appended after. Types the built-in set does
// not mark internal are never touched, and a workspace that has removed an
// internal type altogether is left without it.
func (s *Set) Complete(builtin *Set) {
	for _, b := range builtin.Types {
		if !b.Internal {
			continue
		}
		t, ok := s.byName[b.Name]
		if !ok {
			continue
		}
		for _, f := range b.Fields {
			if _, has := t.Field(f.Name); !has {
				t.Fields = append(t.Fields, f)
			}
		}
	}
}
