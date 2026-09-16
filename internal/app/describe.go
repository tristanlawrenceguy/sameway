package app

import (
	"fmt"
	"sort"
	"strings"
)

// Parts names the sections of a Description that can be read on their own.
var Parts = []string{"types", "components", "arrangements", "tools", "routes", "llm"}

// Part is one section of the description, or one named item in a section,
// so an agent reads what it needs without the whole document: the fields of
// one type before creating a record, or the routes alone. Every surface that
// serves the description (HTTP, MCP, the command line) cuts it here, so a
// part means the same thing everywhere.
func (d Description) Part(part, name string) (any, error) {
	if part == "" {
		return d, nil
	}
	var items []string
	var found any
	switch part {
	case "llm":
		if name == "" {
			return d.LLM, nil
		}
	case "routes":
		if name == "" {
			return d.Routes, nil
		}
		for k, v := range d.Routes {
			items = append(items, k)
			if k == name {
				found = v
			}
		}
	case "types":
		if name == "" {
			return d.Types, nil
		}
		for _, t := range d.Types {
			items = append(items, t.Name)
			if t.Name == name {
				found = t
			}
		}
	case "components":
		if name == "" {
			return d.Components, nil
		}
		for _, c := range d.Components {
			items = append(items, c.Name)
			if c.Name == name {
				found = c
			}
		}
	case "arrangements":
		if name == "" {
			return d.Arrangements, nil
		}
		for _, a := range d.Arrangements {
			items = append(items, a.Name)
			if a.Name == name {
				found = a
			}
		}
	case "tools":
		if name == "" {
			return d.Tools, nil
		}
		for _, t := range d.Tools {
			items = append(items, t.Name)
			if t.Name == name {
				found = t
			}
		}
	default:
		return nil, fmt.Errorf("describe has no part %q; the parts are %s", part, strings.Join(Parts, ", "))
	}
	if found == nil {
		sort.Strings(items)
		return nil, fmt.Errorf("describe has no %s named %q; the names are %s", strings.TrimSuffix(part, "s"), name, strings.Join(items, ", "))
	}
	return found, nil
}
