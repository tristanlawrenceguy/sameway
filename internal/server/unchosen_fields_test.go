package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// TestNothingChosenIsNotSaid: a record of any type, made with only what it
// needs, does not list a field nobody chose, its default or a no
// (schema.Unchosen): no "HTTP method POST" on an action, no "Archived no"
// on a habit, no "Type Alarm" on a reminder.
func TestNothingChosenIsNotSaid(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	made := map[string]string{}
	// Twice round, so a type that needs another's record (an entry its
	// habit) is made once that one is.
	for range 2 {
		for _, ty := range a.Types.Types {
			if !ty.Content() || made[ty.Name] != "" {
				continue
			}
			if rec, err := a.Store.Create(ty.Name, least(ty, made)); err == nil {
				made[ty.Name] = rec.ID
			}
		}
	}
	for _, ty := range a.Types.Types {
		if !ty.Content() {
			continue
		}
		if made[ty.Name] == "" {
			t.Errorf("%s could not be made with only what it needs", ty.Name)
			continue
		}
		page := get(t, h, "/t/"+ty.Name+"/"+made[ty.Name]).Body.String()
		for _, f := range ty.Fields {
			if (f.Default != nil || f.Type == "bool") && strings.Contains(page, "<dt>"+f.Display()+"</dt>") {
				t.Errorf("%s's page lists %s, which nobody chose", ty.Name, f.Display())
			}
		}
	}
}

// least is what a record of the type needs to be made: its title and the
// fields it requires.
func least(ty *schema.Type, made map[string]string) map[string]any {
	fields := map[string]any{}
	for _, f := range ty.Fields {
		switch {
		case f.Type == "string" && (f.Required || f.Name == ty.Title):
			fields[f.Name] = "Something"
		case f.Required && f.Type == "datetime":
			fields[f.Name] = "2026-10-20T09:00:00Z"
		case f.Required && f.Type == "ref":
			fields[f.Name] = made[f.To]
		}
	}
	return fields
}
