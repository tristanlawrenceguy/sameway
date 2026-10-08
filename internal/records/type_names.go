package records

// The content types Sameway itself reads and writes by name, in one place.
// They were declared beside whatever used them first, and the server and
// the chat each had their own file, reminder, habit and entry, which
// nothing kept the same; the server's are these.
const (
	// ActionType holds a person's own actions: a button that does
	// something, inside Sameway or outside it.
	ActionType = "action"
	// ActivityType logs every change.
	ActivityType = "activity"
	// AgentType holds the agents let in with keys.
	AgentType = "agent"
	// BlockType holds canvas items.
	BlockType = "block"
	// CanvasType holds the tabs. The first canvas, Home, needs no record:
	// it is the blocks whose canvas is empty. Every record of this type is
	// one more tab beside it, with blocks of its own.
	CanvasType = "canvas"
	// ConversationType is a chat: its messages with its own history. A
	// person can have several and move between them; the one opened most
	// recently is the one the chat shows, and the only one the model is
	// told about.
	ConversationType = "conversation"
	// EntryType is what is logged against a habit.
	EntryType = "entry"
	// EventType is a meeting.
	EventType = "event"
	// FileType is what a person's files become; the server's upload makes
	// them.
	FileType = "file"
	// HabitType is a habit, which entries are logged against.
	HabitType = "habit"
	// MessageType holds conversation turns.
	MessageType = "message"
	// PersonType holds people.
	PersonType = "person"
	// ProposalType holds changes waiting for an answer.
	ProposalType = "proposal"
	// ReminderType is an alarm.
	ReminderType = "reminder"
	// SuggestionType is a suggested change.
	SuggestionType = "suggestion"
)

// ComponentName is the component that renders the conversation. It is a
// block like any other, so it can be moved, restyled, or removed; the
// server fills it with the live transcript when it renders the canvas.
const ComponentName = "chat"
