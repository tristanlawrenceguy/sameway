package convert

import (
	"strings"
	"testing"
)

// A recording's WebVTT is read into who said what and when, whatever
// notes and tags it carries, and becomes text search and the assistant read.
func TestAWebVTTBecomesWhoSaidWhatAndWhen(t *testing.T) {
	cues := ParseVTT("WEBVTT\r\n\r\nNOTE made by whisper\r\n\r\n1\r\n00:00:00.000 --> 00:00:04.200\r\n<v Hana>Morning. <b>Two</b> things from me.\r\n\r\n00:04.200 --> 00:11,800\r\n<v.loud Hana>The compost order went in [laughter]\r\nat last.\r\n\r\n01:02:03.500 --> 01:02:09.000\r\n<v Sam>The liner comes Thursday.</v>\r\n")
	if len(cues) != 3 {
		t.Fatalf("three cues, got %d: %+v", len(cues), cues)
	}
	if cues[0].Speaker != "Hana" || cues[0].Text != "Morning. Two things from me." || cues[1].Text != "The compost order went in [laughter] at last." || cues[2].Start != 3723.5 || cues[2].Speaker != "Sam" {
		t.Errorf("cues read wrong: %+v", cues)
	}
	if Clock(3723.5) != "1:02:03" || Clock(4.2) != "0:04" || Spoken(3723.5) != "1 hour 2 minutes 3 seconds" || Spoken(0) != "0 seconds" {
		t.Error("times are shown and said as people say them")
	}
	text := Transcript(cues)
	if !strings.HasPrefix(text, "[0:00] **Hana:** Morning. Two things from me.\n\n[0:04] **Hana:** The compost order") || !strings.Contains(text, "\n\n[1:02:03] **Sam:** The liner") {
		t.Errorf("each line is a paragraph with when and who: %q", text)
	}
	// Corrected by hand, the text is still the transcript, at its times.
	back := FromTranscript(strings.Replace(text, "compost order", "compost delivery", 1) + "\n\nAnd one more thing.")
	if len(back) != 3 || back[1].Start != 4 || back[1].Speaker != "Hana" || back[1].Text != "The compost delivery went in [laughter] at last." || back[2].Start != 3723 || !strings.HasSuffix(back[2].Text, "Thursday. And one more thing.") {
		t.Errorf("the text reads back as the lines, corrections and all: %+v", back)
	}
	if FromTranscript("Just some notes about the meeting.") != nil {
		t.Error("text with no times is not a transcript")
	}
	if Kind("memo.m4a") != "audio" || AudioType("x.opus") != "audio/ogg" || !Builtin("a.mp3") {
		t.Error("recordings are known by their extension")
	}
}
