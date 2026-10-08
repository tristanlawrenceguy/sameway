package records

import (
	"context"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Access levels a person can have in a workspace they open from another
// device. The owner is whoever signed the machine in to Tailscale, and
// anyone at the machine itself.
const (
	Owner = "owner"
	Edit  = "edit"
	View  = "view"
	// Host is someone whose own computer holds a copy of the workspace and
	// keeps it in step: they may do what an editor may, and their copy can
	// change anything, so it is the owner's gravest yes.
	Host = "host"
	// Public is anyone on the internet, reading what the owner published.
	Public = "public"
)

// Visitor is who a request comes from when it is not the machine itself:
// the person, by their record and name, their Tailscale login, what they
// may do, and the device they are on.
type Visitor struct {
	Person string // the person record, when there is one
	Name   string
	Login  string
	Access string
	Device string
	// Agent says it is an agent let in with a key of its own, named by
	// the key (agent_keys.go) rather than by what it says of itself.
	Agent bool
}

// Owner says whether the visitor may do everything.
func (v Visitor) Owner() bool { return v.Access == Owner }

// Who is the name the log gives them: "" for the owner, who is "You".
func (v Visitor) Who() string {
	if v.Owner() {
		return ""
	}
	if v.Name != "" {
		return v.Name
	}
	return v.Login
}

type visitorKey struct{}

// WithVisitor marks a request as made by someone on another device.
func WithVisitor(ctx context.Context, v Visitor) context.Context {
	return context.WithValue(ctx, visitorKey{}, v)
}

// VisitorOf is who made a request. A request carrying no visitor was made
// at the machine itself, by its owner.
func VisitorOf(ctx context.Context) Visitor {
	if v, ok := ctx.Value(visitorKey{}).(Visitor); ok {
		return v
	}
	return Visitor{Access: Owner}
}

// WithVia marks a request as made from another device of the owner's.
func WithVia(ctx context.Context, device string) context.Context {
	return WithVisitor(ctx, Visitor{Access: Owner, Device: device})
}

// Via is the device a request came from, or "" for this machine.
func Via(ctx context.Context) string { return VisitorOf(ctx).Device }

// PersonByEmail is the person who signs in with this email, if any.
func (b *Book) PersonByEmail(email string) *store.Record {
	if _, ok := b.Store.Types().Get(PersonType); !ok || email == "" {
		return nil
	}
	people, err := b.Store.List(PersonType, store.ListOptions{})
	if err != nil {
		return nil
	}
	for _, p := range people {
		if e, _ := p.Fields["email"].(string); strings.EqualFold(strings.TrimSpace(e), email) {
			return p
		}
	}
	return nil
}

// PersonColour is the colour someone's changes are shown in, 1 to 6, from
// their login: the same person has the same colour on every computer.
func PersonColour(login string) int {
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return 0
	}
	h := uint32(2166136261)
	for i := 0; i < len(login); i++ {
		h = (h ^ uint32(login[i])) * 16777619
	}
	return int(h%6) + 1
}
