// Package app wires a workspace, its schema, store, components, and chat
// into one object shared by the CLI and the HTTP server.
package app

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// App is a loaded workspace ready to serve.
type App struct {
	Workspace *workspace.Workspace
	Types     *schema.Set
	Store     *store.Store
	Registry  *render.Registry
	Chat      *chat.Service
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
	a.Chat = &chat.Service{
		Store:        st,
		Registry:     reg,
		HistoryLimit: ws.Config.Chat.HistoryLimit,
		ExtraPrompt:  ws.Config.Chat.SystemPrompt,
	}
	a.Chat.Provider, a.Chat.ProviderErr = llm.New(ws.Config.LLM)
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
	if workspaceComponents != "" {
		if err := reg.LoadDir(workspaceComponents, "workspace"); err != nil {
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
	LLM        DescribedLLM         `json:"llm"`
	Types      []DescribedType      `json:"types"`
	Components []DescribedComponent `json:"components"`
	Routes     map[string]string    `json:"routes"`
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
	Name        string          `json:"name"`
	Source      string          `json:"source"`
	Description string          `json:"description"`
	Props       json.RawMessage `json:"props"`
	A11y        json.RawMessage `json:"a11y,omitempty"`
	Machine     json.RawMessage `json:"machine,omitempty"`
}

// Describe builds the description from live state.
func (a *App) Describe() Description {
	d := Description{
		Workspace: a.Workspace.Config.Name,
		LLM:       DescribedLLM{Provider: a.Workspace.Config.LLM.Provider, Model: a.Workspace.Config.LLM.Model, Ready: a.Chat.Provider != nil},
		Routes: map[string]string{
			"describe":      "GET /api/describe",
			"list":          "GET /api/{type}",
			"create":        "POST /api/{type} with a JSON object of fields",
			"get":           "GET /api/{type}/{id}",
			"update":        "PUT /api/{type}/{id} with a JSON object of fields to change",
			"delete":        "DELETE /api/{type}/{id}",
			"chat":          "POST /api/chat with {\"message\": \"...\"}",
			"html_list":     "GET /t/{type}",
			"html_detail":   "GET /t/{type}/{id}",
			"html_new":      "GET /t/{type}/new",
			"css":           "GET /design/sameway.css",
			"canvas_blocks": "GET /api/block",
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
		d.Components = append(d.Components, DescribedComponent{Name: c.Manifest.Name, Source: c.Source, Description: c.Manifest.Description, Props: c.Manifest.Props, A11y: c.Manifest.A11y, Machine: c.Manifest.Machine})
	}
	return d
}
