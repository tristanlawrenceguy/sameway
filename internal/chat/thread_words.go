package chat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A reply read alone is half a conversation: "Thanks, got it!" looked like
// something to do, "Friday works" said nothing of what. So a record in a
// thread (an email answering another) is read with what came before it in
// that thread, the latest few, each saying whether the person wrote it.

// threadMost is how many earlier messages are read with one.
const threadMost = 3

// threadWords is what came before a record in its thread, as the model
// reads it, or "".
func threadWords(st *store.Store, t *schema.Type, rec *store.Record) string {
	f, ok := t.Field("thread")
	if !ok || f.To != t.Name {
		return ""
	}
	root, _ := rec.Fields["thread"].(string)
	if root == "" {
		return ""
	}
	all, err := st.List(t.Name, store.ListOptions{})
	if err != nil {
		return ""
	}
	var before []*store.Record
	for _, r := range all {
		if r.ID == rec.ID {
			continue
		}
		if r.ID == root || r.Fields["thread"] == root {
			if sentOf(r).Before(sentOf(rec)) || sentOf(r).Equal(sentOf(rec)) && r.CreatedAt.Before(rec.CreatedAt) {
				before = append(before, r)
			}
		}
	}
	if len(before) == 0 {
		return ""
	}
	sort.Slice(before, func(i, j int) bool { return sentOf(before[i]).Before(sentOf(before[j])) })
	if len(before) > threadMost {
		before = before[len(before)-threadMost:]
	}
	var lines []string
	for _, r := range before {
		who, _ := r.Fields["from"].(string)
		if mine, _ := r.Fields["from_me"].(bool); mine {
			who = "the person themselves"
		}
		body, _ := r.Fields["body"].(string)
		lines = append(lines, fmt.Sprintf("- %s, from %s: %s", sentOf(r).Format("Mon 2 Jan"), who, clipRunes(strings.Join(strings.Fields(body), " "), 400)))
	}
	return "\n\nThis record is the latest message of a conversation. What came before it, oldest first, is only to understand it: decide by what the latest message asks of the person now. A question to them, or something left for them to answer, is theirs to reply to; what answers or closes what they asked is not.\n" + strings.Join(lines, "\n")
}
