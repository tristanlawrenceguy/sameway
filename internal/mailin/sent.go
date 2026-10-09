package mailin

import (
	"fmt"
	"slices"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// The person's own replies are in their Sent folder, not their inbox: read
// from there when they answer a thread the workspace has, a thread says
// who wrote last, and whether the person is waiting or is waited on.

// sentNames are what mail services call the folder of what was sent, for
// a server that does not mark it.
var sentNames = []string{"Sent", "[Gmail]/Sent Mail", "Sent Items", "Sent Messages", "Sent Mail", "INBOX.Sent"}

// sentBox is the person's Sent folder, or "".
func sentBox(c *imapclient.Client) string {
	boxes, err := c.List("", "*", nil).Collect()
	if err != nil {
		return ""
	}
	have := map[string]bool{}
	for _, b := range boxes {
		if slices.Contains(b.Attrs, imap.MailboxAttrSent) {
			return b.Mailbox
		}
		have[b.Mailbox] = true
	}
	for _, n := range sentNames {
		if have[n] {
			return n
		}
	}
	return ""
}

// readSent is what the person sent since it was last read, that answers a
// thread the workspace has; read the first time from a week ago.
func readSent(c *imapclient.Client, marks map[string]Mark, wanted func([]string) bool) ([]Mail, error) {
	box := sentBox(c)
	if box == "" {
		return nil, nil
	}
	sel, err := c.Select(box, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return nil, nil
	}
	mark, seen := marks["sent"]
	criteria := &imap.SearchCriteria{Since: time.Now().AddDate(0, 0, -7)}
	last := uint32(0)
	if seen && mark.Validity == sel.UIDValidity {
		criteria = &imap.SearchCriteria{UID: []imap.UIDSet{{imap.UIDRange{Start: imap.UID(mark.Last + 1), Stop: 0}}}}
		last = mark.Last
	}
	found, err := c.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("%s could not be searched", box)
	}
	var uids []imap.UID
	next := Mark{Validity: sel.UIDValidity, Last: last}
	for _, u := range found.AllUIDs() {
		if uint32(u) > last {
			uids = append(uids, u)
			next.Last = max(next.Last, uint32(u))
		}
	}
	marks["sent"] = next
	if len(uids) == 0 {
		return nil, nil
	}
	got, err := fetch(c, box, uids)
	var out []Mail
	for _, m := range got {
		if len(m.Refs) > 0 && wanted(m.Refs) {
			m.Sent = true
			out = append(out, m)
		}
	}
	return out, err
}
