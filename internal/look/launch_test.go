package look

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMain lets this test binary stand in for a browser that never opens
// its DevTools port: started with LOOK_STAND_IN set, it notes that it was
// started, says why on stderr as a browser does, and waits to be killed,
// whatever flags a browser is given. It runs on every system, unlike a
// shell script.
func TestMain(m *testing.M) {
	if os.Getenv("LOOK_HOLD") != "" { // a helper of the stand-in, holding its stderr
		time.Sleep(8 * time.Second) // longer than WaitDelay, shorter than a test run
		os.Exit(0)
	}
	if count := os.Getenv("LOOK_STAND_IN"); count != "" {
		f, _ := os.OpenFile(count, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		fmt.Fprintln(f, "started")
		f.Close()
		fmt.Fprintln(os.Stderr, "cannot open display")
		// As Chrome's helpers do, one outlives it with its stderr open.
		helper := exec.Command(os.Args[0])
		helper.Env = append(os.Environ(), "LOOK_STAND_IN=", "LOOK_HOLD=1")
		helper.Stderr = os.Stderr
		helper.Start()
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// A browser that never opens its port is tried twice, well within a
// look's minute, and the error says what it said; one that is not there
// at all is not tried again.
func TestABrowserThatDoesNotStartIsTriedAgainAndSaysWhy(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	count := filepath.Join(t.TempDir(), "count")
	t.Setenv("LOOK_STAND_IN", count)
	defer func(w time.Duration) { portWait = w }(portWait)
	portWait = time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	began := time.Now()
	_, err = launch(ctx, self)
	// The stand-in's helper outlives it; on Windows a running program
	// cannot be deleted, so the test waits for it to go.
	t.Cleanup(func() { time.Sleep(8 * time.Second) })
	var slow errNoPort
	if !errors.As(err, &slow) || !strings.Contains(err.Error(), "cannot open display") {
		t.Fatalf("the error says what the browser said, got %v", err)
	}
	if raw, _ := os.ReadFile(count); strings.Count(string(raw), "started") != 2 {
		t.Errorf("a browser that did not open its port is started twice, got %q", raw)
	}
	// Two waits of a second, and a stand-in that holds its stderr open is
	// let go of within WaitDelay each time, not when it exits a minute on.
	if took := time.Since(began); took > 12*time.Second {
		t.Errorf("both tries took %s; a browser that holds stderr open must not hold the look up", took)
	}
	if _, err := launch(ctx, filepath.Join(t.TempDir(), "nothing-here")); errors.As(err, &slow) {
		t.Errorf("a browser that is not there is not waited for, got %v", err)
	}
}
