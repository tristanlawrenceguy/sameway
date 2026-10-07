package server

// routes is every address the server answers, in one table: it grew past
// server.go's room (tools/check keeps a file under 300 lines).
func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /{$}", s.canvasPage)
	m.HandleFunc("GET /c/{canvas}", s.canvasPage)
	m.HandleFunc("GET /chat", s.chatPage)
	m.HandleFunc("GET /help", s.helpPage)
	m.HandleFunc("POST /help/set", s.helpSet)
	m.HandleFunc("GET /canvas/{id}", s.focusPage)
	m.HandleFunc("POST /chat", s.chatSend)
	m.HandleFunc("POST /chat/stream", s.chatStream)
	m.HandleFunc("POST /chat/stop", s.chatStop)
	m.HandleFunc("GET /chat/live", s.chatLive)
	s.modelRoutes(m)                                       // model_key.go
	s.bringRoutes(m)                                       // bring.go
	s.calendarRoutes(m)                                    // calendar_links.go
	s.todayRoutes(m)                                       // today.go
	m.HandleFunc("POST /workspaces/example", s.tryExample) // example.go
	s.modelRoutes(m)                                       // model_key.go
	s.bringRoutes(m)                                       // bring.go
	s.calendarRoutes(m)                                    // calendar_links.go
	s.todayRoutes(m)                                       // today.go
	m.HandleFunc("POST /feedback", s.feedback)             // feedback.go
	m.HandleFunc("POST /t/{type}/add", s.addRecord)
	m.HandleFunc("POST /chat/clear", s.chatClear)
	m.HandleFunc("POST /chat/new", s.chatNew)
	m.HandleFunc("POST /chat/open", s.chatOpen)
	m.HandleFunc("POST /chat/delete", s.chatDelete)
	m.HandleFunc("POST /proposal/{id}/accept", s.proposalAccept)
	m.HandleFunc("POST /proposal/{id}/dismiss", s.proposalDismiss)
	m.HandleFunc("POST /proposal/{id}/instead", s.proposalInstead)
	m.HandleFunc("POST /activity/{id}/undo", s.undo)
	m.HandleFunc("POST /act/{id}", s.act)
	m.HandleFunc("POST /suggestions/{id}/{answer}", s.suggestionAnswer)
	m.HandleFunc("POST /suggestions/accept-all", s.suggestionsAcceptAll)
	m.HandleFunc("POST /canvas/{id}/props", s.blockProps)
	m.HandleFunc("POST /canvas/{id}/keep", s.canvasKeep)
	m.HandleFunc("POST /canvas/{id}/delete", s.canvasDelete)
	m.HandleFunc("GET /activity", s.activityPage)
	m.HandleFunc("POST /clock/set", s.clockSet)
	m.HandleFunc("POST /habit/{id}/log", s.habitLog)
	m.HandleFunc("POST /clock/{id}/done", s.clockDone)
	m.HandleFunc("POST /clock/{id}/snooze", s.clockSnooze)
	m.HandleFunc("GET /clock/stream", s.clockStream)
	m.HandleFunc("POST /sync", s.syncExchange)
	s.togetherRoutes(m)
	m.HandleFunc("GET /events", s.events)
	m.HandleFunc("GET /workspaces", s.workspacesPage)
	m.HandleFunc("POST /workspaces/start", s.workspacesStart)
	m.HandleFunc("GET /workspaces/new", s.workspacesNewPage)
	m.HandleFunc("POST /workspaces/new", s.workspacesNew)
	m.HandleFunc("GET /workspaces/copy", s.workspacesCopyPage)
	m.HandleFunc("POST /workspaces/copy", s.workspacesCopy)
	m.HandleFunc("GET /workspaces/delete", s.workspacesDeletePage)
	m.HandleFunc("POST /workspaces/delete", s.workspacesDelete)
	m.HandleFunc("POST /workspaces/restore", s.workspacesRestore)
	s.quitRoutes(m) // quit.go
	m.HandleFunc("GET /search", s.searchPage)
	m.HandleFunc("GET /when", s.whenRead)
	m.HandleFunc("GET /design", s.designPage)
	m.HandleFunc("GET /design/sameway.css", s.stylesheet)
	iconRoutes(m) // icon.go
	m.HandleFunc("GET /design/sameway.js", s.script)
	m.HandleFunc("GET /design/base/{file}", s.baseFile)

	m.HandleFunc("GET /t/{type}", s.listPage)
	m.HandleFunc("GET /t/{type}/import", s.importPage)
	m.HandleFunc("POST /t/{type}/import", s.importUpload)
	m.HandleFunc("POST /t/{type}/import/{file}/run", s.importRun)
	m.HandleFunc("GET /t/{type}/{id}", s.detailPage)
	m.HandleFunc("GET /t/{type}/{id}/whole", s.wholePage)
	m.HandleFunc("POST /t/{type}/{id}/parts/move", s.moveParts)
	m.HandleFunc("POST /t/{type}/{id}/delete", s.deleteForm)
	m.HandleFunc("POST /t/{type}/{id}/discard", s.discard)
	m.HandleFunc("POST /t/{type}/{id}/props", s.recordProps)
	m.HandleFunc("POST /t/file/upload", s.upload)
	m.HandleFunc("GET /files/{id}", s.serveFile)
	m.HandleFunc("GET /files/{id}/still", s.serveStill)
	s.recordingRoutes(m)
	s.agentRoutes(m) // api_agent.go
	m.HandleFunc("POST /speech/get", s.speechGet)
	m.HandleFunc("POST /speech/speakers/get", s.speakersGet)
	m.HandleFunc("POST /meetings/teams/connect", s.teamsConnect)
	m.HandleFunc("POST /dictate", s.dictate)

	m.HandleFunc("GET /api/describe", s.apiDescribe)
	m.HandleFunc("GET /api/search", s.apiSearch)
	m.HandleFunc("GET /api/describe/{part}", s.apiDescribePart)
	m.HandleFunc("GET /api/describe/{part}/{name}", s.apiDescribePart)
	m.HandleFunc("GET /api/look", s.apiLook)
	m.HandleFunc("POST /api/look", s.apiLook)
	m.HandleFunc("POST /api/prose", s.apiProse)
	m.HandleFunc("POST /api/types", s.apiAddType)
	m.HandleFunc("POST /api/types/{type}/fields", s.apiAddField)
	m.HandleFunc("POST /api/act/{id}", s.apiAct)
	m.HandleFunc("POST /hook/{token}", s.hook)
	m.HandleFunc("POST /api/chat", s.apiChat)
	m.HandleFunc("POST /api/chat/clear", s.apiChatClear)
	m.HandleFunc("POST /api/file/upload", s.apiFileUpload)
	m.HandleFunc("POST /api/import/{type}", s.apiImport)
	m.HandleFunc("GET /api/{type}", s.apiList)
	m.HandleFunc("POST /api/{type}", s.apiCreate)
	m.HandleFunc("GET /api/{type}/{id}", s.apiGet)
	m.HandleFunc("PUT /api/{type}/{id}", s.apiUpdate)
	m.HandleFunc("PATCH /api/{type}/{id}", s.apiUpdate)
	m.HandleFunc("DELETE /api/{type}/{id}", s.apiDelete)
	m.HandleFunc("/api/", s.apiNotFound)
	m.HandleFunc("POST /restart", s.restart)
	m.HandleFunc("POST /notify/phone", s.phoneSet)
	s.phoneRoutes(m) // phone_lan_page.go
}
