package convert

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// A recording's words, as a WebVTT file holds them: what was said, from
// when to when, and by whom when a voice tag names them. One file is the
// source for all of it: the transcript on the page, its links into the
// recording, the line that is being heard, and the text search and the
// assistant read.

// Cue is one stretch of what was said.
type Cue struct {
	Start, End float64 // seconds
	Speaker    string  // from <v Name>, when the file says
	Text       string  // the words, and sounds such as [laughter]
}

var (
	vttTime  = regexp.MustCompile(`^((?:\d+:)?\d{1,2}:\d{2}[.,]\d{1,3})\s+-->\s+((?:\d+:)?\d{1,2}:\d{2}[.,]\d{1,3})`)
	vttVoice = regexp.MustCompile(`^<v(?:\.[^ >]*)?\s+([^>]+)>`)
	vttTag   = regexp.MustCompile(`</?[a-zA-Z][^>]*>|<\d[^>]*>`)
)

// ParseVTT reads the cues of a WebVTT (or SRT) file, in order. Lines it
// does not understand, such as notes and styles, are passed over.
func ParseVTT(text string) []Cue {
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	var out []Cue
	for _, block := range strings.Split(text, "\n\n") {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		for i, line := range lines {
			m := vttTime.FindStringSubmatch(strings.TrimSpace(line))
			if m == nil {
				continue
			}
			c := Cue{Start: seconds(m[1]), End: seconds(m[2])}
			words := strings.TrimSpace(strings.Join(lines[i+1:], " "))
			if v := vttVoice.FindStringSubmatch(words); v != nil {
				c.Speaker = strings.TrimSpace(v[1])
			}
			c.Text = strings.Join(strings.Fields(vttTag.ReplaceAllString(words, "")), " ")
			if c.Text != "" {
				out = append(out, c)
			}
			break
		}
	}
	return out
}

// seconds reads 01:02:03.500, 02:03.500 or 00:02:03,500.
func seconds(t string) float64 {
	t = strings.Replace(t, ",", ".", 1)
	parts := strings.Split(t, ":")
	total := 0.0
	for _, p := range parts {
		n, _ := strconv.ParseFloat(p, 64)
		total = total*60 + n
	}
	return total
}

// Clock is a time in a recording as it is shown: 4:03, or 1:02:03.
func Clock(s float64) string {
	n := int(s)
	if n >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", n/3600, n/60%60, n%60)
	}
	return fmt.Sprintf("%d:%02d", n/60, n%60)
}

// Spoken is a time in words, as a screen reader should say it: 4 minutes
// 3 seconds, not 4:03.
func Spoken(s float64) string {
	n := int(s)
	h, m, sec := n/3600, n/60%60, n%60
	var parts []string
	unit := func(v int, one string) {
		if v == 1 {
			parts = append(parts, "1 "+one)
		} else if v > 1 {
			parts = append(parts, fmt.Sprintf("%d %ss", v, one))
		}
	}
	unit(h, "hour")
	unit(m, "minute")
	unit(sec, "second")
	if len(parts) == 0 {
		return "0 seconds"
	}
	return strings.Join(parts, " ")
}

// Transcript is the cues as the record's text: each speaker's turn a
// paragraph that begins with when and who, so search finds the words and
// the assistant reads who said them.
func Transcript(cues []Cue) string {
	var b strings.Builder
	last := ""
	for i, c := range cues {
		if i == 0 || c.Speaker != last {
			if i > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString("[" + Clock(c.Start) + "] ")
			if c.Speaker != "" {
				b.WriteString("**" + c.Speaker + ":** ")
			}
		} else {
			b.WriteString(" ")
		}
		b.WriteString(c.Text)
		last = c.Speaker
	}
	return b.String()
}
