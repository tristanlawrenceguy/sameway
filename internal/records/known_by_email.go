package records

import (
	"github.com/tristanlawrenceguy/sameway/internal/fold"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Someone let in, or let to ask, is known by the email on their page:
// whoever signs in to Tailscale with it gets what the owner gave them.
// Access itself is kept by Sameway, but the email was not, so someone who
// may edit could put their own second address on the page of someone who
// hosts the workspace and come in as them. While a person has access, or
// may ask, their email stays as the owner agreed it; it changes once that
// is taken away.

// knownByEmail refuses a change to the email of a person who has access
// or may ask, from every way in.
func knownByEmail(t *schema.Type, fields, now map[string]any) error {
	if t.Name != PersonType || now == nil {
		return nil
	}
	v, sent := fields["email"]
	to, _ := v.(string)
	was, _ := now["email"].(string)
	if !sent || fold.Same(to, was) {
		return nil
	}
	for _, kept := range []string{"access", "may_ask"} {
		if list, _ := now[kept].([]any); len(list) > 0 || now[kept] != nil && now[kept] != "" && list == nil {
			return &schema.ValidationError{Problems: map[string]string{"email": "is who Sameway lets in as this person; take their access away first, then change it"}}
		}
	}
	return nil
}
