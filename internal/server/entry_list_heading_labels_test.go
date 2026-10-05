package server_test

// Entry list page heading links must not embed raw field labels such as
// "(at …)" and must use natural-language dates without machine-format
// weekday+month patterns (acceptance item 2 of task 0681 / goal 0098).

import (
	"regexp"
	"strings"
	"testing"
)

func TestEntryListPageNoRawAtLabelInHeading(t *testing.T) {
	a, h := newApp(t)

	habitRec, err := a.Store.Create("habit", map[string]any{
		"name":   "Exercise",
		"target": 1.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	rec, err := a.Store.Create("entry", map[string]any{
		"habit":  habitRec.ID,
		"at":     "2026-10-03T14:30:00Z",
		"amount": 3.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry").Body.String()

	rowStart := strings.Index(page, `href="/t/entry/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find entry row in page\n%s", truncate(page))
	}
	rowEnd := strings.Index(page[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of entry row\n%s", truncate(page))
	}
	row := page[rowStart : rowStart+rowEnd]

	// The row must NOT contain the raw field label "at" as context.
	if strings.Contains(row, "sw-visually-hidden") && strings.Contains(strings.ToLower(row), "(at ") {
		t.Errorf("entry list row visually hidden context must not start with '(at '; found in:\n%s", truncate(row))
	}
}

func TestEntryListPageNoMachineFormatDate(t *testing.T) {
	a, h := newApp(t)

	habitRec, err := a.Store.Create("habit", map[string]any{
		"name":   "Exercise 2",
		"target": 1.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	rec, err := a.Store.Create("entry", map[string]any{
		"habit":  habitRec.ID,
		"at":     "2026-10-03T14:30:00Z",
		"amount": 5.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry").Body.String()

	rowStart := strings.Index(page, `href="/t/entry/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find entry row in page\n%s", truncate(page))
	}
	rowEnd := strings.Index(page[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of entry row\n%s", truncate(page))
	}
	row := page[rowStart : rowStart+rowEnd]

	// The date text must NOT match machine-format weekday+month like "Sat 3 Oct".
	if matched, _ := regexp.MatchString(`(?i)[A-Z][a-z]{2}\s+\d{1,2}\s+[A-Z][a-z]{2}`, row); matched {
		t.Errorf("entry list row must not contain machine-format date like 'Sat 3 Oct'; found in:\n%s", truncate(row))
	}

	// The date text should use am/pm (12-hour clock).
	if !strings.Contains(strings.ToLower(row), "am") && !strings.Contains(strings.ToLower(row), "pm") {
		t.Errorf("entry list row date should contain 'am' or 'pm'; found in:\n%s", truncate(row))
	}
}
