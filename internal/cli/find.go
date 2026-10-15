package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// workspaceHere is the workspace a command works on: the one named with
// --workspace, else the one this folder is in. Outside any, it says how to
// name one and which this computer knows, open or not: an agent run from
// another folder was told only that there was no workspace.yaml, and went
// looking for one, though the one it wanted was open.
func (c *ctx) workspaceHere() (string, error) {
	if c.workspaceDir != "" {
		return c.workspaceDir, nil
	}
	start := c.Dir
	if start == "" {
		start, _ = os.Getwd()
	}
	found, err := workspace.Find(start)
	if !errors.Is(err, workspace.ErrNotFound) {
		return found, err
	}
	known := workspace.ThisMachine().KnownWorkspaces()
	if len(known) == 0 {
		return "", err
	}
	var said []string
	for _, k := range known[:min(5, len(known))] {
		w := k.Dir
		if k.Addr != "" && answersWithin("http://"+k.Addr, time.Second) {
			w += " (open at http://" + k.Addr + "/)"
		}
		said = append(said, w)
	}
	return "", notHere(fmt.Sprintf("no workspace.yaml in %s or any folder above it. Name one with --workspace <dir> (or SAMEWAY_WORKSPACE); this computer knows %s (sameway workspaces lists them). Or run `sameway init` to make one", start, strings.Join(said, ", ")))
}

// notHere is workspace.ErrNotFound in words that name the workspaces known.
type notHere string

func (e notHere) Error() string        { return string(e) }
func (e notHere) Is(target error) bool { return target == workspace.ErrNotFound }
