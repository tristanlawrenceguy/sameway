package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// FieldGets says what the records a type already has hold in a field just
// added to it, read back as every page, filter and the API read them:
// "12 existing tasks get status To do", or "12 existing tasks get status:
// 9 To do, 3 Done" where they differ. Without it an agent that added a
// field with a default was surprised to find every old record had it,
// even the one that said otherwise. add_field and its REST route say it.
func FieldGets(st *store.Store, t *schema.Type, name string) string {
	f, ok := t.Field(name)
	if !ok || st == nil {
		return ""
	}
	recs, err := st.List(t.Name, store.ListOptions{})
	if err != nil {
		return ""
	}
	if len(recs) == 0 {
		return "there are no " + schema.Plural(t.Name) + " yet"
	}
	var order []string
	count := map[string]int{}
	for _, r := range recs {
		v := valueWords(*f, r.Fields[name])
		if count[v] == 0 {
			order = append(order, v)
		}
		count[v]++
	}
	kind := t.Name
	if len(recs) != 1 {
		kind = schema.Plural(t.Name)
	}
	gets := "get"
	if len(recs) == 1 {
		gets = "gets"
	}
	head := fmt.Sprintf("%d existing %s %s %s", len(recs), kind, gets, name)
	if len(order) == 1 {
		return head + " " + order[0]
	}
	parts := make([]string, len(order))
	for i, v := range order {
		parts[i] = fmt.Sprintf("%d %s", count[v], v)
	}
	return head + ": " + strings.Join(parts, ", ")
}

// valueWords is a stored value as a person reads it (export.Said): a
// choice by its label, yes or no, a day in words, and empty for nothing.
func valueWords(f schema.Field, v any) string {
	if l, ok := v.([]any); v == nil || v == "" || ok && len(l) == 0 {
		return "empty"
	}
	return export.Said(f, v, nil)
}
