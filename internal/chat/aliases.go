package chat

import "github.com/tristanlawrenceguy/sameway/internal/records"

// The records as Sameway keeps them (who asks, writes, the log, names)
// are internal/records'; these are its names here while the rest of the
// code moves to it.

const (
	Owner  = records.Owner
	Edit   = records.Edit
	View   = records.View
	Host   = records.Host
	Public = records.Public

	ActionType       = records.ActionType
	ActivityType     = records.ActivityType
	AgentType        = records.AgentType
	BlockType        = records.BlockType
	CanvasType       = records.CanvasType
	ConversationType = records.ConversationType
	EntryType        = records.EntryType
	EventType        = records.EventType
	FileType         = records.FileType
	HabitType        = records.HabitType
	MessageType      = records.MessageType
	PersonType       = records.PersonType
	ProposalType     = records.ProposalType
	ReminderType     = records.ReminderType
	SuggestionType   = records.SuggestionType
	ComponentName    = records.ComponentName

	ThroughAPI = records.ThroughAPI
	ThroughCLI = records.ThroughCLI
	ThroughMCP = records.ThroughMCP
	ActorAgent = records.ActorAgent

	HomePath = records.HomePath
)

type (
	Visitor   = records.Visitor
	Agent     = records.Agent
	BatchItem = records.BatchItem
	Change    = records.Change
	Who       = records.Who
	Words     = records.Words
)

var (
	WithVisitor = records.WithVisitor
	VisitorOf   = records.VisitorOf
	WithVia     = records.WithVia
	Via         = records.Via

	ErrKeptLog   = records.ErrKeptLog
	AgentName    = records.AgentName
	AgentWho     = records.AgentWho
	Batch        = records.Batch
	CleanSummary = records.CleanSummary
	Imported     = records.Imported
	LocalEntry   = records.LocalEntry
	MachineName  = records.MachineName
	Name         = records.Name
	Print        = records.Print
	Prints       = records.Prints
	Record       = records.Record
	Resay        = records.Resay
	SameVersion  = records.SameVersion
	Sentence     = records.Sentence
	Summarise    = records.Summarise
	Version      = records.Version
	Write        = records.Write
	WriteAs      = records.WriteAs
	WriteKept    = records.WriteKept
	Say          = records.Say
	CanvasPath   = records.CanvasPath
	OnCanvas     = records.OnCanvas
)
