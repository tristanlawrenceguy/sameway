// Package workspace locates and loads a Sameway workspace folder.
//
// A workspace is a plain, git-friendly directory:
//
//	workspace.yaml   name, server, llm, chat settings
//	schema/          one YAML file per content type
//	components/      local components that add to or override built-ins
//	content/         exported records (Markdown with front matter)
//	data.db          live SQLite store, ignored by git
package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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
	LLM  llm.Config `yaml:"llm"`
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
	return w, nil
}

// SchemaDir is where content types live.
func (w *Workspace) SchemaDir() string { return filepath.Join(w.Dir, "schema") }

// ComponentsDir is where local components live.
func (w *Workspace) ComponentsDir() string { return filepath.Join(w.Dir, "components") }

// ContentDir is where exported records live.
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
