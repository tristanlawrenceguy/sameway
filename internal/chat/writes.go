package chat

import (
	"errors"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
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

// ErrKeptLog is the activity log refusing a hand, from every way in: it is
// what makes every other change reversible.
var ErrKeptLog = errors.New("the activity log is kept by Sameway and cannot be changed; to take a change back, undo it: Undo on the activity page, POST /activity/<id>/undo, or undo_change")

// Write creates, updates or deletes one record, and says it as a change
// with what the record was before, so it can be undone. It does not log:
// the assistant's turn logs each change its tools make, with the reply.
// action is created, updated or deleted; id is empty for created.
func Write(st *store.Store, action, typ, id string, fields map[string]any) (*store.Record, Change, error) {
	if t, ok := st.Types().Get(typ); ok {
		if err := refNamesToIDs(st, t, fields); err != nil { // ref_names.go
			return nil, Change{}, err
		}
		var now map[string]any
		if action == "updated" {
			if was, err := st.Get(typ, id); err == nil {
				now = was.Fields
			}
		}
		if err := keptFields(t, fields, now); err != nil {
			return nil, Change{}, err
		}
	}
	return write(st, action, typ, id, fields)
}

// keptFields refuses a field Sameway keeps (a file's path, a person's
// access, whether an action was accepted), for everyone and from every
// way in: the page, the API, the command line and the assistant each
// had their own rule, and the owner could set over the API what the page
// refused them.
// A field sent as it already is changes nothing and is let be: a page
// sends the whole record back.
func keptFields(t *schema.Type, fields, now map[string]any) error {
	problems := map[string]string{}
	for _, f := range t.Fields {
		v, sent := fields[f.Name]
		if !sent || !f.ReadOnly || now != nil && Print(v) == Print(now[f.Name]) {
			continue
		}
		problems[f.Name] = "is kept by Sameway and cannot be changed by hand; leave it out"
	}
	if len(problems) > 0 {
		return &schema.ValidationError{Problems: problems}
	}
	return nil
}

// WriteKept is WriteAs for a record whose kept fields the caller worked
// out itself, such as a file's name and kind from what was sent. Only
// Sameway's own code calls it, never with what someone typed as those
// fields.
func WriteKept(st *store.Store, who Who, action, typ, id string, fields map[string]any) (*store.Record, string, error) {
	rec, c, err := write(st, action, typ, id, fields)
	if err != nil {
		return nil, "", err
	}
	return rec, logAs(st, who, c), nil
}

func write(st *store.Store, action, typ, id string, fields map[string]any) (*store.Record, Change, error) {
	if typ == ActivityType {
		return nil, Change{}, ErrKeptLog
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
	return rec, logAs(st, who, c), nil
}

// logAs logs a change as who's, unless it is to one of the system's own
// types, which change through the tools and are not the person's content.
func logAs(st *store.Store, who Who, c Change) string {
	// An agent's key is the system's kind, but taking one away is a change
	// the owner makes, and undoing it lets the agent back in.
	if t, ok := st.Types().Get(c.Component); !ok || t.Internal && t.Name != AgentType {
		return ""
	}
	c.By, c.Via, c.ByLogin = who.By, who.Via, who.ByLogin
	return Record(st, who.Actor, c)
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
