package chat

import (
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Who wrote a record's words is the records' to say (records/provenance.go);
// this service says it as the one it speaks for reads it.

// Writers reads the activity log for who wrote what. The service as a
// reader from the internet has it names nobody (see For).
func (s *Service) Writers() *records.Writers { return s.WritersFor(s.who.Access == records.Public) }

// RecordView is a record as an agent reads it.
func (s *Service) RecordView(t *schema.Type, rec *store.Record) records.RecordView {
	return s.ViewOf(t, rec, s.Writers())
}
