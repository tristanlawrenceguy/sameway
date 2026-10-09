package records

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Book is one workspace's records with what keeping them needs beyond the
// store: its settings, which a change can set and an undo set back; and
// whose copy this is, which the log names.
type Book struct {
	Store *store.Store
	// SetSetting changes one line of workspace.yaml, when there is one:
	// the pace, which lists show, the model, the name. Set by the app.
	SetSetting func(key, value string) error
	// Setting reads one line of workspace.yaml as it is now, for the
	// questions that say what would change from what; set by the app.
	Setting func(key string) string
	// Owner is who owns this computer's copy, by their Tailscale login and
	// name, once the tailnet says; set by the command line. What they do
	// is theirs by name on the other computers that host the workspace.
	Owner Visitor
	// Clock is the time it is for the person, and Hours24 whether they
	// read times on the 24-hour clock: both the app's, set by it, so two
	// workspaces open at once each keep their own. Nil reads as this
	// computer's clock and the 12-hour one.
	Clock   func() time.Time
	Hours24 func() bool
}

// setting is one line of workspace.yaml, or "" where there is none.
func (b *Book) setting(key string) string {
	if b.Setting == nil {
		return ""
	}
	return b.Setting(key)
}

// Now is the time it is for the person (Clock).
func (b *Book) Now() time.Time {
	if b == nil || b.Clock == nil {
		return time.Now()
	}
	return b.Clock()
}

// H24 is whether the person reads times on the 24-hour clock (Hours24).
func (b *Book) H24() bool { return b != nil && b.Hours24 != nil && b.Hours24() }
