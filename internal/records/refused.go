package records

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// The same mistake is said in the same words by every way in: the API,
// the assistant's tools (and so MCP and POST /api/tools), the CLI and the
// pages. A type that is not there was said three ways ("there is no
// content type", "unknown content type", "there is no type") and a record
// that is not there four ("not found" over the API). Each way in adds
// only its own next step: the route, the tool or the command to run.

// NoType says a type is not there, and what is: the one likely meant,
// when the name is near one, then every one of names.
func NoType(name string, names []string) string {
	out := "there is no content type " + name
	if near := render.Nearest(name, names); near != "" {
		out += " (did you mean " + near + "?)"
	}
	return out + "; the workspace has " + strings.Join(names, ", ")
}

// NoRecord says a record is not there.
func NoRecord(typ, id string) string {
	return fmt.Sprintf("no %s with id %s", typ, id)
}

// NotTaken says what was sent to something that does not take it: each
// word it does not take, with the one likely meant, then every word it
// does. The API says it of a query word, the assistant's tools of an
// argument: one sent and ignored is an answer to another question.
func NotTaken(what string, unknown, takes []string) string {
	return fmt.Sprintf("%s does not take %s. It takes %s", what, Unknown(unknown, takes), strings.Join(takes, ", "))
}

// Unknown is words not taken, each with the one of takes likely meant.
func Unknown(unknown, takes []string) string {
	words := make([]string, len(unknown))
	for i, u := range slices.Sorted(slices.Values(unknown)) {
		words[i] = u
		if near := render.Nearest(u, takes); near != "" {
			words[i] += " (did you mean " + near + "?)"
		}
	}
	return strings.Join(words, ", ")
}
