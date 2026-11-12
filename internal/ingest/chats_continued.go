package ingest

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A chat a person carries on after exporting it is in the next export
// longer: its title the same, its start what was brought before. It is
// one conversation, so it is one note: the note brought before takes the
// new words, rather than a second note coming beside it. An export does
// not keep the conversation's id in the note, so it is known by its title
// and its words.

// Continued takes out of tb the rows that carry on a note already here,
// and says, by each note's id, the words it now has. A row the same as a
// note is left out by Unseen first.
func (tb *Table) Continued(st *store.Store, t *schema.Type) map[string]string {
	if t.Title == "" || !contains(tb.Columns, "body") {
		return nil
	}
	recs, err := st.List(t.Name, store.ListOptions{})
	if err != nil {
		return nil
	}
	here := map[string][]*store.Record{}
	for _, rec := range recs {
		title, _ := rec.Fields[t.Title].(string)
		here[strings.TrimSpace(title)] = append(here[strings.TrimSpace(title)], rec)
	}
	longer := map[string]string{}
	rows := tb.Rows[:0]
	for _, row := range tb.Rows {
		if id := carriedOn(here[strings.TrimSpace(row[t.Title])], strings.TrimSpace(row["body"]), longer); id != "" {
			longer[id] = strings.TrimSpace(row["body"])
			continue
		}
		rows = append(rows, row)
	}
	tb.Rows = rows
	return longer
}

// carriedOn is the note of these whose words body starts with, unless
// another row has carried it on already.
func carriedOn(notes []*store.Record, body string, taken map[string]string) string {
	for _, rec := range notes {
		was, _ := rec.Fields["body"].(string)
		was, now := said(was), said(body)
		if _, ok := taken[rec.ID]; !ok && was != "" && len(now) > len(was) && strings.HasPrefix(now, was) {
			return rec.ID
		}
	}
	return ""
}

// said is a conversation's words without the day it began, which an
// import before said in UTC.
func said(body string) string {
	body = strings.TrimSpace(body)
	if first, rest, ok := strings.Cut(body, "\n\n"); ok && strings.HasPrefix(first, "Began ") {
		return strings.TrimSpace(rest)
	}
	return body
}
