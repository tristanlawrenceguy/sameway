package server_test

import (
	"os"
	"strings"
	"testing"
)

// Voice mode keeps one turn button in place, so the focus is never lost;
// Hearing you is shown and not said, so a screen reader is not recorded
// over the person; and a hint says headphones help.
func TestVoiceModeTurnButtonAndHearing(t *testing.T) {
	t.Parallel()
	tpl, _ := os.ReadFile("../../design/components/talk/template.html")
	s := string(tpl)
	if strings.Count(s, "<button") != 2 || !strings.Contains(s, `sw-talk__turn" hidden>Done speaking</button>`) {
		t.Error("a start button and one turn button, nothing that appears and vanishes under the focus")
	}
	if !strings.Contains(s, `<span class="sw-talk__heard" aria-hidden="true">Hearing you.</span>`) {
		t.Error("Hearing you is shown beside the status, not inside it")
	}
	if !strings.Contains(s, "Headphones help") {
		t.Error("the hint about headphones is shown")
	}
	js, _ := os.ReadFile("../../design/components/talk/enhance.js")
	j := string(js)
	if strings.Contains(j, `"Hearing you."`) {
		t.Error("Hearing you is never put in the status, where a screen reader would say it while the person talks")
	}
	if !strings.Contains(j, `aloud.id = base + "-" + n;`) || !strings.Contains(j, "lab.htmlFor = aloud.id") {
		t.Error("a second voice mode on a page gives its checkbox its own id, so its label still presses it (a 44px target)")
	}
	if !strings.Contains(j, `if (document.activeElement === turn) toggle.focus();`) {
		t.Error("when the turn button goes, the focus goes to the start button")
	}
}

// When replies are not read aloud, voice mode lets the microphone go and
// waits for Speak; pausing lets it go too and says so; what was heard is
// said back as it is sent.
func TestVoiceModeLetsTheMicrophoneGo(t *testing.T) {
	t.Parallel()
	js, _ := os.ReadFile("../../design/components/talk/enhance.js")
	j := string(js)
	if !strings.Contains(j, "if (!aloud.checked || !voice || !text) { wait(); return; }") {
		t.Error("a reply not read aloud leads to waiting, not listening")
	}
	wait := j[strings.Index(j, "function wait()"):]
	if !strings.HasPrefix(strings.TrimSpace(wait[strings.Index(wait, "{")+1:]), "release();") {
		t.Error("waiting lets the microphone go")
	}
	pause := j[strings.Index(j, "function pause("):strings.Index(j, "function end(")]
	if !strings.Contains(pause, "release();") || !strings.Contains(pause, "The microphone is off.") {
		t.Error("pausing lets the microphone go and says so")
	}
	if !strings.Contains(j, `state("answering", "Sent: “"`) {
		t.Error("what was heard is said back as it is sent")
	}
	if strings.Contains(j, "The assistant is answering") {
		t.Error("the chat's own status says the assistant is working; voice mode does not say it again")
	}
}
