package server

import (
	"html/template"
	"net/url"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A list page narrows by where (and sorts by order); anything else in its
// address was passed over without a word, so /t/task?done=false&overdue=true
// showed every task under a title that read as if it had narrowed them, and
// a person took it for the overdue ones. The page now says what it did not
// read, and links to the address that says it: a field given as field=value
// is where=field=value, and overdue or late is the type's day before today.

// listNotRead is the line saying what a list's address asked that the list
// does not read, with a link to the list as asked; "" when it read all.
func listNotRead(t *schema.Type, q url.Values) template.HTML {
	var asked, conds []string
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := q.Get(k)
		switch {
		case k == "where" || k == "order" || k == "page":
		case k == "overdue" || k == "late":
			if day := t.DayField(); day != "" && v != "false" {
				asked, conds = append(asked, k), append(conds, day+"<today")
			}
		default:
			if _, ok := t.Field(k); ok {
				asked, conds = append(asked, k+"="+v), append(conds, k+"="+v)
			}
		}
	}
	if len(conds) == 0 {
		return ""
	}
	to := url.Values{"where": append(q["where"], conds...)}
	if o := q.Get("order"); o != "" {
		to.Set("order", o)
	}
	return template.HTML(`<p class="sw-muted">This list does not read ` + template.HTMLEscapeString(strings.Join(asked, " or ")) +
		` in its address, so it shows every ` + template.HTMLEscapeString(schema.Words(t.Name)) + `. <a class="sw-link" href="/t/` + t.Name + `?` + template.HTMLEscapeString(to.Encode()) + `">Show only those</a></p>`)
}
