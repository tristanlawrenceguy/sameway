package server

import "github.com/tristanlawrenceguy/sameway/internal/chat"

// The content types the server reads and writes by name are the chat's
// (chat/type_names.go), named here too so a page's code and its tests
// read as the server's own.
const (
	EntryType    = chat.EntryType
	FileType     = chat.FileType
	HabitType    = chat.HabitType
	ReminderType = chat.ReminderType
)
