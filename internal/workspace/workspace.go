// Package workspace locates and loads a Sameway workspace folder.
//
// A workspace is a plain, git-friendly directory:
//
//	workspace.yaml   name, server, llm, chat settings
//	schema/          one YAML file per content type
//	components/      local components that add to or override built-ins
//	content/         every record as Markdown with front matter, kept current
//	data.db          live SQLite store, ignored by git
package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// ConfigFile is the marker file every workspace has.
const ConfigFile = "workspace.yaml"

// Config is the parsed workspace.yaml.
type Config struct {
	Name   string `yaml:"name"`
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	LLM llm.Config `yaml:"llm"`
	UI  struct {
		// Controls is "auto" (per-item controls and provenance labels fade
		// until hovered or focused) or "visible" (always shown). Either way
		// they stay in the DOM, the tab order, and the accessibility tree.
		Controls string `yaml:"controls"`
		// Pace is how changes arrive: "calm" (the default: where, then what,
		// then the words, one change at a time with a pause between), "quick"
		// (the same in a third of the time) or "still" (everything at once,
		// as under reduced motion). A person sets it by asking the assistant.
		Pace string `yaml:"pace"`
	} `yaml:"ui"`
	Chat struct {
		// HistoryLimit caps how many past messages are sent to the model.
		HistoryLimit int `yaml:"history_limit"`
		// SystemPrompt is prepended to the built-in instructions.
		SystemPrompt string `yaml:"system_prompt"`
	} `yaml:"chat"`
}

// Workspace is a loaded workspace.
type Workspace struct {
	Dir    string
	Config Config
}

// ErrNotFound is returned when no workspace.yaml is found.
var ErrNotFound = errors.New("no workspace.yaml found here or in any parent folder (run `sameway init` to create one)")

// Find walks up from start until it finds a folder containing workspace.yaml.
func Find(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ConfigFile)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// Load reads the workspace at dir.
func Load(dir string) (*Workspace, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	src, err := os.ReadFile(filepath.Join(abs, ConfigFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	w := &Workspace{Dir: abs}
	if err := yaml.Unmarshal(src, &w.Config); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(abs, ConfigFile), err)
	}
	if w.Config.Name == "" {
		w.Config.Name = filepath.Base(abs)
	}
	if w.Config.Server.Addr == "" {
		w.Config.Server.Addr = "127.0.0.1:8080"
	}
	if w.Config.Chat.HistoryLimit == 0 {
		w.Config.Chat.HistoryLimit = 40
	}
	if w.Config.UI.Controls != "visible" {
		w.Config.UI.Controls = "auto"
	}
	if !ValidPace(w.Config.UI.Pace) {
		w.Config.UI.Pace = "calm"
	}
	return w, nil
}

// Paces are the ways changes can arrive.
var Paces = []string{"calm", "quick", "still"}

// ValidPace says whether a pace is one of the three.
func ValidPace(p string) bool {
	for _, x := range Paces {
		if x == p {
			return true
		}
	}
	return false
}

// SetPace records a pace in workspace.yaml, editing the one line rather
// than rewriting the file, so the comments a person reads there survive.
func (w *Workspace) SetPace(pace string) error {
	if !ValidPace(pace) {
		return fmt.Errorf("pace must be one of %s, not %q", strings.Join(Paces, ", "), pace)
	}
	path := filepath.Join(w.Dir, ConfigFile)
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	var out []string
	done, inUI := false, false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "ui:"):
			inUI = true
		case inUI && strings.HasPrefix(trim, "pace:") && !done:
			line, done = "  pace: "+pace, true
		case inUI && line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t"):
			// The ui block ended without a pace line: it goes at the end.
			if !done {
				out, done = append(out, "  pace: "+pace), true
			}
			inUI = false
		}
		out = append(out, line)
	}
	if !done {
		if !inUI {
			out = append(out, "ui:")
		}
		out = append(out, "  pace: "+pace)
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		return err
	}
	w.Config.UI.Pace = pace
	return nil
}

// SchemaDir is where content types live.
func (w *Workspace) SchemaDir() string { return filepath.Join(w.Dir, "schema") }

// ComponentsDir is where local components live.
func (w *Workspace) ComponentsDir() string { return filepath.Join(w.Dir, "components") }

// ContentDir is where the portable form of every record lives.
func (w *Workspace) ContentDir() string { return filepath.Join(w.Dir, "content") }

// DBPath is the SQLite file.
func (w *Workspace) DBPath() string { return filepath.Join(w.Dir, "data.db") }

// Init copies a template workspace from src/root into dir. dir must not
// already contain a workspace unless force is set.
func Init(dir string, src fs.FS, root string, force bool) error {
	if _, err := os.Stat(filepath.Join(dir, ConfigFile)); err == nil && !force {
		return fmt.Errorf("%s already contains a workspace (use --force to overwrite its config and schema)", dir)
	}
	return fs.WalkDir(src, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
