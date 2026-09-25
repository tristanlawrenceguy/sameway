package schema

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Changing what a type already has is done to its YAML the same way as
// adding: the file keeps its comments and its order, and only the part
// that changes is written again.

// fieldsNode is a type file's root mapping and its fields mapping.
func fieldsNode(src []byte) (*yaml.Node, *yaml.Node, *yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, nil, nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil, nil, fmt.Errorf("the type file is not a mapping")
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "fields" && root.Content[i+1].Kind == yaml.MappingNode {
			return &doc, root, root.Content[i+1], nil
		}
	}
	return &doc, root, nil, nil
}

// ReplaceFieldYAML writes one field's definition again, as f says, where
// it was: a choice added, a label, hidden.
func ReplaceFieldYAML(src []byte, f Field) ([]byte, error) {
	doc, _, fields, err := fieldsNode(src)
	if err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("there is no field called %s", f.Name)
	}
	for i := 0; i+1 < len(fields.Content); i += 2 {
		if fields.Content[i].Value == f.Name {
			var def yaml.Node
			if err := def.Encode(f); err != nil {
				return nil, err
			}
			fields.Content[i+1] = &def
			return yaml.Marshal(doc)
		}
	}
	return nil, fmt.Errorf("there is no field called %s", f.Name)
}

// RemoveFieldYAML takes one field out of a type's file.
func RemoveFieldYAML(src []byte, name string) ([]byte, error) {
	doc, _, fields, err := fieldsNode(src)
	if err != nil {
		return nil, err
	}
	if fields != nil {
		for i := 0; i+1 < len(fields.Content); i += 2 {
			if fields.Content[i].Value == name {
				fields.Content = append(fields.Content[:i], fields.Content[i+2:]...)
				return yaml.Marshal(doc)
			}
		}
	}
	return nil, fmt.Errorf("there is no field called %s", name)
}

// SetTypeYAML sets one key of the type itself, such as hidden, adding it
// when the file does not say, and taking it out when it is set to false.
func SetTypeYAML(src []byte, key string, value bool) ([]byte, error) {
	doc, root, _, err := fieldsNode(src)
	if err != nil {
		return nil, err
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			if !value {
				root.Content = append(root.Content[:i], root.Content[i+2:]...)
			} else {
				root.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}
			}
			return yaml.Marshal(doc)
		}
	}
	if value {
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
	}
	return yaml.Marshal(doc)
}

// Remove takes a type out of the set: a type deleted while the workspace
// runs.
func (s *Set) Remove(name string) {
	delete(s.byName, name)
	for i, t := range s.Types {
		if t.Name == name {
			s.Types = append(s.Types[:i], s.Types[i+1:]...)
			return
		}
	}
}
