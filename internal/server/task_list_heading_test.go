package server_test

import (
	"strings"
	"testing"
)

// Task list heading links contain only the record name — no "(added ..." or
// "(due ..." text inside any <a> tag. The words that tell a row apart are on
// an aria-label for unique accessible names (WCAG 2.4.9), not visible in the
// link's text content. This covers acceptance item 1 of task 0690.
func TestTaskListHeadingLinksHaveNoRawFieldLabelsInVisibleText(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	overdue, _ := seedTwins(t, h)

	page := get(t, h, "/t/task").Body.String()

	// The current buggy code puts apartHTML (the visually-hidden span with
	// told text like "(due ...)" or "(added ...)") INSIDE the <a> tag:
	//   ><a ...>Call plumber<span class="sw-visually-hidden">(due ...)</span></a>
	// After the fix it should be OUTSIDE the <a>:
	//   ><a aria-label="...">Title</a><span class="sw-visually-hidden">(told)</span>

	if strings.Contains(page, `>Call plumber<span class="sw-visually-hidden"> (`+overdue+`)</span></a>`) {
		t.Errorf("the words that tell a row apart are still inside the <a> tag; they should be outside with aria-label on the link instead\n%s", truncate(page))
	}

	// After the fix: apartHTML is after </a> and before </hN>.
	if !strings.Contains(page, `</a><span class="sw-visually-hidden"> (`+overdue+`)</span></h`) {
		t.Errorf("expected heading links with apart text outside the <a>; page snippet:\n%s", truncate(page))
	}

	// Single-title tasks (like "Pay water bill") have no apart text and are fine.
	if !strings.Contains(page, ">Pay water bill</a>") {
		t.Errorf("single-title task link should be present\n%s", truncate(page))
	}
}

// Date metadata (e.g., "Today", "Yesterday at 1:12am") appears as separate text
// outside the heading/link element, within each row's content area. This covers
// acceptance item 2 of task 0690.
func TestTaskListDateMetadataIsOutsideTheHeadingLink(t *testing.T) {
	t.Parallel()
	_, h := newAppAt(t, tonight)
	overdue, later := seedTwins(t, h)

	page := get(t, h, "/t/task").Body.String()

	// The visually hidden span with the told text must appear AFTER the closing </a>.
	if !strings.Contains(page, `</a><span class="sw-visually-hidden"> (`+overdue+`)</span></h`) {
		t.Errorf("date metadata should be outside the <a> tag in a separate visually-hidden span; page snippet:\n%s", truncate(page))
	}

	// The <a> must have an aria-label that includes both title and told text,
	// providing the unique accessible name (WCAG 2.4.9).
	if !strings.Contains(page, `aria-label="Call plumber (`+overdue+`)"`) {
		t.Errorf("the link should have an aria-label with the full name including apart text for WCAG 2.4.9; page snippet:\n%s", truncate(page))
	}

	if !strings.Contains(page, `aria-label="Call plumber (`+later+`)"`) {
		t.Errorf("the second twin link should also have an aria-label with its apart text\n%s", truncate(page))
	}
}
