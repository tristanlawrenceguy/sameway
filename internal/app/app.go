// Package app wires a workspace, its schema, store, components, and chat
// into one object shared by the CLI and the HTTP server.
package app

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/content"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/update"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// App is a loaded workspace ready to serve.
type App struct {
	Workspace *workspace.Workspace
	Types     *schema.Set
	Store     *store.Store
	Registry  *render.Registry
	// Mirror keeps content/ as the portable form of every record.
	Mirror content.Mirror
	Chat   *chat.Service
}

// Load opens the workspace at dir. Pass memoryDB to use an in-memory store
// (tests and read-only commands).
func Load(dir string, memoryDB bool) (*App, error) {
	ws, err := workspace.Load(dir)
	if err != nil {
		return nil, err
	}
	types, err := schema.Load(ws.SchemaDir())
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
	dbPath := ws.DBPath()
	if memoryDB {
		dbPath = ":memory:"
	}
	st, err := store.Open(dbPath, types)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dbPath, err)
	}
	reg, err := NewRegistry(ws.ComponentsDir())
	if err != nil {
		st.Close()
		return nil, err
	}
	a := &App{Workspace: ws, Types: types, Store: st, Registry: reg}
	// The conversation, its questions and the log are history, not content;
	// everything else is written to content/ as it changes.
	a.Mirror = content.Mirror{Dir: ws.ContentDir(), Types: types, Skip: []string{chat.MessageType, chat.ConversationType, chat.ProposalType, chat.ActivityType}}
	st.AfterWrite = a.Mirror.Changed
	a.Chat = &chat.Service{
		Store:        st,
		Registry:     reg,
		HistoryLimit: ws.Config.Chat.HistoryLimit,
		ExtraPrompt:  ws.Config.Chat.SystemPrompt,
	}
	llmCfg := ws.Config.LLM
	llmCfg.Workspace = ws.Dir
	llmCfg.Executable, _ = os.Executable()
	a.Chat.Provider, a.Chat.ProviderErr = llm.New(llmCfg)
	a.Chat.Allow = allowList(ws.Config.Actions.Allow)
	a.Chat.SetSetting = func(key, value string) error {
		if err := ws.Set(key, value); err != nil {
			return err
		}
		if key == "actions.allow" {
			a.Chat.Allow = allowList(ws.Config.Actions.Allow)
		}
		// A new model setting is a new model: the next message goes to it.
		if strings.HasPrefix(key, "llm.") {
			cfg := ws.Config.LLM
			cfg.Workspace = ws.Dir
			cfg.Executable, _ = os.Executable()
			a.Chat.Provider, a.Chat.ProviderErr = llm.New(cfg)
		}
		return nil
	}
	a.Chat.AddField, a.Chat.AddType = a.AddField, a.AddType
	// Keeping the program current is the program's own business, not the
	// workspace's: the updater needs nothing from here.
	a.Chat.Update = update.Updater{}.Run
	chat.Workdir = ws.Dir
	return a, nil
}

// NewRegistry loads the built-in components and then the workspace ones.
func NewRegistry(workspaceComponents string) (*render.Registry, error) {
	reg := render.New()
	tokensCSS, err := fs.ReadFile(design.FS, "tokens/tokens.css")
	if err != nil {
		return nil, fmt.Errorf("design tokens missing; run `go run ./tools/tokens`: %w", err)
	}
	baseCSS, err := baseStyles()
	if err != nil {
		return nil, err
	}
	reg.SetBase(string(tokensCSS), baseCSS)
	baseJS, err := baseScripts()
	if err != nil {
		return nil, err
	}
	reg.SetBaseJS(baseJS)
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		return nil, err
	}
	if err := reg.LoadArrangementsFS(design.FS, "arrangements", "builtin"); err != nil {
		return nil, err
	}
	if workspaceComponents != "" {
		if err := reg.LoadDir(workspaceComponents, "workspace"); err != nil {
			return nil, err
		}
		// A workspace may keep arrangements of its own beside its components.
		if err := reg.LoadArrangementsDir(filepath.Join(filepath.Dir(workspaceComponents), "arrangements"), "workspace"); err != nil {
			return nil, err
		}
	}
	return reg, nil
}

// baseStyles concatenates every stylesheet in design/base in filename
// order. The files are numbered so the order is deterministic and each one
// covers a single concern.
func baseStyles() (string, error) {
	return concatBase(".css")
}

func concatBase(ext string) (string, error) {
	entries, err := fs.ReadDir(design.FS, "base")
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ext) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		src, err := fs.ReadFile(design.FS, "base/"+n)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "\n/* base: %s */\n%s", n, src)
	}
	return b.String(), nil
}

// baseScripts concatenates every script in design/base, the same way its
// stylesheets are concatenated.
func baseScripts() (string, error) {
	return concatBase(".js")
}

// Close releases the store.
func (a *App) Close() error { return a.Store.Close() }

// Description is the machine-readable summary served at /api/describe and
// printed by `sameway describe --json`.
type Description struct {
	Workspace  string               `json:"workspace"`
	Dir        string               `json:"dir"`
	LLM        DescribedLLM         `json:"llm"`
	Types      []DescribedType      `json:"types"`
	Components []DescribedComponent `json:"components"`
	// Arrangements are whole pages of thought the assistant can apply.
	Arrangements []*render.Arrangement `json:"arrangements"`
	// Tools is what the chat assistant can do, straight from the tool loop,
	// so an agent reads the same list the model is given.
	Tools  []DescribedTool   `json:"tools"`
	Routes map[string]string `json:"routes"`
}

// DescribedTool is one tool the assistant can call, with its argument schema.
type DescribedTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"`
}

// DescribedLLM says which model the chat uses without exposing keys.
type DescribedLLM struct {
	Provider string `json:"provider"`
	Model    string `json:"model,omitempty"`
	Ready    bool   `json:"ready"`
	Problem  string `json:"problem,omitempty"`
}

// DescribedType is one content type with its JSON Schema.
type DescribedType struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Internal    bool           `json:"internal,omitempty"`
	Title       string         `json:"title_field,omitempty"`
	Fields      []schema.Field `json:"fields"`
	Schema      map[string]any `json:"schema"`
	Count       int            `json:"count"`
}

// DescribedComponent is one component's full manifest plus its source.
type DescribedComponent struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description"`
	// Use is the thought behind the component: when it serves a person.
	Use     *render.Use     `json:"use,omitempty"`
	Props   json.RawMessage `json:"props"`
	A11y    json.RawMessage `json:"a11y,omitempty"`
	Machine json.RawMessage `json:"machine,omitempty"`
}

// Describe builds the description from live state.
func (a *App) Describe() Description {
	d := Description{
		Workspace: a.Workspace.Config.Name,
		Dir:       a.Workspace.Dir,
		LLM:       DescribedLLM{Provider: a.Workspace.Config.LLM.Provider, Model: a.Workspace.Config.LLM.Model, Ready: a.Chat.Provider != nil},
		Routes: map[string]string{
			"describe":         "GET /api/describe; one part: GET /api/describe/{types|components|arrangements|tools|routes|llm}; one item: GET /api/describe/types/{name}, likewise components and tools",
			"list":             "GET /api/{type}; ?where=<condition> (repeatable) and ?order=<field|-field> take the same query a collection block does: status=draft, due<=+7d, title~garden, tags=health, notes= (empty); dates today, tomorrow, +7d, -1w, 2026-10-01. The list page /t/{type} takes the same ?where= and ?order=",
			"create":           "POST /api/{type} with a JSON object of fields",
			"import":           "POST /api/import/{type} with {\"file\": <id of a file record>, \"mapping\": {column: field}} makes records from a CSV, a vCard or a mailbox the person added (upload first with POST /api/file/upload); the answer says how many were made, skipped and linked to people",
			"get":              "GET /api/{type}/{id}",
			"update":           "PUT or PATCH /api/{type}/{id} with a JSON object of the fields to change; a field the type does not have is refused",
			"look":             "GET /api/look?path=/t/note: the page as a screen reader gets it (title, landmarks, headings, controls with where they lead, live regions, components) and its structural problems; POST /api/look with {\"path\", \"method\", \"form\"} does what a person does and reads where they land, or with {\"component\", \"props\"} reads one component rendered from props. Controls say their value, whether they are checked or disabled, and which form they are in. Add scripts (?scripts=1, or \"scripts\": true) to read the page in a headless browser with its scripts run, and \"steps\" ([{\"press\": name}, {\"type\": words, \"into\": name}, {\"key\": \"Tab\"}, {\"wait\": ms}]) to do what a person does first: the answer adds scripts.did, scripts.focused, scripts.focus_order (Tab pressed for real, from the top) and scripts.errors. only (landmarks,headings,controls,live,components), kind and name narrow a long answer; problems always stay",
			"errors":           "every answer under /api is JSON, errors too: {\"error\": {\"code\", \"message\", \"fields\"}} with code bad_request, invalid (422, fields says which) or not_found",
			"delete":           "DELETE /api/{type}/{id}",
			"chat":             "POST /api/chat with {\"message\": \"...\"}",
			"clear":            "POST /api/chat/clear: clears all messages from the current conversation, leaving the canvas and other chats untouched",
			"mcp":              "sameway mcp (Model Context Protocol over stdio: the tools listed here, plus describe and get_record)",
			"html_list":        "GET /t/{type}",
			"html_detail":      "GET /t/{type}/{id}",
			"act":              "POST /act/{id} with from=<path to return to> runs one of the person's actions (a record of type action: a webhook, a command on this machine accepted once by the person, an arrangement, or a message to the assistant); POST /api/act/{id} runs it for an agent and answers with the result, or with waiting_for when the person's acceptance is needed first. A button block with action set to the id is the same press on the canvas",
			"hook":             "POST /hook/{word} runs the action whose trigger field is that word, from anywhere, and answers with the result: how something outside presses a button here. Actions also run on their own with every, at and on",
			"files":            "POST /t/file/upload (multipart, field file, optional title and from) adds a file: the original is kept and served at GET /files/{id}, and its contents are read into the record's text field as Markdown, at once for text, Markdown, CSV, HTML, EPUB, Word, Excel, PowerPoint and born-digital PDF, or by the converter named in workspace.yaml files.convert for that extension (status says converting until then). POST /api/file/upload with {\"title\": \"...\", \"filename\": \"...\", \"content\": \"<base64>\"} creates a complete record from an agent. Agents can also POST without content to create a stub file record, and POST /chat with a file part attaches it to the message and gives the assistant its text",
			"search":           "GET /api/search?q=words: every record of every content type and every block whose words match, with a snippet and its page; the same search a person has at /search and the assistant has as the search tool",
			"undo":             "POST /activity/{id}/undo with from=<path to return to>: reverses one activity entry for a person; agents call the undo_change tool. An entry that can be undone carries before, the thing as it was",
			"content":          "content/<type>/<id>.md in the workspace is every record as Markdown with front matter, written as it changes; share the folder with git. sameway import reads it back after a pull, sameway export rewrites it",
			"html_props":       "POST /t/{type}/{id}/props, form-encoded with each field named prop-<field>: the inline editor's route, which answers with the page rather than JSON. A field named html-<field> is the rich editor's HTML, turned into Markdown on the way in, with level-<field> the heading level it was shown at",
			"mcp_http":         "POST /mcp: the MCP server over HTTP, one JSON-RPC message or a batch per request, for a client elsewhere; needs Authorization: Bearer <token>, the token being the environment variable named by mcp.token_env in workspace.yaml (SAMEWAY_MCP_TOKEN), and is off without it. The same server over stdio is `sameway mcp`; `sameway connect <tool>` writes the configuration for a local client",
			"types":            "POST /api/types with {name, description, title, fields: [{name, type, description, values, to, required, default}]} makes a content type while the workspace runs: its schema file, its table, its pages; POST /api/types/{type}/fields with one field adds a property to a type. The assistant has the same as add_type and add_field",
			"prose":            "POST /api/prose with {\"markdown\": \"...\", \"level\": 3} gives {\"html\"}: the page's rendering of that Markdown; with {\"html\": \"...\", \"level\": 3} gives {\"markdown\"}: the same words back as Markdown, headings at the level the source had",
			"canvas_props":     "POST /canvas/{block-id}/props, the same form for a block on the canvas",
			"css":              "GET /design/sameway.css",
			"canvas_blocks":    "GET /api/block",
			"proposal_accept":  "POST /proposal/{id}/accept with from=<path to return to>: accepts a pending proposal so the assistant's change goes through; after accepting the person is returned to the page they were on",
			"proposal_dismiss": "POST /proposal/{id}/dismiss with from=<path to return to>: rejects a pending proposal and does not make any changes; after dismissing the person is returned to the page they were on",
			"message":          "GET /api/message",
		},
	}
	if a.Chat.ProviderErr != nil {
		d.LLM.Problem = a.Chat.ProviderErr.Error()
	}
	for _, t := range a.Types.Types {
		n, _ := a.Store.Count(t.Name)
		d.Types = append(d.Types, DescribedType{Name: t.Name, Description: t.Description, Internal: t.Internal, Title: t.Title, Fields: t.Fields, Schema: t.JSONSchema(), Count: n})
	}
	for _, c := range a.Registry.Components() {
		d.Components = append(d.Components, DescribedComponent{Name: c.Manifest.Name, Source: c.Source, Description: c.Manifest.Description, Use: c.Manifest.Use, Props: c.Manifest.Props, A11y: c.Manifest.A11y, Machine: c.Manifest.Machine})
	}
	d.Arrangements = a.Registry.Arrangements()
	for _, t := range a.Chat.Tools() {
		d.Tools = append(d.Tools, DescribedTool{Name: t.Name, Description: t.Description, Schema: t.Schema})
	}
	return d
}
