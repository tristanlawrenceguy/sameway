package chat_test

import (
	"strings"
	"testing"
)

// A block added beside one of the same component over the same records is
// said with which one and how to change it instead; one over other records
// says nothing.
func TestAddingBesideASimilarBlockSaysSo(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	first, _ := svc.Call("add_component", []byte(`{"component":"collection","props":{"type":"task","label":"Due this week"}}`))
	if strings.Contains(first, "already had") {
		t.Fatalf("the first says nothing of others: %s", first)
	}
	second, _ := svc.Call("add_component", []byte(`{"component":"collection","props":{"type":"task","label":"My tasks"}}`))
	if !strings.Contains(second, "This tab already had a collection of the same records") || !strings.Contains(second, "update_component") {
		t.Errorf("the second says the first and what to do: %s", second)
	}
	other, _ := svc.Call("add_component", []byte(`{"component":"collection","props":{"type":"note","label":"Notes"}}`))
	if strings.Contains(other, "already had") {
		t.Errorf("other records are not similar: %s", other)
	}
}
