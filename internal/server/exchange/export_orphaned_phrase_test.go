package exchange_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Download links on record pages do not have an orphaned visually-hidden
// phrase after the format label. The fix moves $.what from a hidden span
// into visible text with a dash separator (e.g., " — these 2 tasks").
func TestDownloadLinksHaveNoOrphanedVisuallyHiddenPhrase(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	_, _ = a.Store.Create("task", map[string]any{"title": "Dig the pond"})

	tasks, _ := a.Store.List("task", store.ListOptions{})
	page := get(t, h, "/t/task/"+tasks[0].ID).Body.String()

	// The export component must not put $.what in a visually-hidden span
	// after a download link's format label. That was the bug: screen readers
	// would hear "Spreadsheet (CSV) , these 2 tasks" as two separate phrases.
	if strings.Contains(page, `<span class="sw-visually-hidden">, this task</span>`) {
		t.Errorf("download links still have an orphaned visually-hidden phrase:\n%s", truncate(page))
	}

	// The fix renders the context visible with a dash separator instead.
	// At least one link should contain " — this task" (visible after format).
	if !strings.Contains(page, " — this task") {
		t.Errorf("download links should show the context visibly with a dash: ' — this task'; got:\n%s", truncate(page))
	}

	// The export heading must still say what is being downloaded.
	if !strings.Contains(page, `Download this task`) {
		t.Error("the export component must say 'Download this task' once")
	}
}

// Download links on list pages similarly have no orphaned visually-hidden
// phrase. A list page (e.g., /t/task) shows export links for the whole list,
// and each link must end with a visible context phrase separated by a dash.
func TestListPageDownloadLinksHaveNoOrphanedVisuallyHiddenPhrase(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	_, _ = a.Store.Create("task", map[string]any{"title": "Task one"})
	_, _ = a.Store.Create("task", map[string]any{"title": "Task two"})

	page := get(t, h, "/t/task").Body.String()

	if strings.Contains(page, `<span class="sw-visually-hidden">, these 2 tasks</span>`) {
		t.Errorf("list page export links must not have orphaned visually-hidden phrases after format labels:\n%s", truncate(page))
	}
}

// Download links on a person's detail page also avoid orphaned phrases,
// covering another record type beyond task (acceptance item 1).
func TestPersonRecordDownloadLinksHaveNoOrphanedVisuallyHiddenPhrase(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	hana, _ := a.Store.Create("person", map[string]any{"name": "Hana Sato"})

	page := get(t, h, "/t/person/"+hana.ID).Body.String()

	if strings.Contains(page, `<span class="sw-visually-hidden">, this person</span>`) {
		t.Errorf("download links on person pages must not have orphaned visually-hidden phrase:\n%s", truncate(page))
	}

	if !strings.Contains(page, " — this person") {
		t.Errorf("person page download links should show visible context with dash: ' — this person'; got:\n%s", truncate(page))
	}
}

// The workspace export (everything) also avoids orphaned phrases on its page.
func TestEverythingExportHasNoOrphanedVisuallyHiddenPhrase(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	page := get(t, h, "/t/task").Body.String()

	if strings.Contains(page, `<span class="sw-visually-hidden">, everything in Home</span>`) {
		t.Errorf("workspace export links must not have orphaned visually-hidden phrases:\n%s", truncate(page))
	}
}
