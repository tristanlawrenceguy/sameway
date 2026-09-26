// Package workspace locates and loads a Sameway workspace folder.
//
// A workspace is a plain, git-friendly directory:
//
//	workspace.yaml   name, server, llm, chat, update settings
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

	"gopkg.in/yaml.v3"

	"github.com/tristanlawrenceguy/sameway/internal/devices"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/tailnet"
	"github.com/tristanlawrenceguy/sameway/internal/update"
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
	// MCP over HTTP at /mcp, for clients elsewhere: ChatGPT's connectors,
	// Claude's custom connectors, a hosted agent. TokenEnv names the
	// environment variable holding the bearer token; unset means off.
	MCP struct {
		TokenEnv string `yaml:"token_env"`
	} `yaml:"mcp"`
	UI struct {
		// Controls is "auto" (per-item controls and provenance labels fade
		// until hovered or focused) or "visible" (always shown). Either way
		// they stay in the DOM, the tab order, and the accessibility tree.
		Controls string `yaml:"controls"`
		// Pace is how changes arrive: "calm" (the default: where, then what,
		// then the words, one change at a time with a pause between), "quick"
		// (the same in a third of the time) or "still" (everything at once,
		// as under reduced motion). A person sets it by asking the assistant.
		Pace string `yaml:"pace"`
		// Lists is which lists the sidebar shows: "filled" (the default: a
		// list with something in it, or one the person made themselves,
		// so an empty built-in list such as files is not in the way) or
		// "all".
		Lists string `yaml:"lists"`
		// Developer is "hidden" (the default) or "shown": the design
		// system and the guide for agents are for whoever builds on the
		// workspace, not for the person using it, so their links stay out
		// of the way unless asked for. The pages themselves are always
		// there.
		Developer string `yaml:"developer"`
		// Show names the parts of a page that are on every time, comma
		// separated, when the default is off: fields, remind, or a
		// connection key such as points-here:task.project. Empty, the
		// default, is none of them, and each is still one link away in
		// the address (?show=<key>). The assistant adds one with
		// set_setting ui.show +<key> when it has a reason the person
		// would want it there always, and takes it back with -<key>.
		Show string `yaml:"show"`
		// Text is how large the words are: "normal" (the default),
		// "large" or "larger". Spacing is "normal" or "wide": more room
		// between lines, words and paragraphs, for people who read more
		// easily that way. Set by asking, or on the Help page.
		Text    string `yaml:"text"`
		Spacing string `yaml:"spacing"`
		// Needs is what the person has said they need, in their words
		// ("I use a screen reader", "keep things simple"): the assistant
		// follows it in every reply and every page it makes.
		Needs string `yaml:"needs"`
		// Language is the workspace's language, as a code such as en,
		// de or es: the pages say it, so screen readers read them in the
		// right voice, and the assistant replies in it. English when empty.
		Language string `yaml:"language"`
	} `yaml:"ui"`
	Chat struct {
		// HistoryLimit caps how many past messages are sent to the model.
		HistoryLimit int `yaml:"history_limit"`
		// SystemPrompt is prepended to the built-in instructions.
		SystemPrompt string `yaml:"system_prompt"`
	} `yaml:"chat"`
	// Actions bounds what a command action may do: Allow names the
	// programs it may run, comma separated (curl, python). Empty means
	// command actions run nothing; webhooks and mqtt actions are not
	// programs and are not bound by it.
	Actions struct {
		Allow string `yaml:"allow"`
	} `yaml:"actions"`
	// MQTT is the broker the workspace talks to for devices: what it
	// subscribes to becomes device records, and mqtt actions publish.
	MQTT devices.Config `yaml:"mqtt"`
	// Tailnet puts the workspace on the person's Tailscale network, so a
	// phone signed in to it opens the workspace from anywhere:
	//   tailnet:
	//     name: home      # https://home.<tailnet>.ts.net
	Tailnet tailnet.Config `yaml:"tailnet"`
	// Publish is what anyone on the internet may read, with no login,
	// through Tailscale Funnel at the workspace's tailnet address: tabs by
	// name and content types by name, comma separated, and whether AI
	// services may read it too (ai: on). Empty is nothing; see publish.go.
	Publish struct {
		Tabs  string `yaml:"tabs"`
		Types string `yaml:"types"`
		AI    string `yaml:"ai"`
	} `yaml:"publish"`
	// Notify is how a reminder reaches a person beyond an open page: a
	// notification on this machine ("on", the default, or "off"), and a
	// command run for each ring with {title}, {text} and {url} in its
	// arguments, for a push service, an email or a text. For example:
	//   notify:
	//     desktop: on
	//     command: curl -d "{title}" ntfy.sh/my-topic
	Notify struct {
		Desktop string `yaml:"desktop"`
		Command string `yaml:"command"`
	} `yaml:"notify"`
	// Update is how a new version of sameway arrives: "auto" (the
	// default) installs a release on its own and says so in the activity
	// log, "manual" only says one is there and waits to be asked. Either
	// way a new version runs from the next start, and a build that cannot
	// say which version it is never replaces itself.
	Update struct {
		Mode string `yaml:"mode"`
	} `yaml:"update"`
	Files struct {
		// Convert names an external converter per file extension, for the
		// formats the built-in readers cannot do justice to: a URL such as
		// docling's server, or a command line with {file} where the path
		// goes. Either answers with Markdown. For example:
		//   convert:
		//     pdf: http://127.0.0.1:5001/v1/convert/file
		//     docx: pandoc {file} -t gfm
		Convert map[string]string `yaml:"convert"`
	} `yaml:"files"`
}

// FilesDir is where the originals of added files are kept.
func (w *Workspace) FilesDir() string { return filepath.Join(w.Dir, "files") }

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
	if w.Config.MCP.TokenEnv == "" {
		w.Config.MCP.TokenEnv = "SAMEWAY_MCP_TOKEN"
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
	if w.Config.UI.Lists != "all" {
		w.Config.UI.Lists = "filled"
	}
	if w.Config.UI.Developer != "shown" {
		w.Config.UI.Developer = "hidden"
	}
	if w.Config.Update.Mode != update.Manual {
		w.Config.Update.Mode = update.Auto
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
