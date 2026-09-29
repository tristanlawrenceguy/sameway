package ingest

import (
	"errors"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
)

// ReadICS reads a calendar's events as rows: title, starts, ends, where,
// notes, repeat, and the calendar's own uid, which an import uses to add
// nothing twice.
func ReadICS(data []byte) (*Table, error) {
	events := convert.ParseICS(data, nil)
	if len(events) == 0 {
		return nil, errors.New("the calendar has no events in it")
	}
	tb := &Table{Source: "calendar", Columns: []string{"title", "starts", "ends", "where", "notes", "repeat", "uid"}}
	for _, e := range events {
		tb.Rows = append(tb.Rows, map[string]string{
			"title": e.Title, "starts": e.Starts, "ends": e.Ends, "where": e.Where,
			"notes": e.Notes, "repeat": e.Repeat, "uid": e.UID,
		})
	}
	return tb, nil
}
