// Package schema loads content type definitions from a workspace.
//
// A content type is one YAML file in workspace/schema/. It is the single
// contract from which the store, the CLI, the JSON API, the HTML views, and
// later the MCP tools are generated.
package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// FieldTypes lists every supported field type, in documentation order.
var FieldTypes = []string{"string", "text", "markdown", "int", "float", "bool", "enum", "list", "json", "datetime"}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// Field describes one field of a content type.
type Field struct {
	Name        string   `yaml:"-" json:"name"`
	Type        string   `yaml:"type" json:"type"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Default     any      `yaml:"default,omitempty" json:"default,omitempty"`
	Values      []string `yaml:"values,omitempty" json:"values,omitempty"`
	Of          string   `yaml:"of,omitempty" json:"of,omitempty"`
	MaxLength   int      `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
}

// Type is one content type.
type Type struct {
	Name        string  `yaml:"name" json:"name"`
	Description string  `yaml:"description,omitempty" json:"description,omitempty"`
	Fields      []Field `yaml:"-" json:"fields"`
	// Title names the field used as a record's display title. Defaults to
	// the first string field.
	Title string `yaml:"title,omitempty" json:"title,omitempty"`
	// Internal types are used by the system (chat messages, canvas blocks)
	// and are hidden from the main navigation.
	Internal bool `yaml:"internal,omitempty" json:"internal,omitempty"`
	// File is the YAML path the type was loaded from, for error messages.
	File string `yaml:"-" json:"-"`
}

// Set is every content type in a workspace, sorted by name.
type Set struct {
	Types  []*Type
	byName map[string]*Type
}

// Load reads every *.yaml file in dir. A missing dir yields an empty set.
func Load(dir string) (*Set, error) {
	set := &Set{byName: map[string]*Type{}}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return set, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		t, err := LoadFile(path)
		if err != nil {
			return nil, err
		}
		if _, dup := set.byName[t.Name]; dup {
			return nil, fmt.Errorf("%s: content type %q is defined twice", path, t.Name)
		}
		set.byName[t.Name] = t
		set.Types = append(set.Types, t)
	}
	sort.Slice(set.Types, func(i, j int) bool { return set.Types[i].Name < set.Types[j].Name })
	return set, nil
}

// LoadFile parses and validates one content type file.
func LoadFile(path string) (*Type, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t, err := Parse(src)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	t.File = path
	return t, nil
}

// Parse decodes YAML into a validated Type, preserving field order.
func Parse(src []byte) (*Type, error) {
	var t Type
	if err := yaml.Unmarshal(src, &t); err != nil {
		return nil, err
	}
	var raw struct {
		Fields yaml.Node `yaml:"fields"`
	}
	if err := yaml.Unmarshal(src, &raw); err != nil {
		return nil, err
	}
	if raw.Fields.Kind != 0 && raw.Fields.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("fields must be a mapping of name to definition")
	}
	for i := 0; i+1 < len(raw.Fields.Content); i += 2 {
		var f Field
		if err := raw.Fields.Content[i+1].Decode(&f); err != nil {
			return nil, fmt.Errorf("field %s: %w", raw.Fields.Content[i].Value, err)
		}
		f.Name = raw.Fields.Content[i].Value
		t.Fields = append(t.Fields, f)
	}
	if err := t.validate(); err != nil {
		return nil, err
	}
	return &t, nil
}

func (t *Type) validate() error {
	if !nameRe.MatchString(t.Name) {
		return fmt.Errorf("name %q must be lowercase letters, digits, or underscores", t.Name)
	}
	if len(t.Fields) == 0 {
		return fmt.Errorf("type %s has no fields", t.Name)
	}
	for _, f := range t.Fields {
		if !nameRe.MatchString(f.Name) || f.Name == "id" || strings.HasSuffix(f.Name, "_at") {
			return fmt.Errorf("type %s: field name %q is invalid or reserved", t.Name, f.Name)
		}
		if !contains(FieldTypes, f.Type) {
			return fmt.Errorf("type %s: field %s has unknown type %q (use one of %s)", t.Name, f.Name, f.Type, strings.Join(FieldTypes, ", "))
		}
		if f.Type == "enum" && len(f.Values) == 0 {
			return fmt.Errorf("type %s: enum field %s needs values", t.Name, f.Name)
		}
		if f.Type == "list" && f.Of == "" {
			f.Of = "string"
		}
	}
	if t.Title == "" {
		for _, f := range t.Fields {
			if f.Type == "string" {
				t.Title = f.Name
				break
			}
		}
	}
	return nil
}

// Get returns a type by name.
func (s *Set) Get(name string) (*Type, bool) {
	t, ok := s.byName[name]
	return t, ok
}

// Names lists type names in sorted order.
func (s *Set) Names() []string {
	names := make([]string, 0, len(s.Types))
	for _, t := range s.Types {
		names = append(names, t.Name)
	}
	return names
}

// Field returns a field by name.
func (t *Type) Field(name string) (*Field, bool) {
	for i := range t.Fields {
		if t.Fields[i].Name == name {
			return &t.Fields[i], true
		}
	}
	return nil, false
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
