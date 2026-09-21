// Package ingest brings records in from files a person already has: a CSV
// from anywhere, a vCard of contacts, a mailbox export. Whatever the file,
// it becomes a table of text under column names; the columns are matched
// to a content type's fields, by name and by what they plainly are; and
// each row becomes a record. Rows that name a person by email or phone
// are linked to the person, made if need be, so an inbox export becomes
// interactions under the people they were with.
package ingest

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Table is rows of text under column names, whatever they came from.
type Table struct {
	// Source says what kind of file it was read as: csv, vcard or mbox.
	Source  string
	Columns []string
	Rows    []map[string]string
}

// Read turns a file into a table, by its name and, failing that, by what
// it holds: a vCard by its BEGIN:VCARD, a mailbox by its From lines, a
// CSV otherwise, with the delimiter it plainly uses.
func Read(name string, data []byte) (*Table, error) {
	ext := strings.ToLower(filepath.Ext(name))
	head := strings.ToUpper(string(bytes.TrimLeft(data[:min(len(data), 64)], "\xef\xbb\xbf \r\n\t")))
	switch {
	case ext == ".vcf" || ext == ".vcard" || strings.HasPrefix(head, "BEGIN:VCARD"):
		return ReadVCard(data)
	case ext == ".mbox" || ext == ".eml" || strings.HasPrefix(head, "FROM ") || strings.HasPrefix(head, "RETURN-PATH:") || strings.HasPrefix(head, "RECEIVED:"):
		return ReadMbox(data)
	case ext == ".csv" || ext == ".tsv" || ext == ".txt" || ext == "":
		return ReadCSV(data)
	}
	return nil, fmt.Errorf("%s is not a file that can be read as records: a .csv, a .vcf of contacts or a .mbox of mail is", name)
}

// ReadCSV reads a delimited file with a header row. The delimiter is the
// one the header uses most: a comma, a semicolon or a tab.
func ReadCSV(data []byte) (*Table, error) {
	text := strings.TrimPrefix(string(data), "\xef\xbb\xbf")
	first := text
	if i := strings.IndexByte(first, '\n'); i >= 0 {
		first = first[:i]
	}
	delim := ','
	for _, d := range []rune{';', '\t'} {
		if strings.Count(first, string(d)) > strings.Count(first, string(delim)) {
			delim = d
		}
	}
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = delim
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("could not read the file as a table: %w", err)
	}
	if len(records) == 0 {
		return nil, errors.New("the file is empty")
	}
	t := &Table{Source: "csv"}
	for _, c := range records[0] {
		t.Columns = append(t.Columns, strings.TrimSpace(c))
	}
	for _, rec := range records[1:] {
		row := map[string]string{}
		empty := true
		for i, c := range t.Columns {
			if i < len(rec) {
				row[c] = strings.TrimSpace(rec[i])
				if row[c] != "" {
					empty = false
				}
			}
		}
		if !empty {
			t.Rows = append(t.Rows, row)
		}
	}
	return t, nil
}

// vCardColumns are the columns a contact file is read into.
var vCardColumns = []string{"name", "email", "phone", "organisation", "role", "notes"}

// ReadVCard reads contacts (vCard 2.1, 3.0 or 4.0): the formatted name or
// the name parts, the first email and phone, the organisation, the title
// and the note. A folded line is unfolded first.
func ReadVCard(data []byte) (*Table, error) {
	text := strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n", "\n"), "\r", "\n")
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\n ", ""), "\n\t", "")
	t := &Table{Source: "vcard", Columns: vCardColumns}
	var row map[string]string
	for _, line := range strings.Split(text, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name := strings.ToUpper(key)
		if i := strings.IndexAny(name, ";"); i >= 0 {
			name = name[:i]
		}
		if strings.Contains(name, ".") {
			name = name[strings.LastIndex(name, ".")+1:]
		}
		value = strings.TrimSpace(unescapeVCard(value))
		switch name {
		case "BEGIN":
			if strings.EqualFold(value, "VCARD") {
				row = map[string]string{}
			}
		case "END":
			if row != nil && strings.EqualFold(value, "VCARD") {
				if row["name"] == "" {
					row["name"] = firstOf(row["email"], row["organisation"], row["phone"])
				}
				if row["name"] != "" {
					t.Rows = append(t.Rows, row)
				}
				row = nil
			}
		case "FN":
			if row != nil && value != "" {
				row["name"] = value
			}
		case "N":
			if row != nil && row["name"] == "" {
				parts := strings.Split(value, ";")
				var words []string
				for _, i := range []int{3, 1, 2, 0, 4} {
					if i < len(parts) && strings.TrimSpace(parts[i]) != "" {
						words = append(words, strings.TrimSpace(parts[i]))
					}
				}
				row["name"] = strings.Join(words, " ")
			}
		case "EMAIL":
			if row != nil && row["email"] == "" {
				row["email"] = value
			}
		case "TEL":
			if row != nil && row["phone"] == "" {
				row["phone"] = strings.TrimPrefix(strings.ToLower(value), "tel:")
			}
		case "ORG":
			if row != nil {
				row["organisation"] = strings.TrimSuffix(strings.ReplaceAll(value, ";", " "), " ")
			}
		case "TITLE":
			if row != nil {
				row["role"] = value
			}
		case "NOTE":
			if row != nil {
				row["notes"] = value
			}
		}
	}
	if len(t.Rows) == 0 {
		return nil, errors.New("no contacts were found in the file")
	}
	return t, nil
}

func unescapeVCard(s string) string {
	r := strings.NewReplacer(`\n`, "\n", `\N`, "\n", `\,`, ",", `\;`, ";", `\\`, `\`)
	return r.Replace(s)
}

func firstOf(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

var spaces = regexp.MustCompile(`\s+`)

// normal is a column or field name as it is compared: lower case, one
// word run, no punctuation.
func normal(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("_", " ", "-", " ", ".", " ", "/", " ").Replace(s)
	return spaces.ReplaceAllString(s, " ")
}
