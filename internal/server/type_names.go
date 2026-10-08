package server

import "github.com/tristanlawrenceguy/sameway/internal/records"

// The content types the server reads and writes by name are the chat's
// (records/type_names.go), named here too so a page's code and its tests
// read as the server's own.
const (
	EntryType    = records.EntryType
	FileType     = records.FileType
	HabitType    = records.HabitType
	ReminderType = records.ReminderType
)
