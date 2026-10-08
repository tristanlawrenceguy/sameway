// Package records is the workspace's records as Sameway keeps them,
// whoever asks: who is asking and what they may do, how a record is
// written and logged, the activity log and undoing it, what a record and
// a change are called, who wrote a record's words, and the pace an agent
// keeps. The pages, the API, MCP, the command line and the assistant all
// go through it, so each writes, logs, names and takes back the same way.
//
// It sits between the store and schema below and the server and chat
// above. The conversation with a model (its turns, prompts and tools)
// stays in internal/chat.
package records
