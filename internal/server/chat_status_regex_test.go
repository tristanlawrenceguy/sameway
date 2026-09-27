package server

import "testing"

func TestReplacePathsDebug(t *testing.T) {
	result := internalPath.ReplaceAllStringFunc("/t/note/abc123", func(m string) string {
		return "[REPLACED]"
	})
	if result != "[REPLACED]" {
		t.Errorf("expected [REPLACED], got %q", result)
	}
}
