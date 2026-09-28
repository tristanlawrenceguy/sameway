package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// The files in schema/ are the workspace's content types, and more than
// one process keeps them: the server, and beside it `sameway mcp`, which
// Claude Code starts to run the assistant's tools when chat goes through
// it (llm provider claude-code), or any other MCP client, the CLI, a
// person with an editor, git pulling from another computer. Records need
// nothing for this: every process opens the same data.db, and SQLite
// shows each what the others wrote. The types are held in memory,
// though, so each process looks at schema/ again (ReloadSchema) and takes
// what changed there, as it would have at start-up.

// SchemaEvery is how often a running workspace looks at schema/.
const SchemaEvery = time.Second

type schemaWatch struct {
	mu   sync.Mutex
	seen string
}

// loadTypes reads a workspace's content types as start-up does.
func loadTypes(dir string) (*schema.Set, error) {
	types, err := schema.Load(dir)
	if err != nil {
		return nil, err
	}
	// The system owns its internal types. A workspace created before a
	// field existed still gets that field, so the tools always work.
	builtin, err := schema.LoadFS(examples.FS, examples.StarterRoot+"/schema")
	if err != nil {
		return nil, err
	}
	types.Complete(builtin)
	if err := types.CheckRefs(); err != nil {
		return nil, err
	}
	return types, nil
}

// schemaStamp says what schema/ holds, cheaply: each file's name, size
// and time of writing. Nothing to read unless it moves.
func schemaStamp(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	lines := []string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue // gone between the listing and now; the next look sees it
		}
		lines = append(lines, fmt.Sprintf("%s %d %d", e.Name(), info.Size(), info.ModTime().UnixNano()))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

// ReloadSchema takes the content types as schema/ now has them, when it
// has changed since the last look: a new type joins, a changed one is
// replaced where it is (so everything holding it sees the change), one
// whose file went is dropped, its table and records kept, and the tables
// get what is new. It says whether anything changed. Files that do not
// read are left alone, with the error, until they change again.
func (a *App) ReloadSchema() (bool, error) {
	w := &a.schemaSeen
	w.mu.Lock()
	defer w.mu.Unlock()
	dir := a.Workspace.SchemaDir()
	stamp, err := schemaStamp(dir)
	if err != nil || stamp == w.seen {
		return false, err
	}
	w.seen = stamp
	fresh, err := loadTypes(dir)
	if err != nil {
		return false, fmt.Errorf("schema/ changed but was not taken: %w", err)
	}
	var changed []*schema.Type
	for _, t := range fresh.Types {
		if have, ok := a.Types.Get(t.Name); ok && reflect.DeepEqual(*have, *t) {
			continue
		}
		a.Types.Put(t)
		have, _ := a.Types.Get(t.Name)
		changed = append(changed, have)
	}
	gone := false
	for _, name := range a.Types.Names() {
		if _, ok := fresh.Get(name); !ok {
			a.Types.Remove(name)
			gone = true
		}
	}
	if len(changed) == 0 {
		return gone, nil
	}
	if err := a.Store.Migrate(); err != nil {
		return true, err
	}
	// A change made by hand travels to the other computers like one asked
	// for; one the other process made was stamped there, and stays so.
	for _, t := range changed {
		a.Store.StampSchema(t)
	}
	return true, nil
}

// WatchSchema looks at schema/ every so often until ctx ends, and tells
// changed (when given) each time the content types changed, so open
// pages follow.
func (a *App) WatchSchema(ctx context.Context, every time.Duration, changed func()) {
	go func() {
		tick := time.NewTicker(every)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			ok, err := a.ReloadSchema()
			if err != nil {
				log.Printf("schema: %v", err)
			}
			if ok && changed != nil {
				changed()
			}
		}
	}()
}

// writeSchema writes a type's file whole or not at all: to a file beside
// it first, then renamed over it, so another process looking at schema/
// never reads half of one. Windows refuses the rename while someone is
// reading the old file, for a moment, so it tries again.
func (a *App) writeSchema(path string, src []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	_, err = tmp.Write(src)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), 0o644)
	}
	if err == nil {
		for i := 0; i < 40; i++ {
			if err = os.Rename(tmp.Name(), path); err == nil {
				return nil
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	os.Remove(tmp.Name())
	return err
}
