package app

import (
	"fmt"
	"sort"
	"strings"
)

// Index is what GET /api/describe and the MCP describe tool answer with
// when nothing more is asked: a few kilobytes that say what this is and how
// to build on it, with the address of everything else. The whole
// description is hundreds of kilobytes; an agent handed that read the
// routes last, never found how to add a block, and wrote an HTML page of
// its own instead, and over MCP it was more than the client would take.
// Each list is of lines, "name: what", so the order holds and it stays small.
type Index struct {
	Workspace  string   `json:"workspace"`
	Dir        string   `json:"dir"`
	About      string   `json:"about"`
	Build      []string `json:"build"`
	Routes     []string `json:"routes"`
	Components []string `json:"components"`
	Types      []string `json:"types"`
	More       []string `json:"more"`
}

const indexAbout = `Sameway is a person's own workspace: records of content types (notes, tasks, habits, people and the rest), each on its page at /t/{type}/{id}, and pages of blocks (Home at /, more tabs at /c/{canvas}), each block a component from the design system showing those records. You build for the person by adding blocks through the API, never by writing HTML of your own. Every answer under /api is JSON; send Authorization: Bearer <key> when you were given one.`

// indexBuild is how to build, in the order it is done.
var indexBuild = []string{
	`1. Find what there is: GET /api/{type} lists a type's records (?where=done=false&order=due narrows and sorts), GET /api/search?q=words searches everything, GET /api/canvas lists the tabs beside Home, GET /api/block the blocks. Use only records that exist: if the person has none, say so or ask; never make up records, and never log entries to test.`,
	`2. Pick a component from components below; GET /api/describe/components/{name} gives the props it takes and an example to start from.`,
	`3. Add a block: POST /api/block with {"component", "props", "canvas", "span", "position"} (block_add below). It is checked before it is written; the answer's shows says in words what the block shows, and layout how the page reads now. Read both.`,
	`4. Lay out the whole page, not only what you added: POST /api/arrange (block_arrange below) puts every block in reading order with its width in one change: most important first, related things together, rows of twelve filled, headings in order.`,
	`5. Change a block with PATCH /api/block/{id}, take it away with DELETE /api/block/{id}. GET /api/look?path=/ reads the page as a screen reader gets it.`,
	`Records: POST /api/{type} makes one (its fields: /api/describe/types/{type}), PATCH /api/{type}/{id} changes one. A page's form answers in JSON only when sent with Accept: application/json (actions below); without it the answer is a page.`,
}

// indexFirst are the routes an agent building needs, in the order it needs
// them, said whole; the rest follow by name, in a line each.
var indexFirst = []string{"block_add", "block_update", "block_arrange", "block_remove", "blocks", "errors"}

// indexShort says the routes most used next in a line, where their own
// first words would not.
var indexShort = map[string]string{
	"list":     "GET /api/{type}: a type's records; ?where=field=value (repeatable), ?order=field or -field, ?page=, ?fields=",
	"get":      "GET /api/{type}/{id}: one record, every field, with its title",
	"create":   "POST /api/{type} with a JSON object of fields",
	"update":   "PATCH /api/{type}/{id} with the fields to change; If-Match with updated_at refuses a change made from an older version",
	"search":   "GET /api/search?q=words: every record and block that matches, counted by kind; &type= narrows",
	"look":     "GET /api/look?path=/: a page as a screen reader gets it, and its problems; on a page of blocks, measured is each block's height and any scroll inside it as the person's own browser drew it; POST does what a person does",
	"actions":  "POST a page's form with Accept: application/json and the answer is JSON: {ok, title, text, location, undo, fields}",
	"keys":     "Authorization: Bearer sw_... is an agent key the owner made (sameway agent add); the log names you by it",
	"describe": "GET /api/describe: this index; /api/describe/{part} and /api/describe/{part}/{name} one part or one thing; ?full=1 everything",
}

// Index cuts the description down to the index.
func (d Description) Index() Index {
	x := Index{Workspace: d.Workspace, Dir: d.Dir, About: indexAbout, Build: indexBuild, More: []string{
		"one route whole: GET /api/describe/routes/{key}; all: /api/describe/routes",
		"one component: GET /api/describe/components/{name} (props and an example; ?full=1 adds its accessibility and machine contract)",
		"one type: GET /api/describe/types/{name}; the assistant's tools: /api/describe/tools; whole pages of blocks: /api/describe/arrangements",
		"everything: GET /api/describe?full=1 (hundreds of kilobytes; read a part instead)",
		"over MCP: the describe tool, with name (meter, task, block_add) or part",
	}}
	seen := map[string]bool{}
	for _, k := range indexFirst {
		if v, ok := d.Routes[k]; ok {
			x.Routes = append(x.Routes, k+": "+v)
			seen[k] = true
		}
	}
	var rest []string
	for k := range d.Routes {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Slice(rest, func(i, j int) bool {
		_, si := indexShort[rest[i]]
		_, sj := indexShort[rest[j]]
		if si != sj {
			return si
		}
		return rest[i] < rest[j]
	})
	for _, k := range rest {
		how, ok := indexShort[k]
		if !ok {
			how = brief(d.Routes[k], 90)
		}
		x.Routes = append(x.Routes, k+": "+how)
	}
	for _, c := range d.Components {
		x.Components = append(x.Components, c.Name+": "+brief(c.Description, 90))
	}
	for _, t := range d.Types {
		if t.Internal {
			continue
		}
		var fields []string
		for _, f := range t.Fields {
			fields = append(fields, f.Name)
		}
		x.Types = append(x.Types, fmt.Sprintf("%s (%d): %s", t.Name, t.Count, strings.Join(fields, ", ")))
	}
	return x
}

// brief is the start of a description, up to its first sentence and at
// most n bytes, cut at a word; the rest is at its own address.
func brief(s string, n int) string {
	if i := strings.Index(s, ". "); i > 0 {
		s = s[:i+1]
	}
	if len(s) <= n {
		return s
	}
	cut := strings.LastIndex(s[:n], " ")
	if cut <= 0 {
		cut = n
	}
	return strings.TrimRight(s[:cut], ",;:") + "..."
}
