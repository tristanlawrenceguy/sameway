package server

import (
	"reflect"
	"testing"
)

// apart adds words only to names that repeat, with the first way that
// differs across all of them; one left without words is the plain one.
func TestApartTellsOnlyRepeatsApart(t *testing.T) {
	names := []string{"Call plumber", "Pay bill", "call  plumber", "Call plumber"}
	ways := [][]string{{"due Fri", "added 1"}, {"due Sat"}, {"due Fri", "added 2"}, {"", "added 3"}}
	got := apart(names, func(i int) []string { return ways[i] })
	want := []string{"added 1", "", "added 2", "added 3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("apart = %q, want %q", got, want)
	}
	ways[3] = []string{"", "added 3"}
	ways[2] = []string{"due Sun", "added 2"}
	if got := apart(names, func(i int) []string { return ways[i] }); !reflect.DeepEqual(got, []string{"due Fri", "", "due Sun", ""}) {
		t.Errorf("one without the day is the plain one: %q", got)
	}
	if got := apart([]string{"A", "B"}, func(int) []string { return []string{"x"} }); got[0] != "" || got[1] != "" {
		t.Errorf("names of their own are left as they are: %q", got)
	}
}

// A block's controls are named after its label, else its component.
func TestBlockNameIsItsLabel(t *testing.T) {
	if got := blockName("collection", map[string]any{"type": "task", "label": "Up next"}); got != "Up next" {
		t.Errorf("blockName = %q, want Up next", got)
	}
	if got := blockName("collection", map[string]any{"type": "task"}); got != "collection" {
		t.Errorf("with no label, blockName = %q, want collection", got)
	}
}
