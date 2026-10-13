package server

// apiRoutes are the JSON API's addresses that belong to a feature of the
// server's own (search, look, prose, types, actions, workspaces); the
// rest of the API is internal/server/api.
var apiRoutes = []route{
	{pattern: "GET /api/search", handle: (*Server).apiSearch, access: people},
	{pattern: "GET /api/look", handle: (*Server).apiLook, access: owner},
	{pattern: "POST /api/look", handle: (*Server).apiLook, access: owner, reach: outward},
	{pattern: "POST /api/prose", handle: (*Server).apiProse, access: people, reach: inward},
	{pattern: "POST /api/types", handle: (*Server).apiAddType, access: people, reach: inward},
	{pattern: "POST /api/types/{type}/fields", handle: (*Server).apiAddField, access: people, reach: inward},
	{pattern: "POST /api/act/{id}", handle: (*Server).apiAct, access: people, reach: outward},
	{pattern: "POST /hook/{token}", handle: (*Server).hook, access: people, reach: outward},
	{pattern: "GET /api/workspaces", handle: (*Server).apiWorkspaces, access: owner},
}
