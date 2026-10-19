package app

import (
	"fmt"
	"sort"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Index is what GET /api/describe and the MCP describe tool answer with
// when nothing more is asked: a few kilobytes that say what this is and how
// to build on it, with the address of everything else. The whole
// description is hundreds of kilobytes; an agent handed that read the
// routes last, never found how to add a block, and wrote an HTML page of
// its own instead, and over MCP it was more than the client would take.
// It says whole only what building takes (how, adding a block, laying out
// a page, what an error says) and the routes most used in a line each;
// every other route, component and type is a name, said whole at its own
// address. Lines for all of them made it 16 KB, a client's whole limit.
type Index struct {
	Workspace  string   `json:"workspace"`
	Dir        string   `json:"dir"`
	About      string   `json:"about"`
	Build      []string `json:"build"`
	Routes     []string `json:"routes"`
	Components string   `json:"components"`
	Types      string   `json:"types"`
	More       []string `json:"more"`
}

const indexAbout = `Sameway is a person's own workspace: records of content types (notes, tasks, habits, people and the rest), each on its page at /t/{type}/{id}, and pages of blocks (Home at /, more tabs at /c/{canvas}), each block a component from the design system showing those records. Most of what a person asks is about their records: find, make and change them (Records below); their Today page already shows what is due. Add blocks only when they ask to see something on a page, and never write HTML of your own. Every answer under /api is JSON; send Authorization: Bearer <key> when you were given one.`

// indexBuild is how to build, in the order it is done.
var indexBuild = []string{
	`1. Find what there is: GET /api/{type} lists a type's records (?where=done=false&where=due<today narrows to what is late, created_at>=today to what was added today, updated_at>=-7d to the last week; order=due sorts), GET /api/search?q=words searches everything, GET /api/canvas lists the tabs beside Home, GET /api/block the blocks. Use only records that exist: if the person has none, say so or ask; never make up records, and never log entries to test.`,
	`2. Pick a component from components below; GET /api/describe/components/{name} gives the props it takes and an example to start from.`,
	`3. Add a block: POST /api/block with {"component", "props", "canvas", "span", "position"} (block_add below). It is checked before it is written; the answer's shows says in words what the block shows, and layout how the page reads now. Read both.`,
	`4. Lay out the whole page, not only what you added: POST /api/arrange (block_arrange below) puts every block in reading order with its width in one change: most important first, related things together, rows of twelve filled, headings in order.`,
	`5. Change a block with PATCH /api/block/{id}, take it away with DELETE /api/block/{id}. GET /api/look?path=/ reads the page as a screen reader gets it.`,
	`The day: GET /api/today is what the person's Today page holds, as JSON: what is late, due and on, and on a long day the three to start with; done_today is what got done; ?days=7 adds the week ahead, and the person's own page for planning the week is /review (done this week, overdue, the week ahead). Records: POST /api/{type} makes one (its fields: /api/describe/types/{type}), or several from a JSON array as one change; PATCH /api/{type}/{id} changes one, and PATCH /api/{type} with ids the same change to several. A page's form answers in JSON only when sent with Accept: application/json (actions below); without it the answer is a page.`,
}

// indexFirst are the routes an agent building needs, in the order it needs
// them, said whole; the rest follow by name, in a line each.
var indexFirst = []string{"block_add", "block_arrange", "errors"}

// indexShort says the routes most used next in a line, where their own
// first words would not.
var indexShort = map[string]string{
	"list":     "GET /api/{type}: a type's records; ?where=field=value (field~part for a part of it, such as name~Lena), due<today, created_at>=today, updated_at>=-7d (repeatable, all hold), ?order=field or -field, ?limit= and ?page= (the answer then says total, pages and next), ?fields=",
	"get":      "GET /api/{type}/{id}: one record, every field, with its title",
	"create":   "POST /api/{type} with a JSON object of fields, or a JSON array of them to make several as one change (all or none, one Undo); PATCH /api/{type} with {\"ids\": [...], \"fields\": {...}} makes the same change to several",
	"update":   "PATCH /api/{type}/{id} with the fields to change, at the top level ({\"due\": \"friday\"}); several to the same: PATCH /api/{type} with ids; If-Match with updated_at refuses a change made from an older version",
	"changes":  "GET /api/changes?since=today (yesterday, -2w): everything that changed since, in one read, oldest first, with a cursor for next time; GET /api/activity?where=created_at>=today&where=actor=assistant is the log narrowed to who and when",
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
	var short, names []string
	for k := range d.Routes {
		switch {
		case seen[k]:
		case indexShort[k] != "":
			short = append(short, k+": "+indexShort[k])
		default:
			names = append(names, k)
		}
	}
	sort.Strings(short)
	sort.Strings(names)
	x.Routes = append(x.Routes, short...)
	x.Routes = append(x.Routes, "and by name, each whole at GET /api/describe/routes/{name}: "+schema.Names(names))
	var comps, types []string
	for _, c := range d.Components {
		comps = append(comps, c.Name)
	}
	for _, t := range d.Types {
		if !t.Internal {
			types = append(types, fmt.Sprintf("%s (%d)", t.Name, t.Count))
		}
	}
	x.Components = schema.Names(comps)
	x.Types = schema.Names(types) + "; how many each has in brackets"
	return x
}
