package server_test

import (
	"os"
	"strings"
	"testing"
)

// TestChatLogHasSmoothScroll verifies acceptance item 1 of task 0303: the chat
// log stylesheet includes scroll-behavior: smooth so native anchor navigation
// works smoothly within the scroll container. Without this, clicking "Skip to
// latest message" may jump or fail entirely on long conversations.
func TestChatLogHasSmoothScroll(t *testing.T) {
	data, err := os.ReadFile("../../design/components/chat/style.css")
	if err != nil {
		t.Fatalf("read chat style.css: %v", err)
	}

	src := string(data)
	if !strings.Contains(src, "scroll-behavior") {
		t.Errorf(".sw-chat__log CSS must include scroll-behavior for smooth anchor navigation;\nthis is needed so clicking 'Skip to latest message' scrolls smoothly\nwithin the chat log container.\n\nCurrent style.css:\n%s", src)
	}

	// The rule should be scoped to .sw-chat__log, not a bare property.
	if !strings.Contains(src, ".sw-chat__log") {
		t.Errorf("stylesheet missing .sw-chat__log selector; scroll-behavior must apply to the chat log container")
	}
}
