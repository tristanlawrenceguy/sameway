package schema

import (
	"gopkg.in/yaml.v3"
)

// A type's own keys, such as hidden, are changed in its YAML as its fields
// are (edit_fields.go), and a type deleted is taken out of the set.

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
