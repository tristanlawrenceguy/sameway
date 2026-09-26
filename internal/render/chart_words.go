package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// A chart's words: its numbers as a person writes them, its labels, and
// what it shows said in a sentence, for whoever cannot see the picture.

func chartTrend(series any) string {
	points := chartPoints(series)
	if len(points) < 2 {
		return ""
	}
	last, prev := points[len(points)-1].Value, points[len(points)-2].Value
	switch {
	case last > prev:
		return "up from " + numberText(prev)
	case last < prev:
		return "down from " + numberText(prev)
	}
	return "the same as before"
}

// numberText writes a value the way a person would: whole when it is
// whole, else to two places.
func numberText(v float64) string {
	text := strconv.FormatFloat(v, 'f', 2, 64)
	if v == math.Trunc(v) {
		text = strconv.FormatInt(int64(v), 10)
	} else {
		// 295.5, not 295.50: a trailing nought says nothing.
		text = strings.TrimRight(text, "0")
	}
	return groupThousands(text)
}

// groupThousands writes 12500 as 12,500, which is read at a glance.
func groupThousands(text string) string {
	sign, whole, frac := "", text, ""
	if strings.HasPrefix(whole, "-") {
		sign, whole = "-", whole[1:]
	}
	if i := strings.IndexByte(whole, '.'); i >= 0 {
		whole, frac = whole[:i], whole[i:]
	}
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	return sign + whole + frac
}

// chartLabel is a group label as it is read under a bar: a day as its
// short date, a month as its name, anything else as it is.
func chartLabel(label string) string {
	if day, ok := strings.CutPrefix(label, "Week of "); ok {
		return "Week of " + chartLabel(day)
	}
	if t, err := time.Parse("2006-01-02", label); err == nil {
		return t.Format("2 Jan")
	}
	if t, err := time.Parse("2006-01", label); err == nil {
		return t.Format("Jan 2006")
	}
	return label
}

// chartSummary says what a chart shows in a sentence, for a chart given no
// description of its own: where it starts and ends, its highest and
// lowest, and how many reached the target or kept within the limit.
func chartSummary(series any, unit string, target any, targetLabel string) string {
	points := chartPoints(series)
	if len(points) == 0 {
		return ""
	}
	with := func(v float64) string {
		if unit == "" {
			return numberText(v)
		}
		return numberText(v) + " " + unit
	}
	first, last := points[0], points[len(points)-1]
	out := fmt.Sprintf("%s for %s", with(last.Value), chartLabel(last.Label))
	if len(points) > 1 {
		hi, lo := first, first
		for _, p := range points {
			if p.Value > hi.Value {
				hi = p
			}
			if p.Value < lo.Value {
				lo = p
			}
		}
		out = fmt.Sprintf("From %s for %s to %s for %s. Highest %s, for %s; lowest %s, for %s",
			with(first.Value), chartLabel(first.Label), with(last.Value), chartLabel(last.Label),
			with(hi.Value), chartLabel(hi.Label), with(lo.Value), chartLabel(lo.Label))
	}
	if t := numberOf(target); t > 0 {
		met := 0
		for _, p := range points {
			if targetLabel == "limit" && p.Value <= t || targetLabel != "limit" && p.Value >= t {
				met++
			}
		}
		if targetLabel == "limit" {
			out += fmt.Sprintf(". Within the limit of %s for %d of %d", with(t), met, len(points))
		} else {
			out += fmt.Sprintf(". Reached the target of %s for %d of %d", with(t), met, len(points))
		}
	}
	return out + "."
}
