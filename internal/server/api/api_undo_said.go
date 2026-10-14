package api

// How to take a write back, in its answer. An agent that tried something
// could not take it back: undo_change takes a log entry's id, and no
// answer gave one, so an agent left what it tried, or deleted it by hand,
// a second change in the log for the person to read past. Every write now
// answers with its entry and the call that undoes it.

// undoSaid is how a write's answer says to take it back.
type undoSaid struct {
	Entry string `json:"entry"`
	How   string `json:"how"`
}

// undoOf is the undo for a log entry, nil when the change was not logged.
func undoOf(entry string) *undoSaid {
	if entry == "" {
		return nil
	}
	return &undoSaid{Entry: entry, How: `POST /api/tools/undo_change with {"id": "` + entry + `"} takes this change back, whole`}
}
