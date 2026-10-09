// Package web is what every handler of the server shares: the route, who
// may use it and where what it changes is, the outcome a person is told,
// the way back to where they acted, the page around a body, JSON answers,
// and Deps, the narrow face of the server a feature's handlers are given.
// The server (internal/server) is the one route table; a feature package
// (internal/server/media, ...) lists its routes here and the server adds
// them, so who may use each address, its tool and its reach are still
// read from one place.
package web

import "net/http"

// Route is one address: what serves it, who may use it, for a page action
// the assistant's tool that does the same or why it is a person's alone,
// whether the internet may read it when what it shows is published, and
// where what it changes is. D is what its handler is given: the server
// itself for the server's own routes, a feature's own service for one of
// a feature package's.
type Route[D any] struct {
	Pattern string
	Handle  func(D, http.ResponseWriter, *http.Request)
	Access  Access
	// Tool is a page action's op (chat/op.go): what the assistant does to
	// do the same. Persons is instead why it is a person's alone.
	Tool, Persons string
	// Public says the internet may read it, when what it shows is
	// published: a published tab, list or record, its files and styles.
	Public bool
	// Reach is where what it changes is, so whether a dry run may try it
	// on a copy. Every route but a GET says.
	Reach Reach
}

// Access says who may use a route. A route nobody said anything about is
// the owner's alone, and the server's tests fail until it is said.
type Access int

const (
	// Unsaid is a route nobody said anything about: the owner's alone.
	Unsaid Access = iota
	// People are everyone let in: those who may look read, and those who
	// may change it change, except the owner's own kinds of record.
	People
	// Owner is the workspace's owner alone.
	Owner
)

// Reach says where what a route changes is, so whether a dry run (a
// change made on a throwaway copy of the workspace) can try it.
type Reach int

const (
	// Unreached is a route nobody said anything about: not tried.
	Unreached Reach = iota
	// Inward changes only this workspace's own records, files and settings.
	Inward
	// Outward reaches beyond them: the network, a notification, a
	// command, another workspace, this computer, the assistant's model,
	// another computer, or work that goes on after the answer (writing a
	// recording down), on a copy that is gone by then.
	Outward
	// ByTool is /api/tools/{name}: the tool it names says, by its
	// Traits.OpenWorld (chat/op.go).
	ByTool
)
