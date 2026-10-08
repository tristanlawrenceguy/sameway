package server

// apiRoutes are the JSON API, for agents: what pages do, as JSON. A page
// action of the API's is no person's and no tool's of its own; it is the
// tool, or the record, it names.
var apiRoutes = []route{
	{pattern: "GET /api/describe", handle: (*Server).apiDescribe, access: people},
	{pattern: "GET /api/describe/{part}", handle: (*Server).apiDescribePart, access: people},
	{pattern: "GET /api/describe/{part}/{name}", handle: (*Server).apiDescribePart, access: people},
	{pattern: "GET /api/search", handle: (*Server).apiSearch, access: people},
	{pattern: "GET /api/look", handle: (*Server).apiLook, access: owner},
	{pattern: "POST /api/look", handle: (*Server).apiLook, access: owner},
	{pattern: "POST /api/prose", handle: (*Server).apiProse, access: people},
	{pattern: "POST /api/types", handle: (*Server).apiAddType, access: people},
	{pattern: "POST /api/types/{type}/fields", handle: (*Server).apiAddField, access: people},
	{pattern: "POST /api/act/{id}", handle: (*Server).apiAct, access: people},
	{pattern: "POST /hook/{token}", handle: (*Server).hook, access: people},
	{pattern: "POST /api/chat", handle: (*Server).apiChat, access: people},
	{pattern: "POST /api/chat/clear", handle: (*Server).apiChatClear, access: people},
	{pattern: "POST /api/file/upload", handle: (*Server).apiFileUpload, access: people},
	{pattern: "POST /api/import/{type}", handle: (*Server).apiImport, access: owner},
	{pattern: "GET /api/workspaces", handle: (*Server).apiWorkspaces, access: owner},
	{pattern: "GET /api/changes", handle: (*Server).apiChanges, access: people},
	{pattern: "POST /api/arrange", handle: (*Server).apiArrange, access: people},
	// And the tools the one asking may have: api_tools.go.
	{pattern: "POST /api/tools/{name}", handle: (*Server).apiTool, access: people},
	{pattern: "GET /api/{type}", handle: (*Server).apiList, access: people},
	{pattern: "POST /api/{type}", handle: (*Server).apiCreate, access: people},
	{pattern: "GET /api/{type}/{id}", handle: (*Server).apiGet, access: people},
	{pattern: "PUT /api/{type}/{id}", handle: (*Server).apiUpdate, access: people},
	{pattern: "PATCH /api/{type}/{id}", handle: (*Server).apiUpdate, access: people},
	{pattern: "DELETE /api/{type}/{id}", handle: (*Server).apiDelete, access: people},
	{pattern: "/api/", handle: (*Server).apiNotFound, access: people},
}
