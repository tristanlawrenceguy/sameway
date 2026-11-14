package agenda

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/fold"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A household reads several mailboxes into one workspace, on its several
// computers, so it sees what no one mailbox does: two people paying for the
// same service apart. The renewal notice, when there is still time to
// change it, says so.

// paidTwice is what the notice adds when someone else in the household
// pays for the same service and has not cancelled it, or "".
func (s *Service) paidTwice(sub *store.Record, subs []*store.Record) string {
	name, _ := sub.Fields["name"].(string)
	mine, _ := sub.Fields["mailbox"].(string)
	if strings.TrimSpace(name) == "" || mine == "" {
		return ""
	}
	for _, o := range subs {
		theirs, _ := o.Fields["mailbox"].(string)
		gone, _ := o.Fields["cancelled"].(bool)
		other, _ := o.Fields["name"].(string)
		if o.ID == sub.ID || gone || theirs == "" || fold.Key(theirs) == fold.Key(mine) || fold.Key(other) != fold.Key(name) {
			continue
		}
		return fmt.Sprintf(" %s pays for %s too: one plan for you both may cost less.", s.whoseMailbox(theirs), strings.TrimSpace(name))
	}
	return ""
}

// whoseMailbox is the person whose email address it is, by name, else
// the address.
func (s *Service) whoseMailbox(address string) string {
	people, _ := s.app.Store.List(records.PersonType, store.ListOptions{})
	for _, p := range people {
		if e, _ := p.Fields["email"].(string); e != "" && fold.Key(e) == fold.Key(address) {
			if n, _ := p.Fields["name"].(string); strings.TrimSpace(n) != "" {
				return strings.TrimSpace(n)
			}
		}
	}
	return address
}
