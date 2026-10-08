package records

import (
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Book is one workspace's records with what keeping them needs beyond the
// store: its settings, which a change can set and an undo set back; whose
// copy this is, which the log names; and what a record says at a glance.
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
	// Glance is what a record says beside its title, in the words its row
	// and its page use (server/glance.go); set by the server.
	Glance func(t *schema.Type, rec *store.Record) string
}

// setting is one line of workspace.yaml, or "" where there is none.
func (b *Book) setting(key string) string {
	if b.Setting == nil {
		return ""
	}
	return b.Setting(key)
}
