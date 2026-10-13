package main

import "testing"

const clockSrc = `package server

import "time"

func a() time.Time { return time.Now() }

func b() time.Time {
	return time.Now() // wall clock: how long a request waits
}

var c = time.Now
`

// A time.Now() under the clock dirs fails unless its line says why it is
// real time; outside them, and in tests, it is not looked at.
func TestNewTimeNowFailsUnlessMarked(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "internal/server/a.go", clockSrc)
	write(t, root, "internal/server/a_test.go", "package server\n\nimport \"time\"\n\nvar d = time.Now()\n")
	write(t, root, "internal/cli/a.go", "package cli\n\nimport \"time\"\n\nvar d = time.Now()\n")
	r := checkWallClock(root, map[string]int{})
	if len(r.problems) != 1 || !has(r.problems, "internal/server/a.go: time.Now() or time.Local read 2 time(s)") {
		t.Errorf("problems = %q, want a.go with two (the marked one left out)", r.problems)
	}
}

const zoneSrc = `package blocks

import "time"

func a(t time.Time) time.Time { return t.Local() }

var b = time.Local

var c = time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local) // wall clock: a file's own time
`

// The computer's zone is the computer's clock's: a day worked out in it
// is not the person's when the workspace is hosted elsewhere.
func TestComputersZoneFailsUnlessMarked(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "internal/blocks/a.go", zoneSrc)
	r := checkWallClock(root, map[string]int{})
	if len(r.problems) != 1 || !has(r.problems, "internal/blocks/a.go: time.Now() or time.Local read 2 time(s)") {
		t.Errorf("problems = %q, want a.go with two (the marked one left out)", r.problems)
	}
}

func TestMarkedTimeNowPasses(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "internal/chat/a.go", "package chat\n\nimport \"time\"\n\nvar d = time.Now() // wall clock: a log line's time\n")
	if r := checkWallClock(root, map[string]int{}); len(r.problems)+len(r.warnings) != 0 {
		t.Errorf("report = %+v, want nothing", r)
	}
}

// A recorded file may read time.Now() as often as recorded or less.
func TestRecordedFileMayOnlyShrink(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "internal/server/same.go", clockSrc)
	write(t, root, "internal/server/grown.go", clockSrc)
	write(t, root, "internal/records/shrunk.go", clockSrc)
	r := checkWallClock(root, map[string]int{
		"internal/server/same.go":    2,
		"internal/server/grown.go":   1,
		"internal/records/shrunk.go": 3,
		"internal/blocks/gone.go":    1,
	})
	if len(r.problems) != 1 || !has(r.problems, "grown.go: time.Now() or time.Local read 2 times, more than the 1 recorded") {
		t.Errorf("problems = %q, want grown alone", r.problems)
	}
	if len(r.warnings) != 2 || !has(r.warnings, "shrunk.go: time.Now() down to 2 from 3") || !has(r.warnings, "gone.go reads time.Now() unexplained no more") {
		t.Errorf("warnings = %q, want shrunk and gone", r.warnings)
	}
}

// Against the base, wallClock may lose entries and lower counts, never
// gain or raise one.
func TestWallClockListMayOnlyGoDown(t *testing.T) {
	t.Parallel()
	was, err := parseDebt([]byte("package main\n\nvar wallClock = map[string]int{\n\t\"internal/server/a.go\": 2,\n\t\"internal/server/b.go\": 1,\n\t\"internal/server/c.go\": 3,\n}\n"))
	if err != nil {
		t.Fatal(err)
	}
	now := map[string]map[string]int{"wallClock": {"internal/server/a.go": 3, "internal/server/b.go": 1, "internal/server/d.go": 1}}
	r := compareDebt(was, now)
	if len(r.problems) != 2 || !has(r.problems, "a.go: raised from 2 to 3 in wallClock") || !has(r.problems, "d.go: added to wallClock") {
		t.Errorf("problems = %q, want a raised and d added (c removed is fine)", r.problems)
	}
}
