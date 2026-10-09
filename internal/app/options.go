package app

import "time"

// Options are what an app takes from where it runs rather than from its
// workspace: what a test sets to have an app of its own, which shares
// nothing with another app in the same program.
type Options struct {
	// MemoryDB keeps the records in memory: tests, read-only commands.
	MemoryDB bool
	// Clock is the time it is for the person, for everything said and
	// worked out by the day (today, overdue, rings at); nil is this
	// computer's clock. A test fixes it, so "tomorrow" is never said a
	// minute before midnight about a minute after.
	Clock func() time.Time
}

// Load opens the workspace at dir. Pass memoryDB to use an in-memory store
// (tests and read-only commands).
func Load(dir string, memoryDB bool) (*App, error) {
	return Open(dir, Options{MemoryDB: memoryDB})
}

// Now is the time it is for the person (Options.Clock).
func (a *App) Now() time.Time { return a.Records.Now() }
