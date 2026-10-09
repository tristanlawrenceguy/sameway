package server_test

// Notes list rows must show dates in natural language, not machine format.
// "Updated Today 00:52" becomes "Today at 12:52am". No "Updated" prefix.
// Acceptance items 1 of task 0258 (backlog 0610).

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// TestNoteListShowsNaturalDate verifies that the notes listing page shows
// dates in natural language like "Today at …am/pm" or "Two days ago at …pm"
// instead of machine format. It creates a note via the API, fetches /t/note,
// extracts the row, and asserts on the date text inside it. (Acceptance 1.)
func TestNoteListShowsNaturalDate(t *testing.T) {
	t.Parallel()
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
	closeSpan := strings.Index(spanContent, "</time>")
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
	t.Parallel()
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
	if matched, _ := regexp.MatchString(`(?i)[A-Z][a-z]{2}\s+\d{1,2}\s+[A-Z][a-z]{2}`, visibleText(row)); matched {
		t.Errorf("note list row must not contain machine-format date like 'Sat 26 Sep'; found in:\n%s", truncate(row))
	}

	// Also verify no "Updated" prefix exists.
	if strings.Contains(row, ">Updated ") {
		t.Errorf("note list row must not show 'Updated' as field-name prefix; found in:\n%s", truncate(row))
	}
}

// TestEntryListShowsNaturalDate verifies that the entries listing page shows
// dates in natural language like "Today at …am/pm" or "Two days ago at …pm"
// instead of machine format. It creates an entry via the API, fetches /t/entry,
// extracts the row, and asserts on the date text inside it. (Acceptance 1 of task 0260.)
func TestEntryListShowsNaturalDate(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	// Create a habit first (entries require a habit ref).
	var habit struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/habit", map[string]any{
		"name": "Test Habit",
	}), &habit)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/entry", map[string]any{
		"note":  "Test entry for date format",
		"habit": habit.ID,
		"at":    when.Store(time.Now(), false),
	}), &rec)

	body := get(t, h, "/t/entry").Body.String()

	rowStart := strings.Index(body, `href="/t/entry/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find entry row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of entry row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	// The word "Updated" must NOT appear as a field-name prefix before the timestamp.
	if strings.Contains(row, "Updated") {
		t.Errorf("entry list row must not show 'Updated' prefix; found in:\n%s", truncate(row))
	}

	// Extract the sw-when span from the row (dayFact renders into this class).
	whenStart := strings.Index(row, `<time class="sw-when`)
	if whenStart < 0 {
		t.Errorf("entry list row should contain a span with class=\"sw-when\" for the date; found in:\n%s", truncate(body))
		return
	}

	// Find the closing > of the span start tag.
	whenEnd := strings.Index(row[whenStart:], `>`)
	if whenEnd < 0 {
		t.Errorf("could not find end of sw-when span; found in:\n%s", truncate(body))
		return
	}

	// Find the closing </span>.
	spanContent := row[whenStart+whenEnd+1:]
	closeSpan := strings.Index(spanContent, "</time>")
	if closeSpan < 0 {
		t.Errorf("could not find end of sw-when span; found in:\n%s", truncate(body))
		return
	}
	content := spanContent[:closeSpan]

	// The text should contain am or pm (case-insensitive) for a 12-hour clock.
	if !strings.Contains(strings.ToLower(content), "am") && !strings.Contains(strings.ToLower(content), "pm") {
		t.Errorf("date text should use 12-hour format with am/pm; expected natural language like 'Today at …am', got: %s", content)
	}

	// The date should NOT match the pattern of machine-format weekday+month like
	// "Sat 26 Sep" or "Mon 15 Sep". These patterns force mental calculation.
	if matched, _ := regexp.MatchString(`(?i)[A-Z][a-z]{2}\s+\d{1,2}\s+[A-Z][a-z]{2}`, content); matched {
		t.Errorf("date text must not contain machine-format weekday+month like 'Sat 26 Sep'; got: %s", content)
	}

	// The date should NOT have a bare 24-hour clock pattern (HH:MM with leading zero).
	if strings.Contains(content, "Today at") || strings.Contains(content, "Yesterday at") || strings.Contains(content, "ago at") {
		// Natural language with "at" — the colon pattern after "at" should be fine.
	} else if matched, _ := regexp.MatchString(`\s\d{1,2}:\d{2}\b`, content); matched {
		t.Errorf("date text should not contain bare 24-hour time like ' 04:03'; got: %s", content)
	}
}

// TestAllListPagesNoMachineFormatDate verifies that record type list pages for
// types without datetime fields (notes, actions, projects, people, files) show
// dates in natural language without "Updated" as a label prefix. (Acceptance 2–4.)
func TestAllListPagesNoMachineFormatDate(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	tests := []struct {
		apiPath string // path to create a record
		payload map[string]any
		listURL string // list page URL
		rowHref string // href pattern in the row
	}{
		{"/api/note", map[string]any{"title": "Note for date format"}, "/t/note", `/t/note/`},
		{"/api/action", map[string]any{"title": "Action item for date check"}, "/t/action", `/t/action/`},
		{"/api/project", map[string]any{"title": "Project for date format"}, "/t/project", `/t/project/`},
		{"/api/person", map[string]any{"name": "Person for date format"}, "/t/person", `/t/person/`},
	}

	for _, tc := range tests {
		t.Run(tc.listURL, func(t *testing.T) {
			var rec struct{ ID string }
			decode(t, postJSON(t, h, http.MethodPost, tc.apiPath, tc.payload), &rec)

			body := get(t, h, tc.listURL).Body.String()

			rowStart := strings.Index(body, tc.rowHref+rec.ID)
			if rowStart < 0 {
				t.Fatalf("could not find record row in page\n%s", truncate(body))
			}
			rowEnd := strings.Index(body[rowStart:], "</li>")
			if rowEnd < 0 {
				t.Fatalf("could not find end of record row\n%s", truncate(body))
			}
			row := body[rowStart : rowStart+rowEnd]

			// The word "Updated" must NOT appear as a field-name prefix.
			if strings.Contains(row, ">Updated ") {
				t.Errorf("list row must not show 'Updated' prefix; found in:\n%s", truncate(row))
			}

			// No machine-format weekday+month pattern like "Sat 26 Sep".
			if matched, _ := regexp.MatchString(`(?i)[A-Z][a-z]{2}\s+\d{1,2}\s+[A-Z][a-z]{2}`, visibleText(row)); matched {
				t.Errorf("list row must not contain machine-format date like 'Sat 26 Sep'; found in:\n%s", truncate(row))
			}

			// For types with datetime fields (not tested here since these don't have them),
			// the sw-when class would be checked for am/pm. These types use facts() which
			// already uses when.Relative().
		})
	}

	// Also check files — they require a workspace to create, so skip if not available.
	var fileRec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/file", map[string]any{
		"title": "File for date format",
	}), &fileRec)
	if fileRec.ID != "" {
		body := get(t, h, "/t/file").Body.String()
		rowStart := strings.Index(body, `/t/file/`+fileRec.ID)
		if rowStart >= 0 {
			rowEnd := strings.Index(body[rowStart:], "</li>")
			if rowEnd > 0 {
				row := body[rowStart : rowStart+rowEnd]
				if strings.Contains(row, ">Updated ") {
					t.Errorf("file list row must not show 'Updated' prefix; found in:\n%s", truncate(row))
				}
				if matched, _ := regexp.MatchString(`(?i)[A-Z][a-z]{2}\s+\d{1,2}\s+[A-Z][a-z]{2}`, visibleText(row)); matched {
					t.Errorf("file list row must not contain machine-format date like 'Sat 26 Sep'; found in:\n%s", truncate(row))
				}
			}
		}
	}
}

// visibleText is what a row shows, its tags and their attributes left out:
// the full date a <time> holds for a pointer is not what the row says.
func visibleText(row string) string {
	return regexp.MustCompile(`<[^>]*>`).ReplaceAllString(row, " ")
}
