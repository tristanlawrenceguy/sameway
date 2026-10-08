package main

// Debt the checks know about. These lists say where the code is harder to
// read than the rules allow today; they are information, not exceptions to
// copy. An entry may only shrink: lower a length when a function gets
// shorter, and take an entry out when it meets the rule. Adding one is for
// moving an entry with its function (a package rename), not for new code.

// longFuncs are non-test functions over MaxFuncLines, keyed by package
// directory and Name or Recv.Name, with their length when the cap came in
// (2026-10-08, updated at the merge with main that day). check fails if one grows.
var longFuncs = map[string]int{
	"internal/app.Description.part":              95,
	"internal/chat.Service.sendTurn":             122,
	"internal/chat.Service.writeUpMeeting":       84,
	"internal/chat.describe":                     82,
	"internal/chat.toolHandlers":                 105,
	"internal/cli.ctx.contentCmd":                106,
	"internal/cli.ctx.openCmd":                   112,
	"internal/convert.pptx":                      91,
	"internal/discover.Browse":                   115,
	"internal/llm.OpenAI.Stream":                 86,
	"internal/look.problems":                     102,
	"internal/render.chartShapeAt":               90,
	"internal/schema.Set.Complete":               92,
	"internal/schema.coerce":                     104,
	"internal/server.Server.apiList":             83,
	"internal/server.Server.detailPage":          129,
	"internal/server.Server.focusPage":           85,
	"internal/server.Server.listPage":            83,
	"internal/server.Server.narrowLog":           112,
	"internal/server.Server.resolveCalendarAt":   105,
	"internal/server.Server.resolveChart":        92,
	"internal/server.Server.resolveCollectionAt": 110,
	"internal/server.Server.routes":              110,
	"internal/server.Server.searchPage":          84,
	"internal/server.Server.shareSave":           81,
	"internal/soundtrack.readMKV":                83,
	"internal/speech.SplitWAV":                   110,
	"internal/track.Summarise":                   87,
}

// splitByName are files named for being split by size (*_more, *_extra,
// *_helpers) from before the rule. Renaming one for what it does, or
// folding it back where it belongs, takes it out of this list.
var splitByName = map[string]bool{
	"internal/server/views_helpers.go":                true,
	"internal/server/detail_attributes_extra_test.go": true,
	"internal/server/outcome_helpers_test.go":         true,
	"internal/server/record_props_extra_test.go":      true,
	"internal/server/said_test_helpers_test.go":       true,
}
