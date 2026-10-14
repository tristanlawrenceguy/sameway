package server

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// refusedSaid is why a list page could not narrow or order its list as its
// address asked, in a person's words: by the labels its pages use, never a
// field as it is stored, and what it can be narrowed by instead.
func refusedSaid(t *schema.Type, err error) string {
	var r *query.Refused
	if !errors.As(err, &r) {
		return "This list could not be shown that way."
	}
	can := strings.Join(query.Narrowable(t), ", ")
	plural := schema.Plural(t.Name)
	switch {
	case r.Field == "":
		return fmt.Sprintf("“%s” is not a way to narrow %s. They can be narrowed by %s.", r.Text, plural, can)
	case r.Order:
		return fmt.Sprintf("%s have nothing called “%s” to put them in order by. They can be put in order by %s.", upperFirst(plural), r.Field, can)
	}
	return fmt.Sprintf("%s have nothing called “%s”, so this list cannot be narrowed by it. They can be narrowed by %s.", upperFirst(plural), r.Field, can)
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
