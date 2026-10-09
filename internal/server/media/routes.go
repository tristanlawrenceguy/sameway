package media

import "github.com/tristanlawrenceguy/sameway/internal/web"

// Routes are media's addresses, which the server adds to its one route
// table (server/routes.go). A file and what is read from it are public
// with the page that shows it; its sound, copied out for writing it down,
// is not.
var Routes = []web.Route[*Service]{
	{Pattern: "POST /dictate", Handle: (*Service).dictate, Access: web.People, Persons: "their voice", Reach: web.Inward},
	{Pattern: "POST /speech/get", Handle: (*Service).speechGet, Access: web.Owner, Persons: "a download they are asked about", Reach: web.Outward},
	{Pattern: "POST /speech/speakers/get", Handle: (*Service).speakersGet, Access: web.Owner, Persons: "the same", Reach: web.Outward},
	{Pattern: "POST /meetings/teams/connect", Handle: (*Service).teamsConnect, Access: web.Owner, Persons: "signing in to their Microsoft account", Reach: web.Outward},
	{Pattern: "POST /t/file/upload", Handle: (*Service).upload, Access: web.People, Persons: "a file comes from their computer; the assistant reads files already added", Reach: web.Inward},
	{Pattern: "GET /files/{id}", Handle: (*Service).serveFile, Access: web.People, Public: true},
	{Pattern: "GET /files/{id}/still", Handle: (*Service).serveStill, Access: web.People, Public: true},
	{Pattern: "GET /files/{id}/captions.vtt", Handle: (*Service).captions, Access: web.People, Public: true},
	{Pattern: "GET /files/{id}/sound", Handle: (*Service).soundPlan, Access: web.People},
	{Pattern: "GET /files/{id}/sound/{n}", Handle: (*Service).soundChunk, Access: web.People},
	{Pattern: "POST /files/{id}/transcribe", Handle: (*Service).transcribeFile, Access: web.People, Tool: "write_down", Reach: web.Outward},
}
