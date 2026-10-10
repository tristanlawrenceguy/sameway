package update

import (
	"context"
	"time"
)

// Every is how often a running workspace looks for a new version. Once a
// day: a release is not news that has to arrive within the hour.
const Every = 24 * time.Hour

// Watcher looks for a new version each time it is asked to Check: the
// workspace's background jobs ask once at the start and then every so
// often (internal/runner). In auto mode it installs what it finds; in
// manual mode it only says so and waits to be asked. tell hears about
// each thing once, whatever happens after: a server left running for
// weeks does not repeat itself.
//
// mode is read at each check rather than held, so a person who switches
// update.mode does not have to restart for it to mean anything.
type Watcher struct {
	u    Updater
	mode func() string
	tell func(Outcome, error)
	// The running program keeps its old version until it is restarted,
	// so a release already installed is remembered here; otherwise the
	// next check would install the same one again.
	here, said map[string]bool
}

// Watcher is a watcher for u, or nil for a build that cannot say which
// version it is: that one does not watch at all.
func (u Updater) Watcher(mode func() string, tell func(Outcome, error)) *Watcher {
	if !Known(u.current()) {
		return nil
	}
	return &Watcher{u: u, mode: mode, tell: tell, here: map[string]bool{}, said: map[string]bool{}}
}

func (w *Watcher) once(key string, out Outcome, err error) {
	if !w.said[key] {
		w.said[key] = true
		w.tell(out, err)
	}
}

// Check looks once, and says what went wrong, if anything.
func (w *Watcher) Check(ctx context.Context) error {
	u := w.u
	rel, err := u.Latest(ctx)
	switch {
	case err != nil:
		w.once("failed "+err.Error(), Outcome{Current: u.current(), Says: err.Error()}, err)
		return err
	case !Newer(u.current(), rel.Version) || w.here[rel.Version]:
	case w.mode() != Auto:
		out := u.compare(rel)
		out.Says += AskFor
		w.once("found "+rel.Version, out, nil)
	default:
		out := u.compare(rel)
		where, err := u.install(ctx, rel)
		if err != nil {
			w.once("install "+rel.Version+" "+err.Error(), out, err)
			return err
		}
		w.here[rel.Version] = true
		setPending(rel.Version)
		out.Installed, out.Path, out.Says = true, where, Installed(rel.Version)
		w.tell(out, nil)
	}
	return nil
}
