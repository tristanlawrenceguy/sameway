package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
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

// ChartShape is a whole chart laid out in a box 260 high: 600 wide, or 360
// for a narrow place, where the same words are drawn larger for the room.
type ChartShape struct {
	Kind          string
	Width, Height float64
	Points        []ChartPoint
	Ticks         []ChartTick
	// Path is the line through the points, for a line chart.
	Path string
	// Baseline is the y of zero, Bottom the foot of the plot, where the
	// labels go (below zero when there are values under it); Left is where
	// the plot starts.
	Baseline, Bottom, Left, Right float64
	Empty                         bool
	// Target is a line across the chart at a value to reach, with
	// HasTarget saying there is one; the scale makes room for it.
	Target    float64
	TargetY   float64
	HasTarget bool
}

const (
	chartW, chartNarrowW, chartH   = 600.0, 360.0, 260.0
	chartRight, chartTop, chartBot = 16.0, 28.0, 44.0
)

// chartShape lays out a series of {label, value} as bars or a line, wide.
func chartShape(series any, kind string, target ...any) ChartShape {
	return chartShapeAt(chartW, series, kind, target...)
}

// chartNarrow is the same chart for a narrow place: fewer labels, values
// written only when there are few, each drawn large enough to read.
func chartNarrow(series any, kind string, target ...any) ChartShape {
	return chartShapeAt(chartNarrowW, series, kind, target...)
}

func chartShapeAt(width float64, series any, kind string, target ...any) ChartShape {
	narrow := width < chartW
	left, charW, maxValues := 56.0, 6.0, 14
	if narrow {
		left, charW, maxValues = 40.0, 7.5, 7
	}
	points := chartPoints(series)
	s := ChartShape{Kind: kind, Width: width, Height: chartH, Points: points, Left: left, Right: width - chartRight}
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
	// The scale runs from zero, or from the lowest value when it is below
	// zero, to the highest or the target: a bar is always measured from
	// zero, down for a value under it.
	top, low := 0.0, 0.0
	for _, p := range points {
		top, low = math.Max(top, p.Value), math.Min(low, p.Value)
	}
	if s.HasTarget && s.Target > top {
		top = s.Target
	}
	if top <= 0 && low == 0 {
		top = 1
	}
	step := niceStep((top - low) / 4)
	// A count of things is never one and a half; whole values get whole
	// gridlines, fewer of them if need be.
	if step < 1 && wholeValues(points) {
		step = 1
	}
	top = math.Ceil(top/step) * step
	low = math.Floor(low/step) * step
	span := top - low
	plotW := width - left - chartRight
	plotH := chartH - chartTop - chartBot
	s.Bottom = chartTop + plotH
	at := func(v float64) float64 { return round(s.Bottom - (v-low)/span*plotH) }
	s.Baseline = at(0)
	// The zero line first, so it is the one drawn solid.
	s.Ticks = append(s.Ticks, ChartTick{Y: s.Baseline, Text: "0"})
	for v := low; v <= top+step/1000; v += step {
		if math.Abs(v) > step/1000 {
			s.Ticks = append(s.Ticks, ChartTick{Y: at(v), Text: numberText(v)})
		}
	}
	if s.HasTarget {
		s.TargetY = at(s.Target)
	}
	slot := plotW / float64(len(points))
	dense := len(points) > maxValues
	// Labels wider than their slot, such as twelve months with their years,
	// are thinned so they do not run into each other; the last always shows.
	widest := 0
	for _, p := range points {
		widest = max(widest, len([]rune(chartLabel(p.Label))))
	}
	every := max(1, int(math.Ceil(float64(widest)*charW/(slot*0.9))))
	var path []string
	for i := range s.Points {
		p := &s.Points[i]
		p.ShowValue = !dense
		p.ShowLabel = (len(points)-1-i)%every == 0
		y := at(p.Value)
		p.W = round(slot * 0.52)
		p.X = round(left + float64(i)*slot + slot*0.24)
		p.Y, p.H = math.Min(y, s.Baseline), round(math.Abs(s.Baseline-y))
		// A value is written above its bar, or below one that goes down.
		p.LabelY = p.Y - 6
		if p.Value < 0 {
			p.LabelY = p.Y + p.H + 14
		}
		if p.LabelY < chartTop-8 {
			p.LabelY = chartTop - 8
		}
		p.CX = round(left + (float64(i)+0.5)*slot)
		p.CY = y
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
	top, low := 0.0, 0.0
	for _, p := range points {
		top, low = math.Max(top, p.Value), math.Min(low, p.Value)
	}
	if top-low <= 0 {
		top = 1
	}
	var out []string
	for i, p := range points {
		x := 4 + float64(i)*(232/float64(len(points)-1))
		y := 44 - (p.Value-low)/(top-low)*40
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
