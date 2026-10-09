package render

import (
	"encoding/json"
	"html/template"
	"sync"
)

// Loading a component is the same work for every registry that loads it,
// and a program opens one registry per workspace: a test binary hundreds.
// What it parses is kept once per program, by the text it was parsed
// from, and handed to each registry: a compiled props schema as it is,
// since it is only read, and a template as a clone given the registry's
// own functions (the master is never run, so it never escapes).
var parsedOnce sync.Map // key -> *propSchema or *template.Template

// propsOnce is compileProps, once per program for each schema.
func propsOnce(name string, raw json.RawMessage) (*propSchema, error) {
	key := "props\x00" + name + "\x00" + string(raw)
	if v, ok := parsedOnce.Load(key); ok {
		return v.(*propSchema), nil
	}
	ps, err := compileProps(name, raw)
	if err != nil {
		return nil, err
	}
	parsedOnce.Store(key, ps)
	return ps, nil
}

// templateOnce is a component's template with funcs, parsed once per
// program for each text and cloned for each use.
func templateOnce(name, body string, funcs template.FuncMap) (*template.Template, error) {
	key := "tmpl\x00" + name + "\x00" + body
	v, ok := parsedOnce.Load(key)
	if !ok {
		t, err := template.New(name).Funcs(funcs).Option("missingkey=zero").Parse(body)
		if err != nil {
			return nil, err
		}
		v, _ = parsedOnce.LoadOrStore(key, t)
	}
	t, err := v.(*template.Template).Clone()
	if err != nil {
		return nil, err
	}
	return t.Funcs(funcs), nil
}
