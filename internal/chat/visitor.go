package chat

import "github.com/tristanlawrenceguy/sameway/internal/records"

// Who is asking and what they may do is the records' to say; see
// internal/records/visitor.go.

const (
	Owner  = records.Owner
	Edit   = records.Edit
	View   = records.View
	Host   = records.Host
	Public = records.Public
)

type Visitor = records.Visitor

var (
	WithVisitor = records.WithVisitor
	VisitorOf   = records.VisitorOf
	WithVia     = records.WithVia
	Via         = records.Via
)
