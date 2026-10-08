package records

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// ContentTypes lists the types the assistant may write to: everything the
// schema declares except the system's own (messages, blocks, activity).
func ContentTypes(st *store.Store) []*schema.Type {
	var out []*schema.Type
	for _, t := range st.Types().Types {
		switch {
		case !t.Content(), t.Name == MessageType, t.Name == BlockType, t.Name == ActivityType:
			continue
		}
		out = append(out, t)
	}
	return out
}

// TypeNames are the content types' names, in order.
func TypeNames(st *store.Store) []string {
	var names []string
	for _, t := range ContentTypes(st) {
		names = append(names, t.Name)
	}
	sort.Strings(names)
	return names
}

// ContentType is the content type a name means, or what the workspace
// has instead.
func ContentType(st *store.Store, name string) (*schema.Type, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, t := range ContentTypes(st) {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, fmt.Errorf("unknown content type %q. The workspace has: %s", name, strings.Join(TypeNames(st), ", "))
}

// DeleteRecord is not a tool: a record goes when a person deletes it, or
// when its creation is undone. Either way the log keeps what it was.
func DeleteRecord(st *store.Store, typeName, id string) (Change, error) {
	t, err := ContentType(st, typeName)
	if err != nil {
		return Change{}, err
	}
	if _, err := st.Get(t.Name, id); err != nil {
		return Change{}, fmt.Errorf("no %s with id %s", t.Name, id)
	}
	_, c, err := Write(st, "deleted", t.Name, id, nil)
	if err != nil {
		return Change{}, fmt.Errorf("could not delete %s %s: %v", t.Name, id, err)
	}
	return c, nil
}
