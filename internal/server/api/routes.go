package api

import "github.com/tristanlawrenceguy/sameway/internal/web"

// Routes are the API's own addresses, which the server adds to its one
// route table (server/routes.go). A page action of the API's is no
// person's and no tool's of its own; it is the tool, or the record, it
// names.
var Routes = []web.Route[*Service]{
	{Pattern: "GET /api/describe", Handle: (*Service).apiDescribe, Access: web.People},
	{Pattern: "GET /api/describe/{part}", Handle: (*Service).apiDescribePart, Access: web.People},
	{Pattern: "GET /api/describe/{part}/{name}", Handle: (*Service).apiDescribePart, Access: web.People},
	{Pattern: "POST /api/chat", Handle: (*Service).apiChat, Access: web.People, Reach: web.Outward},
	{Pattern: "POST /api/chat/clear", Handle: (*Service).apiChatClear, Access: web.People, Reach: web.Inward},
	{Pattern: "POST /api/file/upload", Handle: (*Service).apiFileUpload, Access: web.People, Reach: web.Inward},
	{Pattern: "GET /api/jobs", Handle: (*Service).apiJobs, Access: web.Owner},
	{Pattern: "GET /api/changes", Handle: (*Service).apiChanges, Access: web.People},
	{Pattern: "POST /api/arrange", Handle: (*Service).apiArrange, Access: web.People, Reach: web.Inward},
	{Pattern: "GET /api/layout", Handle: (*Service).apiLayout, Access: web.People}, // api_arrange.go
	// And the tools the one asking may have: api_tools.go.
	{Pattern: "POST /api/tools/{name}", Handle: (*Service).apiTool, Access: web.People, Reach: web.ByTool},
	{Pattern: "GET /api/{type}", Handle: (*Service).apiList, Access: web.People},
	{Pattern: "POST /api/{type}", Handle: (*Service).apiCreate, Access: web.People, Reach: web.Inward},
	{Pattern: "GET /api/{type}/{id}", Handle: (*Service).apiGet, Access: web.People},
	{Pattern: "PUT /api/{type}/{id}", Handle: (*Service).apiUpdate, Access: web.People, Reach: web.Inward},
	{Pattern: "PATCH /api/{type}", Handle: (*Service).apiUpdateMany, Access: web.People, Reach: web.Inward},
	{Pattern: "PATCH /api/{type}/{id}", Handle: (*Service).apiUpdate, Access: web.People, Reach: web.Inward},
	{Pattern: "DELETE /api/{type}/{id}", Handle: (*Service).apiDelete, Access: web.People, Reach: web.Inward},
	{Pattern: "/api/", Handle: (*Service).apiNotFound, Access: web.People, Reach: web.Inward},
}
