package cli

import (
	"fmt"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// The workspaces this computer knows, by command. An agent run outside any
// workspace learned them only from the error a command gave it, and went
// looking through folders for workspace.yaml: sameway workspaces says
// them, each with its name, its folder and the address it is open at, so
// --workspace can name one.

// knownWorkspace is one workspace this computer knows.
type knownWorkspace struct {
	Name string `json:"name,omitempty"`
	Dir  string `json:"dir"`
	// Open is where it answers now, or "" when it is not running.
	Open string `json:"open,omitempty"`
}

func (c *ctx) workspacesCmd() error {
	var list []knownWorkspace
	for _, k := range workspace.ThisMachine().KnownWorkspaces() {
		w := knownWorkspace{Dir: k.Dir}
		if ws, err := workspace.Load(k.Dir); err == nil {
			w.Name = ws.Config.Name
		}
		if k.Addr != "" && answersWithin("http://"+k.Addr, time.Second) {
			w.Open = "http://" + k.Addr + "/"
		}
		list = append(list, w)
	}
	if list == nil {
		list = []knownWorkspace{}
	}
	c.print(list, func() {
		if len(list) == 0 {
			fmt.Fprintln(c.Stdout, "This computer knows no workspace yet. sameway init makes one.")
			return
		}
		for _, w := range list {
			line := w.Dir
			if w.Name != "" {
				line = w.Name + "  " + line
			}
			if w.Open != "" {
				line += "  (open at " + w.Open + ")"
			}
			fmt.Fprintln(c.Stdout, line)
		}
		fmt.Fprintln(c.Stdout, "\nName one with --workspace <dir>.")
	})
	return nil
}
