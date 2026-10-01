package mcp

// What each tool is like, as MCP's annotations say it, so a client can ask
// its person before what changes or reaches outside, and run what only
// reads without asking: a title a person reads, whether it only reads,
// whether what it changes is taken away (removed, cleared, deleted, even
// though Undo puts it back), whether calling it twice is the same as once,
// and whether it reaches beyond this workspace. Every tool listed must be
// here (TestEveryToolSaysWhatItIs), and every tool said to read only is
// called and the workspace checked unchanged.

type traits struct {
	title                                        string
	readOnly, destructive, idempotent, openWorld bool
}

var toolTraits = map[string]traits{
	// Reading.
	"describe":     {title: "Describe the workspace", readOnly: true, idempotent: true},
	"look":         {title: "Read a page", readOnly: true, idempotent: true},
	"find_records": {title: "Find records", readOnly: true, idempotent: true},
	"get_record":   {title: "Read a record", readOnly: true, idempotent: true},
	"search":       {title: "Search everything", readOnly: true, idempotent: true},

	// Content.
	"create_record":    {title: "Make a record"},
	"update_record":    {title: "Change a record", idempotent: true},
	"import_records":   {title: "Bring records in from a file"},
	"write_up_meeting": {title: "Write up a meeting"},
	"add_type":         {title: "Add a kind of record"},
	"add_field":        {title: "Add a field to a kind"},
	"change_field":     {title: "Change or remove a field", destructive: true},

	// The canvas.
	"add_component":    {title: "Add a block"},
	"update_component": {title: "Change a block", idempotent: true},
	"remove_component": {title: "Remove a block", destructive: true, idempotent: true},
	"clear_canvas":     {title: "Clear the page", destructive: true, idempotent: true},
	"create_canvas":    {title: "Make a tab"},
	"remove_canvas":    {title: "Remove a tab", destructive: true, idempotent: true},
	"arrange_canvas":   {title: "Lay a tab out", idempotent: true},
	"add_arrangement":  {title: "Add a ready-made arrangement"},
	"propose_change":   {title: "Ask the person before a change"},

	// Taking back, settings, the conversation.
	"undo_change":        {title: "Undo a change"},
	"set_setting":        {title: "Change a setting", idempotent: true},
	"clear_conversation": {title: "Clear the conversation", destructive: true, idempotent: true},

	// Beyond the records.
	"run_action":     {title: "Run an action", openWorld: true},
	"update_sameway": {title: "Update Sameway", openWorld: true},
	"let_in":         {title: "Let a person in"},
	"write_down":     {title: "Write a recording down"},

	// Workspaces and agents, the owner's.
	"add_workspace":     {title: "Make a workspace"},
	"open_workspace":    {title: "Open a workspace", idempotent: true},
	"restore_workspace": {title: "Restore a deleted workspace"},
	"take_agent_away":   {title: "Take an agent's key away", destructive: true, idempotent: true},
}

// annotations are a tool's traits as MCP spells them, or nil for a tool
// not in the table.
func annotations(name string) map[string]any {
	tr, ok := toolTraits[name]
	if !ok {
		return nil
	}
	return map[string]any{"title": tr.title, "readOnlyHint": tr.readOnly, "destructiveHint": tr.destructive,
		"idempotentHint": tr.idempotent, "openWorldHint": tr.openWorld}
}
