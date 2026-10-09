package ui

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
)

type manifestProps struct {
	Props schema `json:"props"`
}

type schema struct {
	Type       string            `json:"type"`
	Required   []string          `json:"required"`
	Enum       []string          `json:"enum"`
	Properties map[string]schema `json:"properties"`
}

func manifest(t *testing.T, name string) schema {
	t.Helper()
	b, err := design.FS.ReadFile("components/" + name + "/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m manifestProps
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m.Props
}

// Every builder field is a prop its manifest has, of the same kind; every
// prop the manifest requires has a field. A prop the builder leaves out is
// allowed: the map stays the way to give it.
func TestBuildersMatchManifests(t *testing.T) {
	for _, p := range []Part{Button{}, Link{}, Alert{}, TextField{}, Empty{}, Status{}} {
		checkFields(t, p.Component(), reflect.TypeOf(p), manifest(t, p.Component()))
	}
}

func checkFields(t *testing.T, where string, rt reflect.Type, s schema) {
	have := map[string]bool{}
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("prop"), ",")
		have[name] = true
		def, ok := s.Properties[name]
		if !ok {
			t.Errorf("%s: field %s is prop %q, which the manifest does not have", where, f.Name, name)
			continue
		}
		if got := kindOf(f.Type); got != def.Type {
			t.Errorf("%s: field %s is a %s, the manifest's %s is a %s", where, f.Name, got, name, def.Type)
		}
		if def.Type == "object" {
			checkFields(t, where+"."+name, f.Type.Elem(), def)
		}
	}
	for _, r := range s.Required {
		if !have[r] {
			t.Errorf("%s: the manifest requires %q and the builder has no field for it", where, r)
		}
	}
}

func kindOf(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int:
		return "integer"
	case reflect.Struct:
		return "object"
	}
	return t.Kind().String()
}

// Every enum a builder's prop has is the manifest's, word for word, and
// every enum prop of a builder's component is in Enums.
func TestEnumsMatchManifests(t *testing.T) {
	for key, words := range Enums {
		def := manifest(t, key[0]).Properties[key[1]]
		if !slices.Equal(def.Enum, words) {
			t.Errorf("%s %s: Enums says %v, the manifest %v", key[0], key[1], words, def.Enum)
		}
	}
	for _, p := range []Part{Button{}, Link{}, Alert{}, TextField{}, Empty{}, Status{}} {
		for name, def := range manifest(t, p.Component()).Properties {
			if _, ok := Enums[[2]string{p.Component(), name}]; len(def.Enum) > 0 && !ok {
				t.Errorf("%s %s has an enum the builder does not list in Enums", p.Component(), name)
			}
		}
	}
}

func TestPropsLeaveOutWhatIsNotSet(t *testing.T) {
	got := Button{Label: "Save", Variant: Secondary, Pressed: Bool(false)}.Props()
	want := map[string]any{"label": "Save", "variant": Secondary, "pressed": false}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	got = Empty{Message: "None", Action: &EmptyAction{Href: "#new", Label: "Make one"}}.Props()
	want = map[string]any{"message": "None", "action": map[string]any{"href": "#new", "label": "Make one"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
