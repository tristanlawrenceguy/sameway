package chat

import (
	"errors"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A change made through the API or on the command line is a change like
// any other: logged as the person's, with how it came and what the
// record was before, so it can be taken back. It used to go unlogged, so
// an agent told to tidy up could delete what nobody could restore.

// Through names a way in that is not a page, for the log's sentence:
// "You deleted note Plan, through the API".
const (
	ThroughAPI = "through the API"
	ThroughCLI = "through the command line"
)

// Who made a change and how it came, as its sentence says it.
type Who struct {
	Actor   string // human, assistant, agent, system
	By      string // a person's or an agent's name
	Via     string // a device, or a way in such as ThroughAPI
	ByLogin string
}

// Write creates, updates or deletes one record, and says it as a change
// with what the record was before, so it can be undone. It does not log:
// the assistant's turn logs each change its tools make, with the reply.
// action is created, updated or deleted; id is empty for created.
func Write(st *store.Store, action, typ, id string, fields map[string]any) (*store.Record, Change, error) {
	if typ == ActivityType {
		return nil, Change{}, errors.New("the activity log is kept by Sameway and cannot be changed; to take a change back, undo it")
	}
	t, ok := st.Types().Get(typ)
	if !ok {
		return nil, Change{}, fmt.Errorf("there is no type %q", typ)
	}
	var rec, was *store.Record
	var err error
	if action != "created" {
		if was, err = st.Get(typ, id); err != nil {
			return nil, Change{}, err
		}
	}
	switch action {
	case "created":
		rec, err = st.Create(typ, fields)
	case "updated":
		rec, err = st.Update(typ, id, fields)
	case "deleted":
		rec, err = was, st.Delete(typ, id)
	default:
		err = fmt.Errorf("no way to write %q", action)
	}
	if err != nil {
		return nil, Change{}, err
	}
	c := Change{Action: action, Component: typ, ID: rec.ID, Detail: recordTitle(st, t, rec)}
	if action != "deleted" {
		c.Href = "/t/" + typ + "/" + rec.ID
	}
	if was != nil {
		c.Before = was.Fields
	}
	return rec, c, nil
}

// WriteAs is Write, logged as who's: how a record is written from a page,
// the API, the command line or MCP, so each writes, keeps what was there
// and logs it the same way, and none can forget one of the three (a file
// made over the API without content once was never logged, so it could
// not be undone). The system's own types, which change through the tools
// and are not the person's content, are written but not logged. It
// returns the record and its entry in the log.
func WriteAs(st *store.Store, who Who, action, typ, id string, fields map[string]any) (*store.Record, string, error) {
	rec, c, err := Write(st, action, typ, id, fields)
	if err != nil {
		return nil, "", err
	}
	if t, ok := st.Types().Get(typ); !ok || t.Internal {
		return rec, "", nil
	}
	c.By, c.Via, c.ByLogin = who.By, who.Via, who.ByLogin
	return rec, Record(st, who.Actor, c), nil
}

// Imported is the batch an import from a file made: records that were not
// there before, so undoing it takes them away together.
func Imported(typ string, ids []string) map[string]any {
	items := make([]BatchItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, BatchItem{Type: typ, ID: id})
	}
	return Batch(items)
}
