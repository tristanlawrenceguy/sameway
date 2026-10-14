package api

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// A list read with a word it does not take. An agent asked
// GET /api/activity?since=today for what happened today, was given all
// 890 entries, and took them for today's: since was not a word the list
// read, and it said nothing (crew finding 0742). A list now says which word
// it does not take, what it takes, and how to ask for what was meant.

// listParams are the words GET /api/{type} reads.
var listParams = map[string]bool{"where": true, "order": true, "dir": true, "limit": true, "page": true, "fields": true, "canvas": true}

// unknownParams is why a list read cannot be answered as asked, "" when it
// can.
func unknownParams(q url.Values, typ string) string {
	var unknown []string
	for k := range q {
		if !listParams[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) == 0 {
		return ""
	}
	sort.Strings(unknown)
	return fmt.Sprintf("GET /api/%s does not take %s, and read as if it were not there it would answer with every %s. "+
		"A list takes where (a condition, repeatable, such as created_at>=today or done=false), order (a field, -field for newest first), "+
		"limit, page and fields; GET /api/changes?since= is what changed after a point, and GET /api/describe says the rest",
		typ, strings.Join(unknown, ", "), strings.ReplaceAll(typ, "_", " "))
}
