package schema_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// ValidationError header says "could not save:" instead of "invalid record:".
func TestValidationErrorHeader(t *testing.T) {
	typ := parseEveryType()
	_, err := typ.Normalize(map[string]any{}) // missing required field triggers validation error
	if err == nil || !strings.Contains(err.Error(), "could not save:") {
		t.Errorf("validation error header should say 'could not save:', got: %v", err)
	}
}

// Missing required field says "is missing" instead of "is required".
func TestMissingRequiredFieldSaysIsMissing(t *testing.T) {
	typ := parseEveryType()
	_, err := typ.Normalize(map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "is missing") {
		t.Errorf("missing required field should say 'is missing', got: %v", err)
	}
}

// Non-text into a string field says "must be a word or sentence".
func TestCoerceNonTextSaysMustBeWord(t *testing.T) {
	typ := parseEveryType()
	_, err := typ.Normalize(map[string]any{"name": 123})
	if err == nil || !strings.Contains(err.Error(), "must be a word or sentence") {
		t.Errorf("non-text into string should say 'must be a word or sentence', got: %v", err)
	}
}

// Oversized text says "too long" instead of "must be at most".
func TestOversizedTextSaysTooLong(t *testing.T) {
	typ := parseEveryType()
	long := strings.Repeat("x", 20)
	_, err := typ.Normalize(map[string]any{"name": long})
	if err == nil || !strings.Contains(err.Error(), "too long") {
		t.Errorf("oversized text should say 'too long', got: %v", err)
	}
}

// Bad enum value says "pick one:" instead of "must be one of".
func TestBadEnumSaysPickOne(t *testing.T) {
	typ := parseEveryType()
	_, err := typ.Normalize(map[string]any{"name": "x", "kind": "z"})
	if err == nil || !strings.Contains(err.Error(), "pick one:") {
		t.Errorf("bad enum should say 'pick one:', got: %v", err)
	}
}

// Bad bool says "must be yes or no" instead of "must be true or false".
func TestBadBoolSaysMustBeYesOrNo(t *testing.T) {
	typ := parseEveryType()
	_, err := typ.Normalize(map[string]any{"name": "ok", "on": "maybe"})
	if err == nil || !strings.Contains(err.Error(), "must be yes or no") {
		t.Errorf("bad bool should say 'must be yes or no', got: %v", err)
	}
}

// Bad JSON says "not a JSON object or array" instead of "must be valid JSON".
func TestBadJSONSaysNotAJSONObject(t *testing.T) {
	typ := parseEveryType()
	_, err := typ.Normalize(map[string]any{"name": "ok", "meta": "not json"})
	if err == nil || !strings.Contains(err.Error(), "not a JSON object or array") {
		t.Errorf("bad json should say 'not a JSON object or array', got: %v", err)
	}
}

// Non-list into list field says "must be a list".
func TestNonListSaysMustBeAList(t *testing.T) {
	typ := parseEveryType()
	_, err := typ.Normalize(map[string]any{"name": "ok", "tags": 123})
	if err == nil || !strings.Contains(err.Error(), "must be a list") {
		t.Errorf("non-list into list should say 'must be a list', got: %v", err)
	}
}

// Bad datetime says "cannot read" instead of "could not read".
func TestBadDatetimeSaysCannotRead(t *testing.T) {
	typ := parseEveryType()
	_, err := typ.Normalize(map[string]any{"name": "ok", "when": "soon"})
	if err == nil || !strings.Contains(err.Error(), "cannot read") {
		t.Errorf("bad datetime should say 'cannot read', got: %v", err)
	}
}

// List items also report coercion errors with "item".
func TestListItemErrorsSayItem(t *testing.T) {
	typ := parseEveryType()
	_, err := typ.Normalize(map[string]any{"name": "ok", "tags": []any{123}})
	if err == nil || !strings.Contains(err.Error(), "item") {
		t.Errorf("bad item in list should say 'item', got: %v", err)
	}
}

func parseEveryType() *schema.Type {
	typ, err := schema.Parse([]byte(everyType))
	if err != nil {
		panic(err)
	}
	return typ
}
