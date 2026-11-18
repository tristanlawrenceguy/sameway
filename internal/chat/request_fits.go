package chat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A request is checked against its kind's fields before it is put to the
// owner: one a yes could not make, with a field the kind does not have or
// a value it cannot take, goes back to the assistant to fix, so the owner
// is never asked a question whose yes would fail.

// requestType is the kind a request is of: for a new record any kind that
// is not the owner's alone, since it shows nothing the one asking did not
// send; for a change, one they may reach.
func (s *Service) requestType(tool, typeName string) (*schema.Type, error) {
	t, err := s.contentType(typeName)
	if tool == "create_record" && err != nil {
		if kept, e := records.ContentType(s.Store, typeName); e == nil && !kept.Owners {
			return kept, nil
		}
	}
	return t, err
}

// requestTypeNames are the kinds a request may be of: every content type
// that is not the owner's alone, in order.
func (s *Service) requestTypeNames() []string {
	var names []string
	for _, t := range records.ContentTypes(s.Store) {
		if !t.Owners {
			names = append(names, t.Name)
		}
	}
	sort.Strings(names)
	return names
}

// requestFits says why a yes could not make the request, or nil.
func (s *Service) requestFits(tool, typeName, id string, fields map[string]any) error {
	t, err := s.requestType(tool, typeName)
	if err != nil {
		return err
	}
	in := map[string]any{}
	if tool == "update_record" {
		rec, err := s.Store.Get(t.Name, id)
		if err != nil {
			return err
		}
		for k, v := range rec.Fields {
			in[k] = v
		}
	}
	for k, v := range fields {
		in[k] = v
		if f, ok := t.Field(k); ok && (f.ReadOnly || t.Name == records.PersonType && k == "always") {
			// What Sameway keeps, and a standing yes, which is the owner's
			// press alone (request_always.go): no request sets them, nor a
			// yes for good to changing people.
			return fmt.Errorf("%s is not theirs to ask for; leave it out", strings.ToLower(f.Display()))
		}
	}
	if err := records.RefsFit(s.Store, t, fields); err != nil {
		return fmt.Errorf("%v; a link has to name a record that is here: leave it out, and say who in words instead", err)
	}
	if _, err = t.Normalize(in, s.Now()); err != nil {
		// Someone who may not read the kind has not seen its fields, so the
		// refusal names them, for the next try to use.
		var names []string
		for _, f := range t.Shown() {
			if !f.ReadOnly {
				names = append(names, f.Name)
			}
		}
		return fmt.Errorf("%v; %s has the fields %s", err, withA(schema.Words(t.Name)), schema.AndNames(names))
	}
	return nil
}
