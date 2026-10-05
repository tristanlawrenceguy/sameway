package look

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// A browser that never opens its port is tried twice, and the error says
// what it said; one that is not there at all is not tried again.
func TestABrowserThatDoesNotStartIsTriedAgainAndSaysWhy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in browser is a shell script")
	}
	dir := t.TempDir()
	count := filepath.Join(dir, "count")
	fake := filepath.Join(dir, "browser")
	os.WriteFile(fake, []byte("#!/bin/sh\necho started >> "+count+"\necho 'cannot open display' >&2\nsleep 60\n"), 0o755)
	defer func(w time.Duration) { portWait = w }(portWait)
	portWait = time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := launch(ctx, fake)
	var slow errNoPort
	if !errors.As(err, &slow) || !strings.Contains(err.Error(), "cannot open display") {
		t.Fatalf("the error says what the browser said, got %v", err)
	}
	if raw, _ := os.ReadFile(count); strings.Count(string(raw), "started") != 2 {
		t.Errorf("a browser that did not open its port is started twice, got %q", raw)
	}
	if _, err := launch(ctx, filepath.Join(dir, "nothing-here")); errors.As(err, &slow) {
		t.Errorf("a browser that is not there is not waited for, got %v", err)
	}
}
