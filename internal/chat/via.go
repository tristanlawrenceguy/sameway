package chat

import "context"

// Access levels a person can have in a workspace they open from another
// device. The owner is whoever signed the machine in to Tailscale, and
// anyone at the machine itself.
const (
	Owner = "owner"
	Edit  = "edit"
	View  = "view"
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
