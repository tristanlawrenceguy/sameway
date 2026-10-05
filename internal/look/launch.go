package look

import (
	"context"
	"errors"
	"strings"
	"time"
)

// A browser started cold on a busy machine can take longer than its wait
// to open its DevTools port: CI saw /usr/bin/google-chrome miss it once,
// and a look with scripts failed as though the page had. One more start,
// with a fresh profile, while there is time left, before saying so; and
// what the browser itself said goes with the error.

// portWait is how long a browser has to open its port, each time.
var portWait = 20 * time.Second

// errNoPort is a browser that did not open its DevTools port.
type errNoPort struct{ program, said string }

func (e errNoPort) Error() string {
	if e.said == "" {
		return e.program + " did not open its DevTools port"
	}
	return e.program + " did not open its DevTools port: " + e.said
}

// launch starts the browser, twice when the first did not open its port.
func launch(ctx context.Context, program string) (*browser, error) {
	b, err := start(ctx, program)
	var slow errNoPort
	if err == nil || !errors.As(err, &slow) || ctx.Err() != nil {
		return b, err
	}
	if d, ok := ctx.Deadline(); ok && time.Until(d) < 5*time.Second {
		return nil, err
	}
	return start(ctx, program)
}

// lastLine is the last thing a program wrote that says something.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if len(last) > 200 {
		last = last[:200]
	}
	return last
}
