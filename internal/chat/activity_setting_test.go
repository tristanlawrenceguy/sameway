package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestSummariseSettingChangeIsHumanReadable checks that a setting-change
// entry logged by Record() stores a human-readable summary instead of raw
// field paths. "set ui.pace calm" becomes "Assistant changed pace to Calm".
// This covers acceptance items 1 and 4: the stored summary on /activity
// must not contain machine-language identifiers.
func TestSummariseSettingChangeIsHumanReadable(t *testing.T) {
	svc := newFullService(t)

	id := chat.Record(svc.Store, "assistant", chat.Change{
		Action:    "set",
		Component: "ui.pace",
		Detail:    "calm",
	})

	if id == "" {
		t.Fatal("expected activity record to be created")
	}

	entry, err := svc.Store.Get(chat.ActivityType, id)
	if err != nil {
		t.Fatalf("activity not stored: %v", err)
	}

	summary, _ := entry.Fields["summary"].(string)

	want := "Assistant changed pace to Calm"
	if summary != want {
		t.Errorf("setting-change summary should use human-readable names\n\nwant: Assistant changed pace to Calm\n\ngot: %q", summary)
	}

	if strings.Contains(summary, "ui.pace") || strings.Contains(summary, `set ui.`) {
		t.Errorf("summary must not contain internal field paths like 'ui.pace'\n\nwant: no dotted key in %q", summary)
	}
}

// TestSummariseSettingChangeNoInternalFieldNames checks that no raw field
// path like ui.pace, ui.spacing, or ui.text appears in any setting-change
// summary. This covers acceptance item 4.
func TestSummariseSettingChangeNoInternalFieldNames(t *testing.T) {
	svc := newFullService(t)

	tests := []struct {
		key   string
		value string
	}{
		{"ui.pace", "quick"},
		{"ui.spacing", "wide"},
		{"ui.text", "large"},
	}

	for _, tc := range tests {
		id := chat.Record(svc.Store, "assistant", chat.Change{
			Action:    "set",
			Component: tc.key,
			Detail:    tc.value,
		})
		if id == "" {
			t.Fatalf("expected activity record for %q", tc.key)
		}

		entry, _ := svc.Store.Get(chat.ActivityType, id)
		summary, _ := entry.Fields["summary"].(string)

		if containsDotPath(summary) {
			t.Errorf("summary must not contain internal field path like ui.pace\n\nwant: no dotted key in %q", summary)
		}
	}
}

// TestSummariseSettingChangeHumanActor checks that a human actor is shown as
// "You" in the setting-change summary.
func TestSummariseSettingChangeHumanActor(t *testing.T) {
	svc := newFullService(t)

	id := chat.Record(svc.Store, "human", chat.Change{
		Action:    "set",
		Component: "ui.spacing",
		Detail:    "wide",
	})

	entry, _ := svc.Store.Get(chat.ActivityType, id)
	summary, _ := entry.Fields["summary"].(string)

	want := "You changed spacing to Wide"
	if summary != want {
		t.Errorf("human actor should be 'You'\n\nwant: You changed spacing to Wide\n\ngot: %q", summary)
	}
}

// TestSummariseNonSettingChangeIsUnchanged checks that non-setting changes
// (like adding a block or updating a record) are not affected by the human-
// readable transformation. The actor, action, component and detail should be
// joined as before.
func TestSummariseNonSettingChangeIsUnchanged(t *testing.T) {
	svc := newFullService(t)

	id := chat.Record(svc.Store, "assistant", chat.Change{
		Action:    "added",
		Component: "heading",
		Detail:    "Shopping",
	})

	entry, _ := svc.Store.Get(chat.ActivityType, id)
	summary, _ := entry.Fields["summary"].(string)

	want := "Assistant added heading Shopping"
	if summary != want {
		t.Errorf("non-setting changes should use the original format\n\nwant: Assistant added heading Shopping\n\ngot: %q", summary)
	}
}

// containsDotPath reports whether the string looks like it contains a dotted
// internal field path (e.g. ui.pace, llm.model).
func containsDotPath(s string) bool {
	for _, prefix := range []string{"ui.", "llm."} {
		if idx := strings.Index(s, prefix); idx >= 0 {
			end := idx + len(prefix)
			if end < len(s) && s[end] != ' ' && s[end] != ',' {
				return true
			}
		}
	}
	return false
}
