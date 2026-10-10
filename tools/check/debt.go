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
	"internal/cli.ctx.openCmd":             114,
	"internal/convert.pptx":                91,
	"internal/discover.Browse":             115,
	"internal/llm.OpenAI.Stream":           86,
	"internal/look.problems":               102,
	"internal/render.chartShapeAt":         90,
	"internal/schema.Set.Complete":         92,
	"internal/schema.coerce":               104,
	"internal/server.Server.apiList":       83,
	"internal/server.Server.detailPage":    126,
	"internal/server.Server.narrowLog":     112,
	"internal/server.Server.shareSave":     81,
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

// storeWrites are the functions that write the store directly rather than
// through records.Apply, ApplyOps or WriteAs, each with why (writes.go).
// Keyed by file and function. Like the lists above it only goes down.
var storeWrites = map[string]string{
	"internal/app/schema_change.go App.RemoveType":      "a type deleted drops its whole table in one statement, logged as the schema change; its records go with their type, not one by one",
	"internal/content/mirror.go Mirror.Import":          "a record read back from its file keeps the created and updated times the file says, which Apply cannot set; the caller logs the whole import as one change with its ops (cli/portable.go)",
	"internal/devices/devices.go Apply":                 "devices cannot import records (records imports workspace, which imports devices); to take a writer from the caller",
	"internal/ingest/calendar_sync.go SyncCalendar":     "to move onto records.ApplyOps: written before ingest went through records",
	"internal/ingest/import.go Import":                  "to move onto records: written before ingest went through records; its callers log the import with its ops",
	"internal/ingest/people.go linker.find":             "to move onto records.ApplyOps: a person made while importing, written before ingest went through records",
	"internal/server/servertest/fixtures.go SeedTasks":  "a test fixture seeding a fresh workspace; nothing to log or undo",
	"internal/chat/access.go Service.letIn":             "person access, logged by hand: to move after the invite dive, which owns this file now",
	"internal/chat/invite.go Service.GiveAccess":        "person access, logged by hand as human: to move after the invite dive, which owns this file now",
	"internal/chat/suggest_records.go Service.Suggest":  "a suggestion's question: to move after the tagging dive, which owns this file now",
	"internal/chat/schedule.go Service.RunDue":          "an action's last run stamped: to move after the background job runner lands, which owns this file now",
	"internal/chat/actions.go Service.show":             "to move onto records: the action keeps its answer's block",
	"internal/chat/arrange.go Service.arrange":          "to move onto records.Apply: an arrangement written block by block",
	"internal/chat/arrange.go Service.putBack":          "to move onto records.Apply: an arrangement put back by hand",
	"internal/chat/clear.go Service.clearCanvas":        "to move onto records.Apply: a tab cleared block by block",
	"internal/chat/command.go Service.acceptAction":     "to move onto records: a command accepted",
	"internal/chat/consent.go Service.ask":              "to move onto records: a question's words",
	"internal/chat/conversations.go Service.Current":    "to move onto records: the chat's own bookkeeping",
	"internal/chat/conversations.go Service.DeleteChat": "to move onto records: a chat deleted",
	"internal/chat/conversations.go Service.NewChat":    "to move onto records: the chat's own bookkeeping",
	"internal/chat/conversations.go Service.OpenChat":   "to move onto records: the chat's own bookkeeping",
	"internal/chat/conversations.go Service.clearing":   "to move onto records: a chat cleared",
	"internal/chat/conversations.go Service.message":    "to move onto records: the chat's own bookkeeping",
	"internal/chat/propose.go Service.Accept":           "to move onto records: a question answered",
	"internal/chat/propose.go Service.Dismiss":          "to move onto records: a question answered",
	"internal/chat/propose.go Service.propose":          "to move onto records: a question asked",
	"internal/chat/reshape.go Service.Instead":          "to move onto records: a question answered",
	"internal/chat/reshape.go Service.askDelete":        "to move onto records: a question's other answer",
	"internal/chat/suggest.go AcceptSuggestions":        "to move onto records: a suggestion taken",
	"internal/chat/suggest.go DeclineSuggestion":        "to move onto records: a suggestion set aside",
	"internal/chat/suggest.go Service.suggestEdits":     "to move onto records: a suggestion made",
}
