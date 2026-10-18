package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Habit log form fields are labeled: the unit name is a visible label for the
// amount field and "Day" is a visible label for the date field on dated habits;
// labels come before their inputs so screen readers encounter them first. On
// non-dated habits there is no bare unit text after the input. This covers
// acceptance items 1–4 of fixing unlabeled raw text in habit log forms.

func TestDatedHabitLogFieldsHaveLabelsBeforeInputs(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	_, err := a.Store.Create(server.HabitType, map[string]any{
		"name": "Running miles", "cadence": "day", "target": 5,
		"unit": "miles", "aim": "record",
	})
	if err != nil {
		t.Fatal(err)
	}

	// We need the actual habit IDs from the store.
	habits, err := a.Store.List(server.HabitType, store.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(habits) == 0 {
		t.Fatal("no habits found")
	}

	for _, hab := range habits {
		page := get(t, h, "/t/"+server.HabitType+"/"+hab.ID).Body.String()
		unitLabel := `<label class="sw-tracker__unit" for="log-` + hab.ID + `">miles</label>`
		if !strings.Contains(page, unitLabel) {
			t.Errorf("dated habit log form has a label for the amount field: missing %s\n%s", unitLabel, page)
		}

		// The label must come before the input.
		labelIdx := strings.Index(page, unitLabel)
		inputIdx := strings.Index(page, `id="log-`+hab.ID+`"`)
		if labelIdx < 0 || inputIdx < 0 {
			continue // already caught by missing label above
		}
		if labelIdx > inputIdx {
			t.Errorf("the unit label must come before the amount input, but it appears after\n%s", page)
		}

		// Day label also comes before its input.
		dayLabel := `<label class="sw-tracker__unit" for="on-` + hab.ID + `">Day</label>`
		if !strings.Contains(page, dayLabel) {
			t.Errorf("dated habit log form has a 'Day' label: missing %s\n%s", dayLabel, page)
		}
		dayLabelIdx := strings.Index(page, dayLabel)
		dayInputIdx := strings.Index(page, `id="on-`+hab.ID+`"`)
		if dayLabelIdx < 0 || dayInputIdx < 0 {
			continue
		}
		if dayLabelIdx > dayInputIdx {
			t.Errorf("the Day label must come before the date input\n%s", page)
		}

		// No bare concatenated text like "miles Day" between inputs.
		// In the log form, find the portion between the amount input and day input.
		formStart := strings.Index(page, `action="/habit/`+hab.ID+`/log"`)
		if formStart < 0 {
			continue
		}
		formEnd := strings.Index(page[formStart:], `<button type="submit"`)
		if formEnd < 0 {
			continue
		}
		formHTML := page[formStart : formStart+formEnd]
		// Strip out the actual label and input tags to see if bare text remains.
		bareText := strings.ReplaceAll(formHTML, unitLabel, "")
		bareText = strings.ReplaceAll(bareText, dayLabel, "")
		bareText = strings.ReplaceAll(bareText, `<input`, "<INPUT")
		if strings.Contains(strings.ToLower(bareText), "miles day") {
			t.Errorf("dated habit log form has bare concatenated text between inputs: %q\n%s", bareText, page)
		}
	}
}

func TestNonDatedHabitLogFieldHasLabelBeforeInput(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	_, err := a.Store.Create(server.HabitType, map[string]any{
		"name": "Meditate", "cadence": "day", "target": 10,
		"unit": "minutes",
	})
	if err != nil {
		t.Fatal(err)
	}

	habits, _ := a.Store.List(server.HabitType, store.ListOptions{})
	if len(habits) == 0 {
		t.Fatal("no habits found")
	}

	for _, hab := range habits {
		page := get(t, h, "/t/"+server.HabitType+"/"+hab.ID).Body.String()
		unitLabel := `<label class="sw-tracker__unit" for="log-` + hab.ID + `">minutes</label>`
		if !strings.Contains(page, unitLabel) {
			t.Errorf("non-dated habit log form has a label for the amount field: missing %s\n%s", unitLabel, page)
		}

		labelIdx := strings.Index(page, unitLabel)
		inputIdx := strings.Index(page, `id="log-`+hab.ID+`"`)
		if labelIdx < 0 || inputIdx < 0 {
			continue
		}
		if labelIdx > inputIdx {
			t.Errorf("the unit label must come before the amount input\n%s", page)
		}

		// On non-dated habits, there should be no "Day" label or date input.
		if strings.Contains(page, `name="on"`+` type="text"`) {
			t.Errorf("non-dated habit log form should not have a Day/date field\n%s", page)
		}

		// No bare unit text after the input in the form.
		formStart := strings.Index(page, `action="/habit/`+hab.ID+`/log"`)
		if formStart < 0 {
			continue
		}
		formEnd := strings.Index(page[formStart:], `<button type="submit"`)
		if formEnd < 0 {
			continue
		}
		formHTML := page[formStart : formStart+formEnd]
		bareText := strings.ReplaceAll(formHTML, unitLabel, "")
		bareText = strings.ReplaceAll(bareText, `<input`, "<INPUT")
		if strings.Contains(strings.ToLower(bareText), "minutes") {
			t.Errorf("non-dated habit log form has bare unit text after input: %q\n%s", bareText, page)
		}
	}
}
