package server_test

// Notes list rows must show dates in natural language, not machine format.
// "Updated Today 00:52" becomes "Today at 12:52am". No "Updated" prefix.
// Acceptance items 1 of task 0258 (backlog 0610).

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// TestNoteListShowsNaturalDate verifies that the notes listing page shows
// dates in natural language like "Today at …am/pm" or "Two days ago at …pm"
// instead of machine format. It creates a note via the API, fetches /t/note,
// extracts the row, and asserts on the date text inside it. (Acceptance 1.)
func TestNoteListShowsNaturalDate(t *testing.T) {
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Test note for date format",
	}), &rec)

	body := get(t, h, "/t/note").Body.String()

	rowStart := strings.Index(body, `href="/t/note/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find note row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of note row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	// The word "Updated" must NOT appear as a field-name prefix before the timestamp.
	if strings.Contains(row, "Updated") {
		t.Errorf("note list row must not show 'Updated' prefix; found in:\n%s", truncate(row))
	}

	// The date text should be inside a span with class sw-muted and contain
	// am or pm (12-hour clock), not a 24-hour format like "00:52" or "23:17".
	// Extract the sw-muted span from the row.
	mutedStart := strings.Index(row, `class="sw-muted"`)
	if mutedStart < 0 {
		t.Errorf("note list row should contain a span with class=\"sw-muted\" for the date; found in:\n%s", truncate(row))
		return
	}

	// Find the closing > of the span start tag.
	mutedEnd := strings.Index(row[mutedStart:], `>`)
	if mutedEnd < 0 {
		t.Errorf("could not find end of sw-muted span; found in:\n%s", truncate(row))
		return
	}

	// Find the closing </span>.
	spanContent := row[mutedStart+mutedEnd+1:]
	closeSpan := strings.Index(spanContent, "</span>")
	if closeSpan < 0 {
		t.Errorf("could not find end of sw-muted span; found in:\n%s", truncate(row))
		return
	}
	content := spanContent[:closeSpan]

	// The text should contain am or pm (case-insensitive) for a 12-hour clock.
	if !strings.Contains(strings.ToLower(content), "am") && !strings.Contains(strings.ToLower(content), "pm") {
		t.Errorf("date text should use 12-hour format with am/pm; expected natural language like 'Today at …am', got: %s", content)
	}

	// The date should NOT contain machine-format patterns like a weekday abbreviation
	// followed by day and month (e.g. "Sat 26 Sep").
	if strings.Contains(content, "Updated") {
		t.Errorf("date text must not contain 'Updated' prefix; got: %s", content)
	}

	// The date should NOT match the pattern of machine-format weekday+month like
	// "Sat 26 Sep" or "Mon 15 Sep". These patterns force mental calculation.
	if matched, _ := regexp.MatchString(`(?i)[A-Z][a-z]{2}\s+\d{1,2}\s+[A-Z][a-z]{2}`, content); matched {
		t.Errorf("date text must not contain machine-format weekday+month like 'Sat 26 Sep'; got: %s", content)
	}

	// The date should NOT have a 24-hour clock pattern (HH:MM with leading zero)
	// that is NOT part of an am/pm format. A pattern like " 04:03" or " 12:53"
	// without preceding am/pm is machine format.
	if strings.Contains(content, "Today at") || strings.Contains(content, "Yesterday at") || strings.Contains(content, "ago at") {
		// Natural language with "at" — the colon pattern after "at" should be fine.
	} else if matched, _ := regexp.MatchString(`\s\d{1,2}:\d{2}\b`, content); matched {
		t.Errorf("date text should not contain bare 24-hour time like ' 04:03'; got: %s", content)
	}
}

// TestNoteListNoMachineFormatDatePattern verifies that notes list rows do not
// display machine-format timestamps such as "Sat 26 Sep" or "Mon 15 Sep". The
// developer must use natural language like "Two days ago" instead. (Acceptance 1.)
func TestNoteListNoMachineFormatDatePattern(t *testing.T) {
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Old note for pattern check",
	}), &rec)

	body := get(t, h, "/t/note").Body.String()

	rowStart := strings.Index(body, `href="/t/note/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find note row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of note row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	// Machine format: "Updated Mon 15 Sep" or "Mon 15 Sep". We check that the
	// row does NOT contain a pattern like weekday abbreviation + day + month.
	if matched, _ := regexp.MatchString(`(?i)[A-Z][a-z]{2}\s+\d{1,2}\s+[A-Z][a-z]{2}`, row); matched {
		t.Errorf("note list row must not contain machine-format date like 'Sat 26 Sep'; found in:\n%s", truncate(row))
	}

	// Also verify no "Updated" prefix exists.
	if strings.Contains(row, ">Updated ") {
		t.Errorf("note list row must not show 'Updated' as field-name prefix; found in:\n%s", truncate(row))
	}
}
