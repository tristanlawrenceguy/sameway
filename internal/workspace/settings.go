package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// A person changes a setting by asking: the assistant's set_setting tool
// lands here, and the one line in workspace.yaml changes, so the comments
// a person reads there survive. Every setting the file holds is listed,
// so the assistant can change any of them; the ones that hold a secret
// take the name of an environment variable, never the secret.

// Setting is one line of workspace.yaml a person can change by asking.
type Setting struct {
	Key    string   // section.key, or a top-level key
	Kind   string   // enum, int, string or env (the name of an environment variable)
	Values []string // what an enum may be
	Doc    string
}

// Settings are the lines of workspace.yaml, in the order the file has them.
var Settings = []Setting{
	{"name", "string", nil, "what the workspace is called, on every page"},
	{"server.addr", "string", nil, "the address to serve on, from the next start"},
	{"llm.provider", "string", nil, "the kind of model: anthropic, openai-compatible, ollama, claude-code or command"},
	{"llm.model", "string", nil, "the model's name"},
	{"llm.base_url", "string", nil, "where an openai-compatible model answers"},
	{"llm.max_tokens", "int", nil, "the longest reply the model may give"},
	{"llm.api_key_env", "env", nil, "the NAME of the environment variable that holds the model's key"},
	{"mcp.token_env", "env", nil, "the NAME of the environment variable that holds the bearer token for /mcp"},
	{"ui.controls", "enum", []string{"auto", "visible"}, "auto fades per-item controls until hovered; visible keeps them on screen"},
	{"ui.pace", "enum", Paces, "how changes arrive: calm, quick or still"},
	{"ui.text", "enum", []string{"normal", "large", "larger"}, "how large the words are: normal, large or larger"},
	{"ui.spacing", "enum", []string{"normal", "wide"}, "room between lines, words and paragraphs: normal, or wide for people who read more easily with more room"},
	{"ui.needs", "string", nil, "what the person has said they need, in their words (I use a screen reader; keep things simple; I am colour blind): you follow it in every reply and every page you make. Set it the moment they tell you, and add to it, keeping what was there"},
	{"ui.language", "string", nil, "the workspace's language as a code (en, de, es, fr): the pages say it so screen readers use the right voice, and you reply in it"},
	{"ui.lists", "enum", []string{"filled", "all"}, "which lists the sidebar shows: filled (something in them, or made by the person) or all"},
	{"ui.developer", "enum", []string{"hidden", "shown"}, "the design system and the guide for agents: hidden from the sidebar or shown"},
	{"ui.show", "keys", nil, "the parts of a page that are on every time. All of them are off by default, and a page shows no trace of an off one, so this is how something earns a permanent place: fields (a record's whole field list, including the ones its heading and chips already say), remind (the field for setting a reminder about a record, on its page), ask (the way to the assistant with the record in the box), day (the way to the record's day on the calendar), or a connection key from get_record's related, such as points-here:task.project. +key adds one, -key takes it back, a list replaces them all, empty is none. Each is also one address away without this (?show=<key>), so turn one on only when you have a reason the person wants it every time, and say the reason"},
	{"chat.history_limit", "int", nil, "how many past messages go to the model each turn"},
	{"chat.system_prompt", "string", nil, "words put before the built-in instructions to the model"},
	{"update.mode", "enum", update.Modes, "how a new version of sameway arrives: auto installs a release on its own and says so in the activity log, manual only says one is there and waits to be asked (either way it runs from the next start)"},
	{"notify.desktop", "enum", []string{"on", "off"}, "a notification on this machine when a reminder rings, whether or not a page is open"},
	{"notify.command", "string", nil, "a command run when a reminder rings, with {title}, {text} and {url} in its arguments: a push service such as ntfy, an email, a text"},
	{"actions.allow", "string", nil, "the programs a command action may run, by name, comma separated (curl, python); empty means command actions run nothing"},
	{"mqtt.broker", "string", nil, "the MQTT broker for devices, such as tcp://192.168.1.10:1883; empty means none (takes effect at the next start)"},
	{"mqtt.client_id", "string", nil, "how this workspace names itself to the broker"},
	{"mqtt.username_env", "env", nil, "the NAME of the environment variable that holds the broker username"},
	{"mqtt.password_env", "env", nil, "the NAME of the environment variable that holds the broker password"},
	{"tailnet.peers", "string", nil, "the other computers hosting this same workspace, by their machine name on the tailnet, comma separated (bob-home, my-laptop): this copy keeps in step with each, both ways. Each has to host a copy of this workspace and have this computer's owner or a host let in. Empty is none"},
	{"tailnet.name", "string", nil, "the name of this computer on the person's Tailscale network, so their phone and other devices signed in to Tailscale as them open the workspace from anywhere at https://<name>.<tailnet>.ts.net; empty is off. It takes effect at once, and the steps to finish (signing in, turning on HTTPS) come back from this call and appear in the chat. Offer it when the person wants the workspace on their phone or away from this computer"},
}

// SettingKeys lists what Set takes.
func SettingKeys() []string {
	keys := make([]string, 0, len(Settings))
	for _, s := range Settings {
		keys = append(keys, s.Key)
	}
	return keys
}

// SettingsDoc says each setting in a line, for a tool's description.
func SettingsDoc() string {
	var b strings.Builder
	for i, s := range Settings {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(s.Key + ": " + s.Doc)
		if s.Kind == "enum" {
			b.WriteString(" (" + strings.Join(s.Values, ", ") + ")")
		}
	}
	return b.String()
}

var envName = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)

// Set changes one setting in workspace.yaml and in the loaded config.
func (w *Workspace) Set(key, value string) error {
	var st *Setting
	for i := range Settings {
		if Settings[i].Key == key {
			st = &Settings[i]
		}
	}
	if st == nil {
		return fmt.Errorf("there is no setting %q; the settings are %s", key, strings.Join(SettingKeys(), ", "))
	}
	value = strings.TrimSpace(value)
	scalar := value
	switch st.Kind {
	case "enum":
		// Empty is the default, which is where a choice starts: undoing
		// the first change to one puts it back there.
		if value != "" && !contains(st.Values, value) {
			return fmt.Errorf("%s must be one of %s, not %q", key, strings.Join(st.Values, ", "), value)
		}
	case "int":
		if n, err := strconv.Atoi(value); err != nil || n <= 0 {
			return fmt.Errorf("%s must be a whole number above zero, not %q", key, value)
		}
	case "env":
		// Empty is no variable at all, which is where a setting starts.
		if value != "" && !envName.MatchString(value) {
			return fmt.Errorf("%s takes the name of an environment variable, such as OPENAI_API_KEY; a key or token itself is never written into workspace.yaml", key)
		}
	case "keys":
		// A list one thing is added to or taken from, rather than resent
		// whole: turning a part of a page on should not be able to turn
		// another one off by forgetting it.
		value = mergeKeys(w.Config.UI.Show, value)
		scalar = yamlScalar(value)
	default:
		if value == "" && !canBeEmpty[key] {
			return fmt.Errorf("%s needs a value", key)
		}
		scalar = yamlScalar(value)
	}
	path := filepath.Join(w.Dir, ConfigFile)
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(setLine(string(src), key, scalar)), 0o644); err != nil {
		return err
	}
	fresh, err := Load(w.Dir)
	if err != nil {
		return err
	}
	w.Config = fresh.Config
	return nil
}

// canBeEmpty are the settings where nothing is a value: no program on a
// reminder, no programs allowed, no broker. Undoing a change to one of
// them puts it back to nothing.
var canBeEmpty = map[string]bool{
	"notify.command": true, "actions.allow": true, "chat.system_prompt": true, "ui.needs": true, "ui.language": true,
	"mqtt.broker": true, "mqtt.client_id": true, "llm.base_url": true, "tailnet.name": true, "tailnet.peers": true,
}

// Get reads one setting as workspace.yaml has it now, or "" when the file
// does not say.
func (w *Workspace) Get(key string) string {
	src, err := os.ReadFile(filepath.Join(w.Dir, ConfigFile))
	if err != nil {
		return ""
	}
	section, name := "", key
	if i := strings.Index(key, "."); i > 0 {
		section, name = key[:i], key[i+1:]
	}
	in := section == ""
	for _, line := range strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n") {
		trim := strings.TrimSpace(line)
		top := line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "#")
		if section != "" && top {
			in = strings.HasPrefix(line, section+":")
			continue
		}
		if in && strings.HasPrefix(trim, name+":") {
			v := strings.TrimSpace(strings.TrimPrefix(trim, name+":"))
			if i := strings.Index(v, " #"); i >= 0 {
				v = strings.TrimSpace(v[:i])
			}
			return strings.Trim(v, `"'`)
		}
	}
	return ""
}

// SetPace records how changes arrive: the setting the assistant changed
// first, kept by name.
func (w *Workspace) SetPace(pace string) error { return w.Set("ui.pace", pace) }

// mergeKeys reads a change to a list of keys against what is there:
// +key adds one, -key takes one away, and anything else is the whole
// list, so a person can still say exactly which parts are on.
func mergeKeys(current, change string) string {
	var keys []string
	for _, k := range strings.Split(current, ",") {
		if k = strings.TrimSpace(k); k != "" {
			keys = append(keys, k)
		}
	}
	one := strings.TrimSpace(strings.TrimLeft(change, "+-"))
	switch {
	case strings.HasPrefix(change, "+"):
		if one != "" && !contains(keys, one) {
			keys = append(keys, one)
		}
	case strings.HasPrefix(change, "-"):
		var out []string
		for _, k := range keys {
			if k != one {
				out = append(out, k)
			}
		}
		keys = out
	default:
		keys = nil
		for _, k := range strings.Split(change, ",") {
			if k = strings.TrimSpace(k); k != "" && !contains(keys, k) {
				keys = append(keys, k)
			}
		}
	}
	return strings.Join(keys, ", ")
}

// yamlScalar writes a string the way YAML needs it on one line: quoted
// when it has to be, a block when it has lines of its own.
func yamlScalar(value string) string {
	if strings.Contains(value, "\n") {
		return "|-\n" + "    " + strings.ReplaceAll(value, "\n", "\n    ")
	}
	out, err := yaml.Marshal(value)
	if err != nil {
		return strconv.Quote(value)
	}
	return strings.TrimRight(string(out), "\n")
}

// setLine changes one line of a workspace.yaml: the key in its section,
// added at the section's end when it is not there, with the section
// added at the file's end when that is not there either.
func setLine(src, key, scalar string) string {
	section, name := "", key
	if i := strings.Index(key, "."); i > 0 {
		section, name = key[:i], key[i+1:]
	}
	indent := ""
	if section != "" {
		indent = "  "
	}
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	var out []string
	done, in := false, section == ""
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		top := line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "#")
		switch {
		case section != "" && strings.HasPrefix(line, section+":"):
			in = true
		case section != "" && in && top:
			if !done {
				out, done = append(out, indent+name+": "+scalar), true
			}
			in = false
		}
		if in && !done && strings.HasPrefix(trim, name+":") && (section != "" || top) {
			line, done = indent+name+": "+scalar, true
		}
		out = append(out, line)
	}
	if !done {
		if section != "" && !in {
			out = append(out, section+":")
		}
		out = append(out, indent+name+": "+scalar)
	}
	return strings.Join(out, "\n")
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
