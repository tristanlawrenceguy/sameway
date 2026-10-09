package server

// ownerRoutes are the owner's: the workspaces on this machine, the model,
// bringing things in, the phone and the Wi-Fi, and Sameway itself.
var ownerRoutes = []route{
	{pattern: "GET /workspaces", handle: (*Server).workspacesPage, access: owner},
	{pattern: "POST /workspaces/start", handle: (*Server).workspacesStart, access: owner, tool: "open_workspace", reach: outward},
	{pattern: "GET /workspaces/new", handle: (*Server).workspacesNewPage, access: owner},
	{pattern: "POST /workspaces/new", handle: (*Server).workspacesNew, access: owner, tool: "add_workspace", reach: outward},
	{pattern: "GET /workspaces/copy", handle: (*Server).workspacesCopyPage, access: owner},
	{pattern: "POST /workspaces/copy", handle: (*Server).workspacesCopy, access: owner, tool: "add_workspace", reach: outward},
	{pattern: "GET /workspaces/delete", handle: (*Server).workspacesDeletePage, access: owner},
	{pattern: "POST /workspaces/delete", handle: (*Server).workspacesDelete, access: owner, persons: "deleting a whole workspace is its owner's", reach: outward},
	{pattern: "POST /workspaces/restore", handle: (*Server).workspacesRestore, access: owner, tool: "restore_workspace", reach: outward},
	{pattern: "POST /workspaces/example", handle: (*Server).tryExample, access: owner, tool: "add_workspace", reach: outward},
	{pattern: "POST /workspaces/from-copy", handle: (*Server).fromCopy, access: owner, persons: "a copy is a file they choose on their computer", reach: outward},
	{pattern: "POST /backup/cloud", handle: (*Server).cloudSet, access: owner, tool: "set_setting", reach: outward},
	{pattern: "POST /cloud-sync", handle: (*Server).cloudSyncOn, access: owner, persons: "which computers hold a copy of the workspace is its owner's choice", reach: outward},
	{pattern: "POST /cloud-sync/off", handle: (*Server).cloudSyncOff, access: owner, persons: "which computers hold a copy of the workspace is its owner's choice", reach: outward},
	{pattern: "POST /workspaces/join", handle: (*Server).cloudJoin, access: owner, persons: "which computers hold a copy of the workspace is its owner's choice", reach: outward},
	{pattern: "POST /meaning/fetch", handle: (*Server).meaningFetch, access: owner, persons: "what is fetched to this computer is its owner's choice", reach: outward},
	{pattern: "GET /calendars", handle: (*Server).calendarsPage, access: owner},
	{pattern: "POST /calendars/add", handle: (*Server).calendarAdd, access: owner, persons: "a calendar link is their own secret, pasted by them", reach: outward},
	{pattern: "POST /calendars/remove", handle: (*Server).calendarRemove, access: owner, persons: "a calendar link is their own secret, pasted by them", reach: outward},
	{pattern: "POST /notify/phone", handle: (*Server).phoneSet, access: owner, tool: "set_setting", reach: outward},
	{pattern: "POST /phone/on", handle: (*Server).phoneOn, access: owner, persons: "letting other devices reach the workspace is its owner's choice", reach: outward},
	{pattern: "POST /phone/off", handle: (*Server).phoneOff, access: owner, persons: "letting other devices reach the workspace is its owner's choice", reach: outward},
	{pattern: "POST /phone/forget", handle: (*Server).phoneForget, access: owner, persons: "letting other devices reach the workspace is its owner's choice", reach: outward},
	{pattern: "POST /phone/invite", handle: (*Server).phoneInvite, access: owner, persons: "who may come in is the owner's to say, face to face", reach: outward}, // lan_invite.go
	{pattern: "POST /feedback", handle: (*Server).feedback, access: owner, persons: "what goes to the makers is theirs to read and send", reach: inward},
	{pattern: "POST /at-login", handle: (*Server).atLoginSet, access: owner, persons: "what starts when this computer does is its owner's choice", reach: outward},
	{pattern: "POST /quit", handle: (*Server).quit, access: owner, persons: "stopping Sameway ends the assistant's own turn", reach: outward},
	{pattern: "POST /restart", handle: (*Server).restart, access: owner, persons: "restarting ends the assistant's own turn", reach: outward},
}
