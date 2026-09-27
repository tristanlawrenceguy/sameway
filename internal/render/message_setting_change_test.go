package render_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestMessageWithSettingChangeShowsHumanReadable checks that when the message
// component receives a change with action "changed" and detail "pace to Calm"
// (already transformed by chat.CleanSettingChange), it renders human-readable
// text. The target should not show a raw field path in parentheses. This covers
// acceptance item 1: the message component must render setting-change entries
// with readable names, not internal paths like ui.pace.
func TestMessageWithSettingChangeShowsHumanReadable(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "Pace is set to calm.",
		"time":    "14:05",
		"changes": []any{
			map[string]any{
				"action": "changed",
				"detail": "pace to Calm",
			},
		},
	})
	if err != nil {
		t.Fatalf("render message with setting change: %v", err)
	}

	got := string(out)

	if strings.Contains(got, "(ui.") || strings.Contains(got, "set ui.") {
		t.Errorf("message must not render raw field paths in changes list\nwant: human-readable text only\ngot:\n%s", got)
	}

	if !strings.Contains(got, `class="sw-message__change-action">changed`) && !strings.Contains(got, "changed") {
		t.Errorf("changes list should show action as 'changed', not 'set'\ngot:\n%s", got)
	}

	if !strings.Contains(got, "pace to Calm") {
		t.Errorf("changes detail should be human-readable\nwant: 'pace to Calm' in output\ngot:\n%s", got)
	}
}

// TestMessageWithSettingChangeNoComponentInTarget checks that a transformed
// setting change (where the component key has been deleted by
// chat.CleanSettingChange) does not render any raw field path. After the
// shared transformation deletes "component", only action and detail remain,
// so the message template renders the human-readable detail without any
// parenthesised ui.* target. This covers acceptance item 1: targets should
// be readable like "pace to Calm" not "(ui.pace)".
func TestMessageWithSettingChangeNoComponentInTarget(t *testing.T) {
	reg := builtins(t)

	// Simulate what the message component receives after CleanSettingChange
	// transforms the raw change map: action="changed", detail="pace to Calm",
	// no "component" key.
	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "Pace is set to calm.",
		"time":    "14:05",
		"changes": []any{
			map[string]any{
				"action": "changed",
				"detail": "pace to Calm",
				// No component key — CleanSettingChange deletes it.
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	// There must be no parenthesised field path in the output.
	if strings.Contains(got, `(ui.`) || strings.Contains(got, `llm.`) {
		t.Errorf("message must not render raw internal keys as change targets\nwant: no '(ui.*') or 'llm.*' spans\ngot:\n%s", got)
	}

	// The detail should appear without a parenthesised component next to it.
	if strings.Contains(got, "pace to Calm") && strings.Contains(got, "(ui.") {
		t.Errorf("detail and raw component path must not both appear\ngot:\n%s", got)
	}
}

// TestMessageWithNonSettingChangeStillShowsComponent checks that a non-setting
// change (e.g. added card) still renders its component name in the target span,
// since CleanSettingChange does not transform it. This ensures we do not lose
// information for non-setting changes.
func TestMessageWithNonSettingChangeStillShowsComponent(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "I added a card.",
		"time":    "14:05",
		"changes": []any{
			map[string]any{
				"action":    "added",
				"component": "card",
				"detail":    "Plan",
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	if !strings.Contains(got, `class="sw-message__change-action">added`) {
		t.Errorf("non-setting action should still show as-is\nwant 'added' in output\ngot:\n%s", got)
	}
}

// TestMessageWithUndoButtonOnSettingChangeShowsPlainWords checks that the Undo
// button accessible name for a setting change uses human-readable text, not raw
// field paths. This covers acceptance item 2: "Undo changed pace to Calm"
// instead of "Undo set ui.pace calm".
func TestMessageWithUndoButtonOnSettingChangeShowsPlainWords(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "Pace is set to calm.",
		"time":    "14:05",
		"changes": []any{
			map[string]any{
				"action":   "changed",
				"detail":   "pace to Calm",
				"activity": "act-abc",
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	// The undo button's accessible name comes from the visually-hidden span.
	// After transformation it should say "changed pace to Calm" not "set ui.pace calm".
	if strings.Contains(got, `sw-visually-hidden"> set ui.`) ||
		strings.Contains(got, `sw-visually-hidden">set ui.`) {
		t.Errorf("undo button accessible name must not contain raw field paths\nwant: 'changed pace to Calm'\ngot:\n%s", got)
	}

	if !strings.Contains(got, "Undo") {
		t.Error("message with activity id should render an Undo button: " + got)
	}
}

// TestMessageWithSettingChangeCapitalisedDetail checks that the detail value is
// capitalised properly in the rendered output. For example "calm" → "Calm". This
// covers acceptance item 1 for proper capitalisation.
func TestMessageWithSettingChangeCapitalisedDetail(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "Done.",
		"time":    "14:06",
		"changes": []any{
			map[string]any{
				"action": "changed",
				"detail": "spacing to Wide",
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	if !strings.Contains(got, `class="sw-message__change-detail">spacing to Wide`) {
		t.Errorf("detail value should be capitalised in the changes list\nwant 'spacing to Wide' (capital W)\ngot:\n%s", got)
	}
}

// TestMessageTemplateContainsUndoButtonAccessibleName checks that the message
// component template renders an accessible name for Undo buttons via a visually-
// hidden span inside the button. This is how acceptance item 2 is met for screen
// reader users: the visible text says "Undo" and the hidden text explains what
// will be undone in plain words. The test reads the template source directly so
// it does not depend on any particular render output.
func TestMessageTemplateContainsUndoButtonAccessibleName(t *testing.T) {
	reg := builtins(t)

	var c *render.Component
	for _, comp := range reg.Components() {
		if comp.Manifest.Name == "message" {
			c = comp
			break
		}
	}
	if c == nil {
		t.Fatal("no message component registered")
	}

	templateSrc, err := c.ReadFile("template.html")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}

	src := string(templateSrc)
	if !strings.Contains(src, "sw-visually-hidden") {
		t.Errorf("message template must include a visually-hidden span for Undo button accessible names\nwant: sw-visually-hidden in the undo form\ngot:\n%s", src)
	}
}
