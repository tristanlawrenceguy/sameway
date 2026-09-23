package chat

import "context"

type viaKey struct{}

// WithVia marks a request as made from another device of the person's,
// such as their phone over the tailnet, so what it changes is logged with
// the device's name. A request on this machine carries none.
func WithVia(ctx context.Context, device string) context.Context {
	return context.WithValue(ctx, viaKey{}, device)
}

// Via is the device a request came from, or "" for this machine.
func Via(ctx context.Context) string {
	v, _ := ctx.Value(viaKey{}).(string)
	return v
}
