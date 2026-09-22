package update

import (
	"context"
	"time"
)

// Every is how often a running workspace looks for a new version. Once a
// day: a release is not news that has to arrive within the hour.
const Every = 24 * time.Hour

// Watch looks for a new version while ctx lasts: once at the start and
// then every so often. In auto mode it installs what it finds; in manual
// mode it only says so and waits to be asked. tell hears about each thing
// once, whatever happens after: a server left running for weeks does not
// repeat itself.
//
// mode is read at each check rather than held, so a person who switches
// update.mode does not have to restart for it to mean anything.
//
// A build that cannot say which version it is does not watch at all.
func (u Updater) Watch(ctx context.Context, mode func() string, every time.Duration, tell func(Outcome, error)) {
	if !Known(u.current()) {
		return
	}
	if every <= 0 {
		every = Every
	}
	go func() {
		tick := time.NewTicker(every)
		defer tick.Stop()
		// The running program keeps its old version until it is restarted,
		// so a release already installed is remembered here; otherwise the
		// next check would install the same one again.
		here, said := map[string]bool{}, map[string]bool{}
		once := func(key string, out Outcome, err error) {
			if !said[key] {
				said[key] = true
				tell(out, err)
			}
		}
		for {
			rel, err := u.Latest(ctx)
			switch {
			case err != nil:
				once("failed "+err.Error(), Outcome{Current: u.current(), Says: err.Error()}, err)
			case !Newer(u.current(), rel.Version) || here[rel.Version]:
			case mode() != Auto:
				out := u.compare(rel)
				out.Says += AskFor
				once("found "+rel.Version, out, nil)
			default:
				out := u.compare(rel)
				where, err := u.install(ctx, rel)
				if err != nil {
					once("install "+rel.Version+" "+err.Error(), out, err)
					break
				}
				here[rel.Version] = true
				out.Installed, out.Path, out.Says = true, where, Installed(rel.Version)
				tell(out, nil)
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}
