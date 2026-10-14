package render_test

import (
	"strings"
	"testing"
)

// TestErrorMessageAuthorIsUserFriendly renders an error-role message without
// setting an explicit author and asserts that the default label is a person-
// friendly name ("Sameway") rather than machine-language ("System"). This covers
// acceptance item 2: failed assistant messages on /chat show a user-friendly
// label instead of "System". Acceptance item 1 (no page shows "System did not
// finish") starts here — the template must not produce "System" for error role.
func TestErrorMessageAuthorIsUserFriendly(t *testing.T) {
	t.Parallel()
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "error",
		"content": "Could not reach the model.",
	})
	if err != nil {
		t.Fatalf("error message must render without error: %v", err)
	}

	got := string(out)

	if strings.Contains(got, ">System</span>") {
		t.Errorf("error-role message must not default author to \"System\" (machine-language);\nwant a person-friendly label like \"Sameway\"\ngot:\n%s", got)
	}

	if !strings.Contains(got, `>Sameway</span>`) && !strings.Contains(got, ">Sameway") {
		t.Errorf("error-role message must default author to \"Sameway\" when no explicit author is set;\ngot:\n%s", got)
	}
}

// TestErrorMessageFailureIndicatorIsPlainLanguage renders an error-role message
// and asserts that the failure indicator uses plain language instead of the raw
// Go-level system phrase "did not finish". This covers acceptance item 1: no
// page displays "System did not finish" as visible content. The change is from
// `<span class="sw-message__failed">did not finish</span>` to something like
// `<span class="sw-message__failed">could not complete this</span>`.
func TestErrorMessageFailureIndicatorIsPlainLanguage(t *testing.T) {
	t.Parallel()
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "error",
		"content": "Could not reach the model at http://localhost:11434.",
	})
	if err != nil {
		t.Fatalf("error message must render without error: %v", err)
	}

	got := string(out)

	if strings.Contains(got, "did not finish") {
		t.Errorf("error-role message must not show machine-language \"did not finish\";\nwant plain language like \"could not complete this\"\ngot:\n%s", got)
	}

	if !strings.Contains(got, `class="sw-message__failed"`) {
		t.Errorf("error-role message must still render the failure indicator span with class sw-message__failed;\ngot:\n%s", got)
	}

	// The failure text should contain a user-friendly phrase.
	if !strings.Contains(got, "could not complete this") {
		t.Errorf("error-role message must show plain-language failure text like \"could not complete this\";\ngot:\n%s", got)
	}
}

// TestErrorMessageContainsNoMachineLanguage asserts that the rendered error
// message does not produce both "System" as author AND "did not finish" as the
// failure text — the exact machine-language combination reported in backlog 0669.
// The template renders these as separate spans, so checking for the literal
// phrase alone is insufficient; we must check that neither piece appears together
// as visible text on the page. This covers acceptance item 1.
func TestErrorMessageContainsNoMachineLanguage(t *testing.T) {
	t.Parallel()
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "error",
		"content": "Something went wrong.",
	})
	if err != nil {
		t.Fatalf("error message must render without error: %v", err)
	}

	got := string(out)

	// The visible text combination "System did not finish" is what users see.
	// Even though HTML tags separate them, both pieces appear on the page and
	// a screen reader will read them in sequence as machine-language.
	hasSystemAuthor := strings.Contains(got, ">System</span>")
	hasDidNotFinish := strings.Contains(got, "did not finish")

	if hasSystemAuthor && hasDidNotFinish {
		t.Errorf("error message must never show the visible combination \"System did not finish\";"+
			" either the author or the failure text (or both) uses machine-language\nwant: person-friendly labels like \"Sameway could not complete this\"\ngot:\n%s", got)
	}
}

// TestErrorMessageWithCustomAuthorPreservesOverride renders an error-role
// message with an explicit author and asserts that custom authors are still
// honoured. This ensures the fix does not break the existing author override
// behaviour for non-error messages or when a specific author is set.
func TestErrorMessageWithCustomAuthorPreservesOverride(t *testing.T) {
	t.Parallel()
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "error",
		"content": "Could not reach the model.",
		"author":  "CustomBot",
	})
	if err != nil {
		t.Fatalf("error message with custom author must render: %v", err)
	}

	got := string(out)

	if !strings.Contains(got, ">CustomBot</span>") {
		t.Errorf("explicit author \"CustomBot\" should appear in output;\ngot:\n%s", got)
	}

	if strings.Contains(got, "System") || strings.Contains(got, "Sameway") {
		t.Errorf("explicit author must override the default; expected no system/Sameway label\nwant: CustomBot\ngot:\n%s", got)
	}
}
