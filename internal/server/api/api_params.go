package api

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// A read with a word it does not take. An agent asked
// GET /api/activity?since=today for what happened today, was given all
// 890 entries, and took them for today's: since was not a word the list
// read, and it said nothing (crew finding 0742). Every read now says which
// word it does not take, the one likely meant, what it takes, and where
// to read more, as Stripe answers "Received unknown parameter" and
// JSON:API answers 400. A word read as if it were not there is an answer
// to another question.

// The words each read takes.
var (
	listWords     = []string{"where", "order", "dir", "limit", "page", "fields", "canvas"}
	recordWords   = []string{"fields"}
	changesWords  = []string{"since", "limit", "wait"}
	describeWords = []string{"full"}
)

// unknownParams is why a list read cannot be answered as asked, "" when it
// can.
func unknownParams(q url.Values, typ string) string {
	unknown := unknownWords(q, listWords)
	if len(unknown) == 0 {
		return ""
	}
	return fmt.Sprintf("GET /api/%s does not take %s, and read as if it were not there it would answer with every %s. "+
		"A list takes where (a condition, repeatable, such as created_at>=today or done=false), order (a field, -field for newest first), "+
		"limit, page and fields; GET /api/changes?since= is what changed after a point, and GET /api/describe says the rest",
		typ, records.Unknown(unknown, listWords), strings.ReplaceAll(typ, "_", " "))
}

// refusedWords refuses a read sent a word it does not take, and says
// whether it did.
func refusedWords(w http.ResponseWriter, r *http.Request, route string, takes []string) bool {
	unknown := unknownWords(r.URL.Query(), takes)
	if len(unknown) == 0 {
		return false
	}
	web.WriteJSON(w, http.StatusBadRequest, map[string]any{"error": web.APIError{Code: "bad_request",
		Message: records.NotTaken("GET "+route, unknown, takes) + "; GET /api/describe/routes says the rest", Describe: "/api/describe/routes"}})
	return true
}

// unknownWords is the words in q not among takes, in order.
func unknownWords(q url.Values, takes []string) []string {
	var unknown []string
	for k := range q {
		if !slices.Contains(takes, k) {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	return unknown
}
