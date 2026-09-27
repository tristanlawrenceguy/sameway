package when

import "time"

// Occurrences is every time a repeat falls after the one in v on the days
// from and to, both YYYY-MM-DD, as it is stored: the times a calendar
// shows for a month. At most most of them, so a month shown is always a
// bounded list, never the whole repeat.
func Occurrences(repeat, v, from, to string, most int) []string {
	r, ok := stored(repeat)
	if !ok || v == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, v); err != nil {
		return nil
	}
	base, loc, day := start(v, time.Now())
	var out []string
	r.each(base, loc, func(t time.Time) bool {
		d := t.Format("2006-01-02")
		if d > to || len(out) == most {
			return false
		}
		if t.After(base) && d >= from {
			out = append(out, Store(t, day))
		}
		return true
	})
	return out
}
