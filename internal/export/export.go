// Package export takes records out in the formats people bring them in
// with: any list as a spreadsheet (CSV, Excel), people as contacts
// (vCard), and anything with a date as a calendar (iCalendar). Values are
// written as a person reads them and as the import reads them back, so a
// file taken out and brought in again comes back as it was: days as
// 2026-10-05, times as 2026-10-05 09:30, a choice by its label, a repeat
// in words, a yes or no, a list with commas.
package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A Format is one way out: what it is called, its extension and its type.
type Format struct {
	Ext, Label, Type string
}

var (
	CSV   = Format{"csv", "Spreadsheet (CSV)", "text/csv; charset=utf-8"}
	Excel = Format{"xlsx", "Excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}
	VCard = Format{"vcf", "Contacts (vCard)", "text/vcard; charset=utf-8"}
	ICS   = Format{"ics", "Calendar (iCalendar)", "text/calendar; charset=utf-8"}
)

// For is the formats a type can be taken out as: every list as a
// spreadsheet, people as contacts, anything with a date as a calendar.
func For(t *schema.Type) []Format {
	out := []Format{CSV, Excel}
	if contactish(t) {
		out = append(out, VCard)
	}
	if dateField(t) != "" {
		out = append(out, ICS)
	}
	return out
}

// ByExt is the format with an extension.
func ByExt(t *schema.Type, ext string) (Format, bool) {
	for _, f := range For(t) {
		if f.Ext == ext {
			return f, true
		}
	}
	return Format{}, false
}

func contactish(t *schema.Type) bool {
	_, email := t.Field("email")
	_, phone := t.Field("phone")
	return email || phone
}

// dateField is the field a record's day comes from: starts, else the
// first date field.
func dateField(t *schema.Type) string {
	if f, ok := t.Field("starts"); ok && f.Type == "datetime" {
		return f.Name
	}
	for _, f := range t.Shown() {
		if f.Type == "datetime" {
			return f.Name
		}
	}
	return ""
}

// Dated says whether a record has the day a calendar places it on.
func Dated(t *schema.Type, rec *store.Record) bool {
	f := dateField(t)
	v, _ := rec.Fields[f].(string)
	return f != "" && v != ""
}

// Titles says what a ref points at, by its title.
type Titles func(f schema.Field, id string) string

// Fields are the fields taken out: the ones a page shows, and a hidden
// uid, which lets the file come back in without adding anything twice.
func Fields(t *schema.Type) []schema.Field {
	out := t.Shown()
	if f, ok := t.Field("uid"); ok && f.Hidden {
		out = append(out, *f)
	}
	return out
}

// Header is each field's name as a person reads it.
func Header(fields []schema.Field) []string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = f.Label
		if out[i] == "" {
			out[i] = strings.ToUpper(f.Name[:1]) + strings.ReplaceAll(f.Name[1:], "_", " ")
		}
	}
	return out
}

// Value is one field of a record as it is written out.
func Value(f schema.Field, v any, titles Titles) string {
	if v == nil {
		return ""
	}
	switch f.Type {
	case "list":
		items, _ := v.([]any)
		parts := make([]string, 0, len(items))
		for _, it := range items {
			if f.RefList() && titles != nil {
				parts = append(parts, titles(f, fmt.Sprint(it))) // people by name
				continue
			}
			parts = append(parts, fmt.Sprint(it))
		}
		return strings.Join(parts, ", ")
	case "bool":
		if b, _ := v.(bool); b {
			return "yes"
		}
		return "no"
	case "datetime":
		return Day(fmt.Sprint(v))
	case "repeat":
		return when.RepeatText(fmt.Sprint(v))
	case "enum":
		return f.ValueLabel(fmt.Sprint(v))
	case "ref":
		if titles != nil {
			return titles(f, fmt.Sprint(v))
		}
	}
	return fmt.Sprint(v)
}

// Day is a stored time as it is written out: a day alone, or a day and a
// time on this computer's clock.
func Day(v string) string {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return v
	}
	if strings.HasSuffix(v, "T00:00:00Z") {
		return t.UTC().Format("2006-01-02")
	}
	return t.Local().Format("2006-01-02 15:04")
}

// Rows are the records as rows of text under the header.
func Rows(fields []schema.Field, recs []*store.Record, titles Titles) [][]string {
	rows := make([][]string, 0, len(recs))
	for _, r := range recs {
		row := make([]string, len(fields))
		for i, f := range fields {
			row[i] = Value(f, r.Fields[f.Name], titles)
		}
		rows = append(rows, row)
	}
	return rows
}
