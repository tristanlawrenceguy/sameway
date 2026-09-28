package server_test

import (
	"strings"
	"testing"
)

// importDescriptionText verifies that each content type's import page shows
// description copy appropriate for its kind. The person type should mention
// vCard and mailbox formats; all others must not — they accept CSV/TSV/TXT.
func TestImportPageDescriptionsAreTypeSpecific(t *testing.T) {
	_, h := newApp(t)

	// "person" is the only built-in format that legitimately supports vCard/mbox,
	// so its description should keep those references.
	personRec := get(t, h, "/t/person/import")
	if !strings.Contains(personRec.Body.String(), "vCard (.vcf)") {
		t.Errorf("/t/person/import: expected description to mention vCard (.vcf), got:\n%s", truncate(personRec.Body.String()))
	}
	if !strings.Contains(personRec.Body.String(), "mailbox (.mbox)") {
		t.Errorf("/t/person/import: expected description to mention mailbox (.mbox), got:\n%s", truncate(personRec.Body.String()))
	}

	// note, task, and action should NOT mention contact/mail formats.
	for _, typ := range []string{"note", "task", "action"} {
		rec := get(t, h, "/t/"+typ+"/import")
		body := rec.Body.String()

		// Extract just the description paragraph (<p class="sw-muted">...</p>).
		startIdx := strings.Index(body, `<p class="sw-muted">`)
		if startIdx == -1 {
			t.Errorf("/t/%s/import: expected a <p class=\"sw-muted\"> description paragraph\n%s", typ, truncate(body))
			continue
		}
		endIdx := strings.Index(body[startIdx:], "</p>")
		if endIdx == -1 {
			t.Errorf("/t/%s/import: could not find closing </p> of sw-muted paragraph\n%s", typ, truncate(body))
			continue
		}
		desc := body[startIdx : startIdx+endIdx]

		if strings.Contains(desc, "vCard (.vcf)") || strings.Contains(desc, ".vcf") {
			t.Errorf("/t/%s/import: description must not mention vCard or .vcf\n%s", typ, truncate(body))
		}
		if strings.Contains(desc, "mailbox (.mbox)") || strings.Contains(desc, ".mbox") {
			t.Errorf("/t/%s/import: description must not mention mailbox or .mbox\n%s", typ, truncate(body))
		}
		// The type-specific copy should reference the plural name.
		switch typ {
		case "note":
			if !strings.Contains(desc, "notes") {
				t.Errorf("/t/note/import: expected description to mention \"notes\"\n%s", truncate(body))
			}
		case "task":
			if !strings.Contains(desc, "tasks") {
				t.Errorf("/t/task/import: expected description to mention \"tasks\"\n%s", truncate(body))
			}
		case "action":
			if !strings.Contains(desc, "actions") {
				t.Errorf("/t/action/import: expected description to mention \"actions\"\n%s", truncate(body))
			}
		}
	}
}
