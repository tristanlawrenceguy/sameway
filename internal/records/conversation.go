package records

import (
	"sort"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An email conversation is the email that started it and every email
// whose thread names that one, in the order they were sent. Gmail, Apple
// Mail and Thunderbird all show a conversation this way, oldest first, so
// a reply always follows what it answers.

// EmailType is the record an email that came in is kept as.
const EmailType = "email"

// Conversation is the conversation an email is part of, oldest first, or
// nil when it is the only one.
func Conversation(st *store.Store, rec *store.Record) []*store.Record {
	t, ok := st.Types().Get(EmailType)
	if !ok || rec == nil || rec.Type != EmailType {
		return nil
	}
	if _, ok := t.Field("thread"); !ok {
		return nil
	}
	root, _ := rec.Fields["thread"].(string)
	if root == "" {
		root = rec.ID
	}
	first, err := st.Get(EmailType, root)
	if err != nil {
		return nil
	}
	rest, err := query.Filter(st, t, []string{"thread=" + root}, "", 0, time.Now())
	if err != nil || len(rest) == 0 {
		return nil
	}
	out := append([]*store.Record{first}, rest...)
	sort.SliceStable(out, func(i, j int) bool {
		a, _ := out[i].Fields["received"].(string)
		b, _ := out[j].Fields["received"].(string)
		if a != b && a != "" && b != "" {
			return a < b
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}
