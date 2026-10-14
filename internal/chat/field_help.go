package chat

import (
	"errors"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// fieldHelp is what a refused write is told it should have given: what
// each refused field takes, as the API says it (schema/expect.go), so a
// model and an agent read the same words, then every field the type takes;
// the type's fields alone when the refusal names none.
func fieldHelp(err error, t *schema.Type) string {
	var ve *schema.ValidationError
	if !errors.As(err, &ve) || len(ve.Expects) == 0 {
		return typeHelp(t)
	}
	var said []string
	for _, s := range ve.Expects {
		said = append(said, s)
	}
	sort.Strings(said)
	return strings.Join(said, ". ") + ". " + typeHelp(t)
}
