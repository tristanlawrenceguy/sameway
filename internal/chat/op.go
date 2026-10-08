package chat

import (
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// What the assistant can do is one list, the registry: each operation
// once, with everything said about it beside its code. The model's tool
// list, MCP's tools/list and /api/describe are all read from it, so an op
// added or changed shows up on every surface at once, and nothing has to
// be kept in step with anything by hand.

// Op is one thing the assistant can do.
type Op struct {
	llm.Tool        // its name, what it does and what it takes, as the model reads them
	Title    string // what it does, as a person reads it: "Make a record"
	Traits
	Access Access
	// Offered says whether this workspace offers it, and fills in what its
	// schema names from the workspace (its content types, its kinds) on
	// the copy it is given; nil is always.
	Offered func(s *Service, t *llm.Tool) bool
}

// Traits are what an op is like, as MCP's annotations say it, so a client
// can ask its person before what changes or reaches outside, and run what
// only reads without asking: whether it only reads, whether what it
// changes is taken away (removed, cleared, deleted, even though Undo puts
// it back), whether calling it twice is the same as once, and whether it
// reaches beyond this workspace.
type Traits struct{ ReadOnly, Destructive, Idempotent, OpenWorld bool }

// Access is who may have an op done: the pages, the assistant, MCP and
// the API all go by it.
type Access int

const (
	ForEditors Access = iota // whoever may change the workspace
	ForViewers               // whoever may look at it: the op only reads
	ForOwner                 // the workspace's owner alone
)

// registry is every op, in the order the model is given them (the order
// matters to a model here, which reads again only from where a turn's
// tools differ: warm.go). It is put together in init from the lists
// beside each op's code, because what an op does can lead back to
// running another, and a variable cannot refer to itself.
var registry []Op

var opIndex = map[string]int{}

func init() {
	for _, list := range [][]Op{blockOps, {undoOp, searchOp, actionOp, updateOp, arrangementOp, settingOp},
		recordOps, meetingOps, organiseOps, suggestOps, recordingOps, canvasOps, shapeOps, lookOps, accessOps, homeOps} {
		registry = append(registry, list...)
	}
	for i, o := range registry {
		if _, twice := opIndex[o.Name]; twice {
			panic("two ops are called " + o.Name)
		}
		opIndex[o.Name] = i
	}
}

// OpFor is the op of that name.
func OpFor(name string) (Op, bool) {
	i, ok := opIndex[name]
	if !ok {
		return Op{}, false
	}
	return registry[i], true
}

// allTools is every op this workspace offers, as the model reads it.
func (s *Service) allTools() []llm.Tool {
	var out []llm.Tool
	for _, o := range registry {
		t := o.Tool
		if o.Offered == nil || o.Offered(s, &t) {
			out = append(out, t)
		}
	}
	return out
}

// has offers an op where the workspace has every one of these content
// types.
func has(types ...string) func(*Service, *llm.Tool) bool {
	return func(s *Service, _ *llm.Tool) bool { return s.hasTypes(types...) }
}

func (s *Service) hasTypes(types ...string) bool {
	for _, t := range types {
		if _, ok := s.Store.Types().Get(t); !ok {
			return false
		}
	}
	return true
}

// withProp sets one property of a tool's schema to what the workspace has
// now, on a copy, so the registry's own stays as it was written.
func withProp(t *llm.Tool, name string, prop map[string]any) {
	schema := map[string]any{}
	for k, v := range t.Schema {
		schema[k] = v
	}
	props := map[string]any{}
	if old, ok := t.Schema["properties"].(map[string]any); ok {
		for k, v := range old {
			props[k] = v
		}
	}
	props[name] = prop
	schema["properties"] = props
	t.Schema = schema
}
