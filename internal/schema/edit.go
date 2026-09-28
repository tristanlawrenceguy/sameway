package schema

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// Put adds a type to the set, or replaces the one with its name, keeping
// the set sorted. It is how a type made while the workspace runs joins
// the ones loaded at the start.
func (s *Set) Put(t *Type) {
	if s.byName == nil {
		s.byName = map[string]*Type{}
	}
	if old, ok := s.byName[t.Name]; ok {
		*old = *t
		return
	}
	s.byName[t.Name] = t
	s.Types = append(s.Types, t)
	sort.Slice(s.Types, func(i, j int) bool { return s.Types[i].Name < s.Types[j].Name })
}

// AddFieldYAML appends one field to a type's YAML, leaving every comment
// and the order of what is there alone: the file a person wrote stays
// theirs, with one more field at the end of fields.
func AddFieldYAML(src []byte, f Field) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("the type file is not a mapping")
	}
	root := doc.Content[0]
	var fields *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "fields" {
			fields = root.Content[i+1]
		}
	}
	if fields == nil {
		fields = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "fields"}, fields)
	}
	if fields.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("fields must be a mapping of name to definition")
	}
	for i := 0; i+1 < len(fields.Content); i += 2 {
		if fields.Content[i].Value == f.Name {
			return nil, fmt.Errorf("there is already a field called %s", f.Name)
		}
	}
	var def yaml.Node
	if err := def.Encode(f); err != nil {
		return nil, err
	}
	fields.Content = append(fields.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: f.Name}, &def)
	return yaml.Marshal(&doc)
}

// TypeYAML writes a type as the file it would be in schema/, fields in
// order, so a type made while the workspace runs reads like one a person
// wrote.
func TypeYAML(t *Type) ([]byte, error) {
	fields := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, f := range t.Fields {
		var def yaml.Node
		if err := def.Encode(f); err != nil {
			return nil, err
		}
		fields.Content = append(fields.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: f.Name}, &def)
	}
	out := struct {
		Name        string     `yaml:"name"`
		Description string     `yaml:"description,omitempty"`
		Title       string     `yaml:"title,omitempty"`
		Hidden      bool       `yaml:"hidden,omitempty"`
		Fields      *yaml.Node `yaml:"fields"`
	}{t.Name, t.Description, t.Title, t.Hidden, fields}
	return yaml.Marshal(out)
}
