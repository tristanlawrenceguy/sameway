package connect

import "github.com/tristanlawrenceguy/sameway/internal/web"

// Routes are connect's addresses, which the server adds to its one route
// table (server/routes.go).
var Routes = []web.Route[*Service]{
	{Pattern: "GET /apps", Handle: (*Service).appsPage, Access: web.Owner},
	{Pattern: "POST /apps/connect", Handle: (*Service).appsConnect, Access: web.Owner, Persons: "what reaches into their other programs is theirs to say", Reach: web.Outward},
	{Pattern: "POST /model/use", Handle: (*Service).modelUse, Access: web.Owner, Tool: "set_setting", Reach: web.Outward},
	{Pattern: "POST /model/check", Handle: (*Service).modelCheck, Access: web.Owner, Persons: "checking the model is checking the assistant itself", Reach: web.Outward},
	{Pattern: "GET /model/wait", Handle: (*Service).modelWaitState, Access: web.Owner},
	{Pattern: "POST /model/key", Handle: (*Service).modelKey, Access: web.Owner, Persons: "a key is their own secret, pasted by them; no model is handed one", Reach: web.Outward},
	{Pattern: "POST /model/ollama", Handle: (*Service).ollamaFetch, Access: web.Owner, Persons: "fetching gigabytes onto their computer is theirs to start", Reach: web.Outward},
}
