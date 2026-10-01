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
	return namedInline(out)
}

// namedInline reads who spoke from "Name: words", as Zoom writes its
// transcripts, when the file has no voice tags and most of its cues begin
// that way: a cue that merely contains a colon is not taken for a name.
func namedInline(cues []Cue) []Cue {
	named := 0
	for _, c := range cues {
		if c.Speaker != "" {
			return cues
		}
		if inlineName.MatchString(c.Text) {
			named++
		}
	}
	if named < 2 || named*2 < len(cues) {
		return cues
	}
	for i, c := range cues {
		if m := inlineName.FindStringSubmatch(c.Text); m != nil {
			cues[i].Speaker, cues[i].Text = strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		}
	}
	return cues
}

var inlineName = regexp.MustCompile(`^([^:\[\]]{1,40}?):\s+(.+)$`)

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

// Transcript is the cues as the record's text: a paragraph each, which
// begins with when and who, so search finds the words, the assistant reads
// who said them, and either of them can correct a word and the recording's
// page shows the correction at its time (FromTranscript).
func Transcript(cues []Cue) string {
	parts := make([]string, 0, len(cues))
	for _, c := range cues {
		p := "[" + Clock(c.Start) + "] "
		if c.Speaker != "" {
			p += "**" + c.Speaker + ":** "
		}
		parts = append(parts, p+c.Text)
	}
	return strings.Join(parts, "\n\n")
}

var transcriptLine = regexp.MustCompile(`(?s)^\[((?:\d+:)?\d{1,2}:\d{2})\]\s+(?:\*\*([^*]+?):\*\*\s*)?(.*)$`)

// FromTranscript reads a transcript back from text written as Transcript
// writes it, however it has been edited since: a paragraph that begins
// with its time is a line, and one that does not belongs to the line
// before it. Text with no times in it is not a transcript.
func FromTranscript(text string) []Cue {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out []Cue
	for _, para := range strings.Split(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if m := transcriptLine.FindStringSubmatch(para); m != nil {
			out = append(out, Cue{Start: seconds(m[1] + ".0"), Speaker: strings.TrimSpace(m[2]), Text: strings.Join(strings.Fields(m[3]), " ")})
			continue
		}
		if len(out) > 0 {
			out[len(out)-1].Text += " " + strings.Join(strings.Fields(para), " ")
		}
	}
	return out
}

// SRT writes cues as SubRip subtitles, each until the next begins when it
// has no end of its own, the last for a few seconds.
func SRT(cues []Cue) string {
	var b strings.Builder
	for i, c := range cues {
		end := c.End
		if end <= c.Start {
			end = c.Start + 5
			if i+1 < len(cues) && cues[i+1].Start > c.Start {
				end = cues[i+1].Start
			}
		}
		text := c.Text
		if c.Speaker != "" {
			text = c.Speaker + ": " + text
		}
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1, srtTime(c.Start), srtTime(end), text)
	}
	return b.String()
}

func srtTime(s float64) string {
	ms := int(s*1000 + 0.5)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}
