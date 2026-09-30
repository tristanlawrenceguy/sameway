package app

import (
	"fmt"
	"sort"
	"strings"
)

// Parts names the sections of a Description that can be read on their own.
var Parts = []string{"types", "components", "arrangements", "tools", "routes", "llm", "index", "full"}

// Part is one section of the description, or one named item in a section,
// so an agent reads what it needs without the whole document: the fields of
// one type before creating a record, or the routes alone. Every surface that
// serves the description (HTTP, MCP, the command line) cuts it here, so a
// part means the same thing everywhere.
//
// With nothing asked it is the index, a few kilobytes; "full" is the whole
// description, and a name without a part is looked for in every part.
// Components come compact (the props a writer gives and one example)
// unless full is asked, as ?full=1 does over HTTP.
func (d Description) Part(part, name string) (any, error) { return d.part(part, name, false) }

// FullPart is Part with components whole, manifest and all.
func (d Description) FullPart(part, name string) (any, error) { return d.part(part, name, true) }

func (d Description) part(part, name string, full bool) (any, error) {
	switch {
	case part == "" && name == "" && full, part == "full":
		return d, nil
	case part == "" && name == "", part == "index":
		return d.Index(), nil
	case part == "":
		for _, p := range []string{"components", "types", "tools", "routes", "arrangements"} {
			if v, err := d.part(p, name, full); err == nil {
				return v, nil
			}
		}
		return nil, fmt.Errorf("describe has no part, component, type, tool, route or arrangement named %q; the parts are %s, and the index (GET /api/describe, or describe with no arguments) names the rest", name, strings.Join(Parts, ", "))
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
		if name == "" && full {
			return d.Components, nil
		}
		var list []ComponentSummary
		for _, c := range d.Components {
			items = append(items, c.Name)
			list = append(list, ComponentSummary{c.Name, c.Description, c.Use})
			if c.Name == name && full {
				found = c
			} else if c.Name == name {
				found = c.Compact()
			}
		}
		if name == "" {
			return list, nil
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
