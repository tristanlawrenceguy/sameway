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
		if !b.Internal && !b.Provided {
			continue
		}
		t, ok := s.byName[b.Name]
		if !ok {
			// A whole internal type the workspace predates, such as the
			// canvas tabs: the system owns it, so the workspace gets it.
			added := *b
			added.Fields = append([]Field(nil), b.Fields...)
			s.byName[added.Name] = &added
			s.Types = append(s.Types, &added)
			continue
		}
		for _, f := range b.Fields {
			if _, has := t.Field(f.Name); !has {
				t.Fields = append(t.Fields, f)
				continue
			}
			// Names for an enum's values reach a workspace made before they
			// were written, for the values its copy has; its own names win.
			if len(f.Labels) > 0 {
				for i := range t.Fields {
					if t.Fields[i].Name == f.Name && t.Fields[i].Type == "enum" && len(t.Fields[i].Labels) == 0 {
						t.Fields[i].Labels = map[string]string{}
						for _, v := range t.Fields[i].Values {
							if l, ok := f.Labels[v]; ok {
								t.Fields[i].Labels[v] = l
							}
						}
					}
				}
			}
			// A field the system keeps stays the system's, whatever an
			// older copy of the type says.
			if f.ReadOnly {
				for i := range t.Fields {
					if t.Fields[i].Name == f.Name {
						t.Fields[i].ReadOnly = true
					}
				}
			}
		}
		// What names a record of an internal type is the system's to say
		// too: a workspace copy of the activity log from before the
		// summary field named records by their verb, so every heading on
		// the activity page read "said" long after the field existed.
		if b.Internal && b.Title != "" {
			t.Title = b.Title
		}
	}
	sort.Slice(s.Types, func(i, j int) bool { return s.Types[i].Name < s.Types[j].Name })
}

// CheckRefs says which ref field points at a type the workspace does not
// have, once the provided types are in. A ref to nothing is a schema
// mistake worth stopping on, with the type named.
func (s *Set) CheckRefs() error {
	for _, t := range s.Types {
		for _, f := range t.Fields {
			if f.Type != "ref" {
				continue
			}
			if _, ok := s.byName[f.To]; !ok {
				return fmt.Errorf("type %s: field %s points at %q, which is not a content type here (the workspace has %s)", t.Name, f.Name, f.To, strings.Join(s.Names(), ", "))
			}
		}
	}
	return nil
}
