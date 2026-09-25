package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// The shape of a chart, computed here because a template cannot do the
// arithmetic and a picture of numbers must not need JavaScript. The
// template only places what this returns; the numbers themselves are
// also a table under the picture, which is what a screen reader gets.

// ChartPoint is one value on a chart, with where it is drawn.
type ChartPoint struct {
	Label string
	Value float64
	Text  string
	// Bar geometry, and the point for a line.
	X, Y, W, H float64
	CX, CY     float64
	// LabelY is where the value is written above a bar.
	LabelY float64
	// ShowValue and ShowLabel thin a dense chart: past fourteen points the
	// values are left to the table, and only every few labels is written,
	// the last always among them.
	ShowValue, ShowLabel bool
}

// ChartTick is one gridline with its value.
type ChartTick struct {
	Y    float64
	Text string
}

// ChartShape is a whole chart laid out in a 600 by 260 box.
type ChartShape struct {
	Kind          string
	Width, Height float64
	Points        []ChartPoint
	Ticks         []ChartTick
	// Path is the line through the points, for a line chart.
	Path string
	// Baseline is the y of zero; Left is where the plot starts.
	Baseline, Left, Right float64
	Empty                 bool
	// Target is a line across the chart at a value to reach, with
	// HasTarget saying there is one; the scale makes room for it.
	Target    float64
	TargetY   float64
	HasTarget bool
}

const (
	chartW, chartH                            = 600.0, 260.0
	chartLeft, chartRight, chartTop, chartBot = 56.0, 16.0, 28.0, 44.0
)

// chartShape lays out a series of {label, value} as bars or a line.
func chartShape(series any, kind string, target ...any) ChartShape {
	points := chartPoints(series)
	s := ChartShape{Kind: kind, Width: chartW, Height: chartH, Points: points, Left: chartLeft, Right: chartW - chartRight}
	if len(target) > 0 {
		if t := numberOf(target[0]); t > 0 {
			s.Target, s.HasTarget = t, true
		}
	}
	if kind != "line" {
		s.Kind = "bar"
	}
	if len(points) == 0 {
		s.Empty = true
		return s
	}
	top := 0.0
	for _, p := range points {
		if p.Value > top {
			top = p.Value
		}
	}
	if s.HasTarget && s.Target > top {
		top = s.Target
	}
	if top <= 0 {
		top = 1
	}
	step := niceStep(top / 4)
	// A count of things is never one and a half; whole values get whole
	// gridlines, fewer of them if need be.
	if step < 1 && wholeValues(points) {
		step = 1
	}
	top = math.Ceil(top/step) * step
	plotW := chartW - chartLeft - chartRight
	plotH := chartH - chartTop - chartBot
	s.Baseline = chartTop + plotH
	for v := 0.0; v <= top+step/1000; v += step {
		s.Ticks = append(s.Ticks, ChartTick{Y: round(s.Baseline - v/top*plotH), Text: numberText(v)})
	}
	if s.HasTarget {
		s.TargetY = round(s.Baseline - s.Target/top*plotH)
	}
	slot := plotW / float64(len(points))
	dense := len(points) > 14
	every := (len(points) + 7) / 8
	if !dense {
		// Labels wider than their slot, such as twelve months with their
		// years, are thinned so they do not run into each other.
		widest := 0
		for _, p := range points {
			widest = max(widest, len([]rune(chartLabel(p.Label))))
		}
		every = max(1, int(math.Ceil(float64(widest)*6/(slot*0.9))))
	}
	var path []string
	for i := range s.Points {
		p := &s.Points[i]
		p.ShowValue = !dense
		p.ShowLabel = (len(points)-1-i)%every == 0
		h := p.Value / top * plotH
		if p.Value < 0 {
			h = 0
		}
		p.W = round(slot * 0.52)
		p.X = round(chartLeft + float64(i)*slot + slot*0.24)
		p.H = round(h)
		p.Y = round(s.Baseline - h)
		p.LabelY = p.Y - 6
		if p.LabelY < chartTop-8 {
			p.LabelY = chartTop - 8
		}
		p.CX = round(chartLeft + (float64(i)+0.5)*slot)
		p.CY = p.Y
		path = append(path, fmt.Sprintf("%g,%g", p.CX, p.CY))
	}
	s.Path = strings.Join(path, " ")
	return s
}

// sparkline is the line alone, for a glance or a brief size: the points of
// a polyline in a 240 by 48 box.
func sparkline(series any) string {
	points := chartPoints(series)
	if len(points) < 2 {
		return ""
	}
	top := 0.0
	for _, p := range points {
		if p.Value > top {
			top = p.Value
		}
	}
	if top <= 0 {
		top = 1
	}
	var out []string
	for i, p := range points {
		x := 4 + float64(i)*(232/float64(len(points)-1))
		y := 44 - p.Value/top*40
		out = append(out, fmt.Sprintf("%g,%g", round(x), round(y)))
	}
	return strings.Join(out, " ")
}

// chartLast is the last point of a series, and chartTrend says in a word
// how it compares with the one before: up, down, level, or nothing to
// compare with.
func chartLast(series any) ChartPoint {
	points := chartPoints(series)
	if len(points) == 0 {
		return ChartPoint{}
	}
	return points[len(points)-1]
}

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

// chartPoints reads the plain maps a component's props carry.
func chartPoints(series any) []ChartPoint {
	list, _ := series.([]any)
	var out []ChartPoint
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		label, _ := m["label"].(string)
		v := numberOf(m["value"])
		out = append(out, ChartPoint{Label: label, Value: v, Text: numberText(v)})
	}
	return out
}

func numberOf(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case float64:
		return x
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f
	}
	return 0
}

// numberText writes a value the way a person would: whole when it is
// whole, else to two places.
func numberText(v float64) string {
	if v == math.Trunc(v) {
		return strconv.FormatInt(int64(v), 10)
	}
	// 295.5, not 295.50: a trailing nought says nothing.
	return strings.TrimRight(strconv.FormatFloat(v, 'f', 2, 64), "0")
}

// niceStep is the distance between gridlines: 1, 2 or 5 times a power of
// ten, at least the size asked for, so the lines fall on numbers people
// read and the top of the axis is a whole number of them.
func niceStep(v float64) float64 {
	if v <= 0 {
		return 1
	}
	exp := math.Floor(math.Log10(v))
	base := math.Pow(10, exp)
	for _, m := range []float64{1, 2, 5, 10} {
		if v <= m*base {
			return m * base
		}
	}
	return 10 * base
}

func round(v float64) float64 { return math.Round(v*10) / 10 }

func wholeValues(points []ChartPoint) bool {
	for _, p := range points {
		if p.Value != math.Trunc(p.Value) {
			return false
		}
	}
	return true
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
