package agenda

import "github.com/tristanlawrenceguy/sameway/internal/web"

// Routes are the agenda's addresses, which the server adds to its one
// route table (server/routes.go).
var Routes = []web.Route[*Service]{
	{Pattern: "GET /today", Handle: (*Service).todayPage, Access: web.People},
	{Pattern: "POST /today/done", Handle: (*Service).todayDone, Access: web.People, Tool: "update_record", Reach: web.Inward},
	{Pattern: "POST /today/move", Handle: (*Service).todayMove, Access: web.People, Tool: "update_record", Reach: web.Inward},
	{Pattern: "POST /today/late", Handle: (*Service).todayLate, Access: web.People, Tool: "update_record", Reach: web.Inward},
	{Pattern: "POST /today/sort", Handle: (*Service).todaySort, Access: web.People, Persons: "it is what they say to the assistant", Reach: web.Outward},
	{Pattern: "POST /brief", Handle: (*Service).briefSet, Access: web.Owner, Tool: "set_setting", Reach: web.Inward},
	{Pattern: "GET /review", Handle: (*Service).reviewPage, Access: web.People},
	{Pattern: "POST /review/next", Handle: (*Service).reviewMove, Access: web.People, Tool: "update_record", Reach: web.Inward},
	{Pattern: "POST /review/undate", Handle: (*Service).reviewUndate, Access: web.People, Tool: "update_record", Reach: web.Inward},
	{Pattern: "POST /review/done", Handle: (*Service).reviewDone, Access: web.People, Tool: "update_record", Reach: web.Inward},
	{Pattern: "POST /review/on", Handle: (*Service).reviewOn, Access: web.Owner, Tool: "set_setting", Reach: web.Inward},
}
