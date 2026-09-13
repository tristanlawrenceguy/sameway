package store_test

import (
	"encoding/json"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestListReturnsNonNilEmptySlice covers the bug where an empty result set
// was returned as nil, causing JSON marshalling to produce null instead of [].
func TestListReturnsNonNilEmptySlice(t *testing.T) {
	st := open(t)

	// No records created — list should return a non-nil empty slice.
	list, err := st.List("note", store.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list == nil {
		t.Fatal("List returned nil instead of an empty slice when no records match")
	}
	if len(list) != 0 {
		t.Errorf("expected zero records, got %d", len(list))
	}

	// Verify JSON marshalling produces [] not null.
	b, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	if string(b) != "[]" {
		t.Errorf("expected \"[]\", got %s", b)
	}

	// After creating and deleting a record, the slice must still be non-nil.
	rec, err := st.Create("note", map[string]any{"title": "temp"})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Delete("note", rec.ID); err != nil {
		t.Fatal(err)
	}
	list, err = st.List("note", store.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error after delete: %v", err)
	}
	if list == nil {
		t.Fatal("List returned nil after creating and deleting a record")
	}
	if len(list) != 0 {
		t.Errorf("expected zero records after delete, got %d", len(list))
	}

	b, err = json.Marshal(list)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	if string(b) != "[]" {
		t.Errorf("expected \"[]\" after delete cycle, got %s", b)
	}
}

// TestListWithFilterReturnsNonNilEmptySlice covers the case where records
// exist but all are deleted — List must still return a non-nil slice.
func TestListWithFilterReturnsNonNilEmptySlice(t *testing.T) {
	st := open(t)

	// Create one note then delete it so we test an empty-after-populated path.
	rec, err := st.Create("note", map[string]any{"title": "Hello"})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Delete("note", rec.ID); err != nil {
		t.Fatal(err)
	}

	list, err := st.List("note", store.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list == nil {
		t.Fatal("List returned nil instead of an empty slice")
	}
	if len(list) != 0 {
		t.Errorf("expected zero records, got %d", len(list))
	}

	b, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	if string(b) != "[]" {
		t.Errorf("expected \"[]\", got %s", b)
	}
}
