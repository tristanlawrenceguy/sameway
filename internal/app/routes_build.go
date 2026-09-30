package app

// buildRoutes are the routes for building a page: adding, changing and
// taking away blocks. A block is a record of the block type, which is
// internal (no page of its own), so the generic record routes never said
// it could be written; an agent over REST built nothing in five tasks for
// want of these lines, and wrote HTML of its own instead.
var buildRoutes = map[string]string{
	"block_add":     `POST /api/block with {"component": "<name>", "props": {...}} adds a block to a page, shown at once to whoever has it open. Also: "canvas" (the tab, as a canvas id from GET /api/canvas; "" or left out is Home), "span" (width in twelfths: 12 full, 6 half, 4 a third; 6 when left out), "position" (a number, lower first; 0 when left out), "region" (main, left, right, header or footer), "frame" (card or bare), "tone" (none, accent, success, warning, danger, info), "size" (full, compact, icon). props must fit the component's schema: GET /api/describe/components/{name} gives it with an example. The block is checked before it is written, the way the in-app assistant's are: a component there is not, props that do not fit, or props naming a type, field or record the workspace does not have are refused with 422 and nothing written, {"error": {"code": "invalid" or "cannot_show", "message": what is wrong and what there is instead}}; fix what it says and send again. The answer is 201 with the block (id, fields) and shows: what the block shows, in words, such as "3 tasks, not done, by due". Read shows: it is how you know the block shows what you meant`,
	"block_update":  `PATCH /api/block/{id} with only what changes: props (the whole props object, which replaces the old), span, position, canvas, region, frame, tone or size. A change to component or props is checked as block_add is and answers with shows; a move or a restyle is not checked`,
	"block_arrange": `POST /api/arrange with {"canvas": "<canvas id; empty is Home>", "blocks": [{"id", "span", "region", "frame", "size"}, ...]} lays out a whole tab in one change: every block on it (header and footer ones may be left out) in the order it should be read, each with the width and place it should have. One Undo takes it all back. Refused with 422 and nothing changed when a block is missing or listed twice, or a heading would skip a level. Every block write (add, change, remove, arrange) answers with layout: how the tab reads now, row by row, what to look at (holes, a tall block beside short ones, headings out of order or saying the same, related blocks apart, what is late below what is not) and an arrangement that fixes it. Read it and arrange what was there, not only what you added`,
	"block_remove":  `DELETE /api/block/{id} takes a block off its page; it can be undone from /activity`,
	"blocks":        `GET /api/block lists every block with its component, props and place; ?where=canvas=<id> the blocks on one tab (?where=canvas= for Home). GET /api/canvas lists the tabs beside Home; POST /api/canvas with {"name"} adds one, shown at /c/{id}`,
}

// withBuild adds the build routes to the rest.
func withBuild(routes map[string]string) map[string]string {
	for k, v := range buildRoutes {
		routes[k] = v
	}
	return routes
}
