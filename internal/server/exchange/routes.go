package exchange

import "github.com/tristanlawrenceguy/sameway/internal/web"

// Routes are exchange's addresses, which the server adds to its one route
// table (server/routes.go).
var Routes = []web.Route[*Service]{
	{Pattern: "POST /api/import/{type}", Handle: (*Service).apiImport, Access: web.Owner, Reach: web.Inward},
	{Pattern: "GET /bring", Handle: (*Service).bringPage, Access: web.Owner},
	{Pattern: "POST /bring", Handle: (*Service).bring, Access: web.Owner, Tool: "import_records", Reach: web.Inward},
	{Pattern: "GET /t/{type}/import", Handle: (*Service).importPage, Access: web.Owner},
	{Pattern: "POST /t/{type}/import", Handle: (*Service).importUpload, Access: web.Owner, Tool: "import_records", Reach: web.Inward},
	{Pattern: "POST /t/{type}/import/{file}/run", Handle: (*Service).importRun, Access: web.Owner, Tool: "import_records", Reach: web.Inward},
	{Pattern: "GET /files/{id}/transcript.srt", Handle: (*Service).transcriptFile, Access: web.People, Public: true},
	{Pattern: "GET /files/{id}/transcript.txt", Handle: (*Service).transcriptFile, Access: web.People, Public: true},
	{Pattern: "GET /export/workspace.zip", Handle: (*Service).exportEverything, Access: web.Owner},
	{Pattern: "GET /export/all.ics", Handle: (*Service).exportCalendar, Access: web.People},
	{Pattern: "GET /export/{file}", Handle: (*Service).exportFile, Access: web.People, Public: true},
	{Pattern: "GET /export/{type}/{file}", Handle: (*Service).exportDocument, Access: web.People, Public: true},
}
