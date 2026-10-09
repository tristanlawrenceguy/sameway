package server_test

import (
	"os"
	"strings"
	"testing"
)

// The voice button's script says why the microphone could not start (not
// allowed, none found, in use elsewhere) with what still works; stops
// dictation at a limit it shows, keeping what was said; keeps a recording
// when the microphone goes away; is unavailable, not silent, while words
// are written down; and passes on no program's own error text.
func TestVoiceSaysWhatHappenedInWords(t *testing.T) {
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	js := read("../../design/components/voice/enhance.js")
	for _, want := range []struct{ text, why string }{
		{`"NotAllowedError"`, "a refused microphone is told apart"},
		{`"NotFoundError"`, "no microphone is told apart"},
		{`"NotReadableError"`, "a microphone in use elsewhere is told apart"},
		{"It can be allowed in the browser's settings for this site.", "a refusal says how to undo it"},
		{"No microphone was found", "no microphone is said in words"},
		{"Another program may be using it", "a busy microphone is said in words"},
		{" The message can still be typed.", "dictation's refusal says what still works"},
		{" A file can still be chosen.", "recording's refusal says what still works"},
		{"var LONGEST = 5 * 60", "dictation has a limit"},
		{`" of " + clock(LONGEST)`, "the limit is shown beside the time"},
		{"Listening, for up to 5 minutes.", "the limit is said when listening starts"},
		{"Dictation stops after 5 minutes; press Dictate to go on.", "reaching the limit keeps the words and says how to go on"},
		{"r.onend = function", "a microphone that stops by itself ends the recording"},
		{"The microphone stopped, so the recording ended.", "and says so"},
		{`btn.setAttribute("aria-disabled", "true")`, "the button is unavailable while words are written down"},
		{"setSelectionRange(at + put.length", "the cursor goes after dictated words"},
		{"e && e.plain ? e.message : failed)", "only plain words are passed on"},
	} {
		if !strings.Contains(js, want.text) {
			t.Errorf("%s: enhance.js lacks %q", want.why, want.text)
		}
	}
	if strings.Contains(js, "Press Stop recording to finish") || strings.Contains(js, "Press Stop dictating when") {
		t.Error("the start is said briefly: the microphone would catch a long announcement")
	}
	speech := read("../../design/base/28-speech.js")
	if !strings.Contains(speech, `addEventListener("ended"`) || !strings.Contains(speech, `rec.state === "inactive"`) {
		t.Error("28-speech.js notices a microphone that stops by itself and still gives what was heard")
	}
}
