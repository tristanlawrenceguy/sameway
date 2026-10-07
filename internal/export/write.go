package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Write takes records of a type out in a format.
func Write(w io.Writer, f Format, t *schema.Type, recs []*store.Record, titles Titles) error {
	switch f.Ext {
	case "csv":
		return writeCSV(w, t, recs, titles)
	case "xlsx":
		return writeXLSX(w, t, recs, titles)
	case "vcf":
		return writeVCard(w, t, recs, titles)
	case "ics":
		return writeICS(w, t, recs)
	}
	return fmt.Errorf("%s cannot be taken out as .%s", t.Name, f.Ext)
}

// A CSV starts with the byte-order mark, so Excel reads its accents.
func writeCSV(w io.Writer, t *schema.Type, recs []*store.Record, titles Titles) error {
	fields := Fields(t)
	w.Write([]byte{0xEF, 0xBB, 0xBF}) // the byte-order mark
	c := csv.NewWriter(w)
	c.UseCRLF = true // RFC 4180
	c.Write(Header(fields))
	rows := Rows(fields, recs, titles)
	for _, row := range rows {
		for i, v := range row {
			if fields[i].Type != "int" && fields[i].Type != "float" {
				row[i] = Inert(v)
			}
		}
	}
	c.WriteAll(rows)
	c.Flush()
	return c.Error()
}

// Inert is a cell a spreadsheet will not run: text that starts as a
// formula does (=, +, -, @, a tab or a return) goes out with a ' before
// it, which a spreadsheet shows as text (OWASP, CSV injection). Import
// takes it off again. Numbers are written as numbers, never through here.
func Inert(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

// An Excel sheet has its header in bold, kept in sight as it scrolls,
// numbers as numbers, and columns as wide as what they hold.
func writeXLSX(w io.Writer, t *schema.Type, recs []*store.Record, titles Titles) error {
	fields := Fields(t)
	x := excelize.NewFile()
	defer x.Close()
	sheet := "Sheet1"
	name := strings.ToUpper(t.Name[:1]) + t.Name[1:]
	if len(name) <= 31 {
		x.SetSheetName(sheet, name)
		sheet = name
	}
	bold, _ := x.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	widths := make([]int, len(fields))
	for i, h := range Header(fields) {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		x.SetCellValue(sheet, cell, h)
		x.SetCellStyle(sheet, cell, cell, bold)
		widths[i] = len(h)
	}
	for r, row := range Rows(fields, recs, titles) {
		for i, v := range row {
			cell, _ := excelize.CoordinatesToCellName(i+1, r+2)
			switch fields[i].Type {
			case "int", "float":
				if n, err := strconv.ParseFloat(v, 64); err == nil {
					x.SetCellValue(sheet, cell, n)
					continue
				}
			}
			x.SetCellValue(sheet, cell, v)
			widths[i] = max(widths[i], min(len(v), 60))
		}
	}
	for i, wd := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		x.SetColWidth(sheet, col, col, float64(wd+2))
	}
	x.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	return x.Write(w)
}

// vCard 3.0, which every address book reads.
func writeVCard(w io.Writer, t *schema.Type, recs []*store.Record, titles Titles) error {
	get := func(r *store.Record, name string) string {
		f, ok := t.Field(name)
		if !ok {
			return ""
		}
		return Value(*f, r.Fields[name], titles)
	}
	for _, r := range recs {
		name := get(r, t.Title)
		lines := []string{"BEGIN:VCARD", "VERSION:3.0", "FN:" + esc(name), "N:" + nameParts(name)}
		for _, p := range []struct{ field, prop string }{{"email", "EMAIL;TYPE=INTERNET"}, {"phone", "TEL"}, {"organisation", "ORG"}, {"role", "TITLE"}, {"notes", "NOTE"}, {"tags", "CATEGORIES"}} {
			if v := get(r, p.field); v != "" {
				if p.field == "tags" {
					lines = append(lines, p.prop+":"+v)
				} else {
					lines = append(lines, p.prop+":"+esc(v))
				}
			}
		}
		lines = append(lines, "UID:"+r.ID+"@sameway", "END:VCARD")
		if err := writeLines(w, lines); err != nil {
			return err
		}
	}
	return nil
}

// nameParts is a name as vCard's N: family name first.
func nameParts(name string) string {
	parts := strings.Fields(name)
	if len(parts) < 2 {
		return esc(name) + ";;;;"
	}
	return esc(parts[len(parts)-1]) + ";" + esc(strings.Join(parts[:len(parts)-1], " ")) + ";;;"
}

// iCalendar: each record an event at its day or time, repeating as it
// does, with its own id so a calendar that takes it again keeps one.
func writeICS(w io.Writer, t *schema.Type, recs []*store.Record) error {
	return Calendar(w, strings.ToUpper(t.Name[:1])+t.Name[1:], []Group{{t, recs}})
}

// A Group is the records of one type.
type Group struct {
	Type    *schema.Type
	Records []*store.Record
}

// Calendar writes one calendar of the records of several types, each on
// its day, such as everything on the calendar at once.
func Calendar(w io.Writer, name string, groups []Group) error {
	lines := []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//sameway//sameway//EN", "CALSCALE:GREGORIAN", "X-WR-CALNAME:" + esc(name)}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	for _, g := range groups {
		lines = append(lines, events(g.Type, g.Records, stamp)...)
	}
	lines = append(lines, "END:VCALENDAR")
	return writeLines(w, lines)
}

// events are a type's records as VEVENTs, those with a day.
func events(t *schema.Type, recs []*store.Record, stamp string) []string {
	start := t.DayField()
	var lines []string
	for _, r := range recs {
		when, _ := r.Fields[start].(string)
		if when == "" {
			continue
		}
		uid, _ := r.Fields["uid"].(string)
		if uid == "" {
			uid = r.ID + "@sameway"
		}
		lines = append(lines, "BEGIN:VEVENT", "UID:"+uid, "DTSTAMP:"+stamp, "SUMMARY:"+esc(t.Called(r.ID, r.Fields)), "DTSTART"+icsTime(when))
		allDay := strings.HasPrefix(icsTime(when), ";VALUE=DATE")
		if end, _ := r.Fields["ends"].(string); end != "" && start == "starts" {
			lines = append(lines, "DTEND"+icsEnd(end, allDay))
		}
		if rule, _ := r.Fields["repeat"].(string); rule != "" {
			lines = append(lines, "RRULE:"+icsRule(rule, allDay))
		}
		if where, _ := r.Fields["where"].(string); where != "" {
			lines = append(lines, "LOCATION:"+esc(where))
		}
		if notes, _ := r.Fields["notes"].(string); notes != "" {
			lines = append(lines, "DESCRIPTION:"+esc(notes))
		}
		if t.Done(r.Fields) {
			lines = append(lines, "STATUS:CONFIRMED", "X-SAMEWAY-DONE:TRUE")
		}
		lines = append(lines, "END:VEVENT")
	}
	return lines
}

// icsTime is a stored time as DTSTART takes it: a day alone, or UTC.
func icsTime(v string) string {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		if d, err := time.Parse("2006-01-02", v); err == nil {
			return ";VALUE=DATE:" + d.Format("20060102")
		}
		return ":" + v
	}
	if strings.HasSuffix(v, "T00:00:00Z") {
		return ";VALUE=DATE:" + t.UTC().Format("20060102")
	}
	return ":" + t.UTC().Format("20060102T150405Z")
}

// icsEnd is an end as DTEND takes it. An all-day end is the day after the
// last day, since DTEND is not part of the event (RFC 5545 3.6.1), so an
// event that ends on the 13th ends on the 14th in the file.
func icsEnd(v string, allDay bool) string {
	out := icsTime(v)
	if day, ok := strings.CutPrefix(out, ";VALUE=DATE:"); ok && allDay {
		if d, err := time.Parse("20060102", day); err == nil {
			return ";VALUE=DATE:" + d.AddDate(0, 0, 1).Format("20060102")
		}
	}
	return out
}

// icsRule is a repeat as RRULE takes it: UNTIL must be the same kind of
// value as the start (RFC 5545 3.3.10), so a timed event's last day runs
// to its end, in UTC.
func icsRule(rule string, allDay bool) string {
	if allDay {
		return rule
	}
	parts := strings.Split(rule, ";")
	for i, p := range parts {
		if day, ok := strings.CutPrefix(p, "UNTIL="); ok && len(day) == 8 {
			parts[i] = "UNTIL=" + day + "T235959Z"
		}
	}
	return strings.Join(parts, ";")
}

func esc(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`).Replace(s)
}

// writeLines writes lines as vCard and iCalendar want them: CRLF, and
// folded at 75 bytes, never inside a character.
func writeLines(w io.Writer, lines []string) error {
	var b strings.Builder
	for _, l := range lines {
		for len(l) > 75 {
			cut := 75
			for cut > 0 && l[cut]&0xC0 == 0x80 {
				cut--
			}
			b.WriteString(l[:cut] + "\r\n ")
			l = l[cut:]
		}
		b.WriteString(l + "\r\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}
