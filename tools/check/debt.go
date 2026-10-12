package main

// Debt the checks know about. These lists say where the code is harder to
// read than the rules allow today; they are information, not exceptions to
// copy. The lists only go down: lower a length when a function gets
// shorter, and take an entry out when it meets the rule. CI runs
// `go run ./tools/check -base <base>` and fails a change that adds an entry
// or raises a length here; shorten or split the code instead.

// longFuncs are non-test functions over MaxFuncLines, keyed by package
// directory and Name or Recv.Name, with their length when the cap came in
// (2026-10-08, updated at the merge with main that day). check fails if one grows.
var longFuncs = map[string]int{
	"internal/app.Description.part":        95,
	"internal/chat.Service.sendTurn":       131,
	"internal/chat.Service.writeUpMeeting": 84,
	"internal/cli.ctx.contentCmd":          106,
	"internal/convert.pptx":                91,
	"internal/discover.Browse":             115,
	"internal/llm.OpenAI.Stream":           86,
	"internal/look.problems":               102,
	"internal/render.chartShapeAt":         90,
	"internal/schema.Set.Complete":         92,
	"internal/schema.coerce":               104,
	"internal/server.Server.apiList":       83,
	"internal/server.Server.detailPage":    120,
	"internal/server.Server.narrowLog":     112,
	"internal/soundtrack.readMKV":          83,
	"internal/speech.SplitWAV":             110,
	"internal/track.Summarise":             87,
}

// splitByName are files named for being split by size (*_more, *_extra,
// *_helpers) from before the rule. Renaming one for what it does, or
// folding it back where it belongs, takes it out of this list.
var splitByName = map[string]bool{
	"internal/server/detail_attributes_extra_test.go": true,
	"internal/server/outcome_helpers_test.go":         true,
	"internal/server/record_props_extra_test.go":      true,
	"internal/server/said_test_helpers_test.go":       true,
}

// wallClock are files under clockDirs that read time.Now() where the app's
// clock belongs, with how many times, when the rule came in (2026-10-10).
// Mail in is being moved onto the background runner elsewhere at the
// time. Move each to the app's clock, or mark a use that must be real
// time `// wall clock: <reason>`, and lower or remove its entry.
var wallClock = map[string]int{
	"internal/server/mail_in.go": 2,
}

// storeWrites are the functions that write the store directly rather than
// through records.Apply, ApplyOps or WriteAs, each with why (writes.go).
// Keyed by file and function. Like the lists above it only goes down.
var storeWrites = map[string]string{
	"internal/app/schema_change.go App.RemoveType":     "a type deleted drops its whole table in one statement, logged as the schema change; its records go with their type, not one by one",
	"internal/content/mirror.go Mirror.Import":         "a record read back from its file keeps the created and updated times the file says, which Apply cannot set; the caller logs the whole import as one change with its ops (cli/portable.go)",
	"internal/server/servertest/fixtures.go SeedTasks": "a test fixture seeding a fresh workspace; nothing to log or undo",
}
