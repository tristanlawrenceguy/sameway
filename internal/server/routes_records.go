package server

// recordRoutes are the records' pages and what is done from them: lists,
// a record, its files, exports, and the pages that gather records
// (templates, sharing).
var recordRoutes = []route{
	{pattern: "GET /t/{type}", handle: (*Server).listPage, access: people, public: true},
	{pattern: "POST /t/{type}/add", handle: (*Server).addRecord, access: people, tool: "create_record", reach: inward},
	{pattern: "GET /t/{type}/{id}", handle: (*Server).detailPage, access: people, public: true},
	{pattern: "GET /t/{type}/{id}/whole", handle: (*Server).wholePage, access: people, public: true},
	{pattern: "POST /t/{type}/{id}/parts/move", handle: (*Server).moveParts, access: people, tool: "organise_writing", reach: inward},
	{pattern: "POST /t/{type}/{id}/delete", handle: (*Server).deleteForm, access: people, persons: "a record goes when a person deletes it, or when its making is undone", reach: inward},
	{pattern: "POST /t/{type}/{id}/keep-one", handle: (*Server).keepOneForm, access: people, tool: "keep_one", reach: inward},
	{pattern: "POST /t/{type}/{id}/discard", handle: (*Server).discard, access: people, tool: "undo_change", reach: inward},
	{pattern: "POST /t/{type}/{id}/props", handle: (*Server).recordProps, access: people, tool: "update_record", reach: inward},
	// A file and what is read from it are public with the page that shows
	// it; its sound, copied out for writing it down, is not.
	{pattern: "POST /act/{id}", handle: (*Server).act, access: people, tool: "run_action", reach: outward},
	{pattern: "POST /suggestions/{id}/{answer}", handle: (*Server).suggestionAnswer, access: people, persons: "a suggestion to their writing is theirs to accept or decline, never the one who suggested it", reach: inward},
	{pattern: "POST /suggestions/accept-all", handle: (*Server).suggestionsAcceptAll, access: people, persons: "the same, for every fix or formatting change at once", reach: inward},
	{pattern: "GET /templates", handle: (*Server).templatesPage, access: people},
	{pattern: "POST /templates/use", handle: (*Server).templateUse, access: people, tool: "add_arrangement", reach: inward},
	{pattern: "GET /share", handle: (*Server).sharePage, access: people},
	{pattern: "POST /share", handle: (*Server).shareSave, access: people, tool: "create_record", reach: outward},
}
