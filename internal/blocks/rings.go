package blocks

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
)

// What a reminder rings about: the record its about names, and what the
// ring says of it.

// AboutOf reads a page path, /t/{type}/{id}, back to the record it names.
func AboutOf(st *store.Store, path string) (*schema.Type, *store.Record, bool) {
	path, _, _ = strings.Cut(path, "?") // a part to open, such as a meeting's recording
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(path), "/"), "/")
	if len(parts) != 3 || parts[0] != "t" {
		return nil, nil, false
	}
	t, ok := st.Types().Get(parts[1])
	if !ok {
		return nil, nil, false
	}
	rec, err := st.Get(t.Name, parts[2])
	if err != nil {
		return nil, nil, false
	}
	return t, rec, true
}

// RingWords is what a ring says beyond the page and where it leads: the
// thing it is about, and for a habit where it stands; the reminder
// itself when it is about nothing.
func RingWords(st *store.Store, rec *store.Record) (text, url string) {
	url = "/t/" + records.ReminderType + "/" + rec.ID
	text = "It is time."
	about, _ := rec.Fields["about"].(string)
	t, target, ok := AboutOf(st, about)
	if !ok {
		return text, url
	}
	url = about
	text = records.Name(st, t, target)
	// A meeting's reminder to record says what to do, in its own notes.
	if notes, _ := rec.Fields["notes"].(string); t.Name == records.EventType && strings.Contains(about, "show=recording") && notes != "" {
		text = notes
	}
	if t.Name == records.HabitType {
		h := HabitOf(target)
		sum := track.Summarise(h, EntriesOf(st, h.ID), time.Now(), 1)
		text = h.Name + ": " + track.Progress(h, sum) + " so far"
	}
	return text, url
}
