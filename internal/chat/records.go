package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/atlogin"
	"github.com/tristanlawrenceguy/sameway/internal/ingest"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Records are the person's content: the notes, the tasks, whatever the
// workspace's schema declares. The canvas tools place components; these
// tools make and change the things a person then finds on /t/<type>. They
// are generated from the schema, so a workspace that adds a content type
// gives the assistant the tool for it without a line of code.

// contentTypes lists the types the assistant may write to: everything the
// schema declares except the system's own (messages, blocks, activity).
func (s *Service) contentTypes() []*schema.Type {
	var out []*schema.Type
	for _, t := range s.Store.Types().Types {
		switch {
		case !t.Content(), t.Name == MessageType, t.Name == BlockType, t.Name == ActivityType:
			continue
		}
		out = append(out, t)
	}
	return out
}

func (s *Service) typeNames() []string {
	var names []string
	for _, t := range s.contentTypes() {
		names = append(names, t.Name)
	}
	sort.Strings(names)
	return names
}

// recordTools are offered only when the workspace has something to write to.
func (s *Service) recordTools() []llm.Tool {
	names := s.typeNames()
	if len(names) == 0 {
		return nil
	}
	typeArg := map[string]any{"type": "string", "enum": names, "description": "A content type, as the prompt lists it."}
	return []llm.Tool{
		{Name: "import_records", Description: "Make records from a file the person added: a CSV with a header row, a vCard (.vcf) of contacts, or a mailbox (.mbox) of mail. Each column is matched to a field by name; a column for an email, phone or name links each row to its person, made when new. Use it when the person attaches such a file and wants its contents as records, rather than creating them one by one. Returns how many were made.",
			Schema: obj(map[string]any{
				"type":    typeArg,
				"file":    map[string]any{"type": "string", "description": "The id of the file record, from the message it came with or from find_records on file."},
				"mapping": map[string]any{"type": "object", "description": "Optional: which column feeds which field, as {column: field}. Leave out to match by name.", "additionalProperties": map[string]any{"type": "string"}},
			}, "type", "file")},
		{Name: "create_record", Description: "Make a record of a content type: a note, a task, whatever the workspace declares. It appears on its own page at /t/<type> and in the listing there. Fields must match the type's schema in the catalogue. Returns the new record's id and page.",
			Schema: obj(map[string]any{
				"type":   typeArg,
				"fields": map[string]any{"type": "object", "description": "Field values matching the type's schema. Leave a field out to take its default."},
			}, "type", "fields")},
		{Name: "update_record", Description: "Change fields on a record that exists. Only the fields given change. Use find_records first to get the id.",
			Schema: obj(map[string]any{
				"type":    typeArg,
				"id":      map[string]any{"type": "string", "description": "The record's id, from find_records or from a page URL /t/<type>/<id>."},
				"fields":  map[string]any{"type": "object", "description": "The fields to change and their new values."},
				"version": map[string]any{"type": "string", "description": "The version get_record gave, when you read the record first: if it has changed since, nothing is written and you are shown it as it is now, to change again."},
			}, "type", "id", "fields")},
		{Name: "find_records", Description: "List records of a type to get their ids: all of them, those holding every word of the query in their title or words, or those matching where. The same where and order a collection block takes.",
			Schema: obj(map[string]any{
				"type":  typeArg,
				"query": map[string]any{"type": "string", "description": "Words each record found must hold, in its title or its words, in any order. Leave empty for every record."},
				"where": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Conditions that must all hold. " + query.Grammar},
				"order": map[string]any{"type": "string", "description": "A field, or -field for the largest or newest first. Newest first when left out."},
				"limit": map[string]any{"type": "integer", "description": "How many to list. Defaults to 10."},
			}, "type")},
		{Name: "get_record", Description: "Read one record with every field, by id: a note's body, a file's text. Use it before answering from what a record says. It also returns related: everything the record is joined to — what points at it, what is set about it, what sits beside it under the same parent, what else falls on its day — each with a count and the where that lists them. Their page shows only the counts. When you have a reason to put one in front of the person, send them the page with that connection open: /t/<type>/<id>?show=<key>.",
			Schema: obj(map[string]any{
				"type": typeArg,
				"id":   map[string]any{"type": "string", "description": "The record's id, from find_records or from a page URL /t/<type>/<id>."},
			}, "type", "id")},
	}
}

func (s *Service) contentType(name string) (*schema.Type, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, t := range s.contentTypes() {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, fmt.Errorf("unknown content type %q. The workspace has: %s", name, strings.Join(s.typeNames(), ", "))
}

func (s *Service) createRecord(typeName string, fields map[string]any) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if fields == nil {
		fields = map[string]any{}
	}
	rec, c, err := Write(s.Store, "created", t.Name, "", fields)
	if err != nil {
		return fail("I couldn't save those changes — %s. %s Fix the fields and call create_record again.", humanizeValidationError(err.Error()), typeHelp(t))
	}
	text := fmt.Sprintf("created %s %s: %q. The person can open it at /t/%s/%s.", t.Name, rec.ID, c.Detail, t.Name, rec.ID)
	text += s.datesSaid(t, fields, rec, false) + s.sameTitle(t, rec) + s.whoseSaid(t, rec) + s.actionSaid(t, rec) + s.splitSaid(t) // dates_said.go, same_title.go, whose_said.go, action_when_said.go
	if t.Name == "reminder" && atlogin.Path() != "" && !atlogin.On() {
		text += " Reminders ring only while Sameway is open, and it does not open when this computer starts; if this one matters, tell the person that Open Sameway when I sign in, on Workspaces, keeps it ringing."
	}
	if t.Name == EventType && len(s.recordingTools()) > 0 {
		text += fmt.Sprintf(" If the person wants this meeting recorded, call the record_meeting tool yourself now with event %s (how app when Teams, Zoom or Meet records it): it sets up the reminder that opens the page ready to record. The tools are yours; never tell the person to use them.", rec.ID)
	}
	return toolResult{text: text, change: &c}
}

func (s *Service) updateRecord(typeName, id string, fields map[string]any, version string) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if len(fields) == 0 {
		return fail("nothing to change: pass the fields to change and their new values")
	}
	was, err := s.Store.Get(t.Name, id)
	if err != nil {
		return fail("no %s with id %s. Use find_records to get the id", t.Name, id)
	}
	if r, ok := s.whichOne(t, was); !ok { // which_one.go
		return r
	}
	if version != "" && !SameVersion(was, version) {
		now, _ := json.Marshal(was.Fields)
		return fail("%s %s has changed since version %s, so nothing was written. As it is now (version %s): %s. Make your change to this and send it with the new version", t.Name, id, version, Version(was), now)
	}
	rec, c, err := Write(s.Store, "updated", t.Name, id, fields)
	if err != nil {
		return fail("I couldn't save those changes — %s. %s Fix the fields and call update_record again.", humanizeValidationError(err.Error()), typeHelp(t))
	}
	title := c.Detail
	// Finished, a thing that repeats is due again at once; say so, or the
	// assistant reads its own tick as undone.
	again := ""
	if repeat, day, ok := t.Repeats(); ok && t.Advanced(fields, rec.Fields) {
		again = fmt.Sprintf(" It repeats (%s), so it is not finished but due again at %v.", when.RepeatText(fmt.Sprint(rec.Fields[repeat])), rec.Fields[day])
	}
	return toolResult{
		text:   fmt.Sprintf("updated %s %s: %q, at /t/%s/%s.%s", t.Name, rec.ID, title, t.Name, rec.ID, again) + s.datesSaid(t, fields, rec, len(daysOf(t, was, s.clock())) > 0) + timeLost(t, fields, was, rec),
		change: &c,
	}
}

func (s *Service) findRecords(typeName, words string, where []string, order string, limit int) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if limit <= 0 {
		limit = 10
	}
	recs, err := query.Filter(s.Store, t, where, order, 0, time.Now())
	if err != nil {
		return fail("%v", err)
	}
	words = strings.ToLower(strings.TrimSpace(words))
	days := search.DaysAsked(words, s.clock()) // a day asked for finds what falls on it
	writers := s.Writers()
	var lines []string
	for _, rec := range recs {
		title := recordTitle(s.Store, t, rec)
		if words != "" && !holdsAll(title, rec, words) {
			if _, on := search.FallsOn(t, rec, days, s.clock()); !on {
				continue
			}
		}
		line := fmt.Sprintf("%s\t%s\t%s", rec.ID, oneLine(title), writers.Of(t.Name, rec).Words)
		if days := daysOf(t, rec, s.clock()); len(days) > 0 { // days_shown.go
			line += "\t" + strings.Join(days, "; ")
		}
		lines = append(lines, line)
		if len(lines) == limit {
			break
		}
	}
	if len(lines) == 0 {
		if words == "" && len(where) == 0 {
			return toolResult{text: fmt.Sprintf("there are no %s records yet", t.Name)}
		}
		said := strings.TrimSpace(strings.Join([]string{query.Words(t, where), words}, " "))
		if len(where) == 0 {
			if all := s.noneOfKind(t.Name); all != "" { // search_none.go
				return toolResult{text: fmt.Sprintf("no %s has the words %s.", t.Name, said) + all}
			}
		}
		return toolResult{text: fmt.Sprintf("no %s matches %s. Leave out query to list them all and judge by their titles; search finds words in every kind at once.", t.Name, said)}
	}
	// The titles are fenced, each line saying who wrote it; see provenance.go.
	matching := ""
	if len(where) > 0 {
		matching = " " + query.Words(t, where)
	}
	return toolResult{text: fmt.Sprintf("%s records%s, newest first (id, title, written by, and its days in this computer's time). Each title was written by the one on its line; %s.\n<<<record text\n%s\nrecord text>>>", t.Name, matching, Untrusted, strings.Join(lines, "\n"))}
}

// deleteRecord is not a tool: a record goes when a person deletes it, or
// when its creation is undone. Either way the log keeps what it was.
func (s *Service) deleteRecord(typeName, id string) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if _, err := s.Store.Get(t.Name, id); err != nil {
		return fail("no %s with id %s", t.Name, id)
	}
	_, c, err := Write(s.Store, "deleted", t.Name, id, nil)
	if err != nil {
		return fail("could not delete %s %s: %v", t.Name, id, err)
	}
	return toolResult{text: fmt.Sprintf("deleted %s %s", t.Name, id), change: &c}
}

// importRecords makes records of a type from a kept file, through ingest,
// and says how it went. The file is the original the person added.
func (s *Service) importRecords(typeName, fileID string, mapping map[string]any) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	file, err := s.Store.Get(FileType, fileID)
	if err != nil {
		return fail("no file with id %s; the id is on the message the file came with, or find_records on file", fileID)
	}
	stored, _ := file.Fields["path"].(string)
	name, _ := file.Fields["name"].(string)
	if stored == "" || strings.ContainsAny(stored, `/\`) || Workdir == "" {
		return fail("the file %s has no original kept to read", fileID)
	}
	data, err := os.ReadFile(filepath.Join(Workdir, "files", stored))
	if err != nil {
		return fail("could not read the file: %v", err)
	}
	tb, err := ingest.Read(name, data)
	if err != nil {
		return fail("%v", err)
	}
	m := ingest.Mapping{}
	for col, f := range mapping {
		if field, ok := f.(string); ok {
			m[col] = field
		}
	}
	if len(m) == 0 {
		m = ingest.Guess(t, tb.Columns)
	}
	report := ingest.Import(s.Store, t, tb, m)
	title := fmt.Sprintf("%d %s from %s", report.Made, schema.Plural(t.Name), name)
	return toolResult{
		text:   fmt.Sprintf("%s: %s. The person can see them at /t/%s.", t.Name, report.String(), t.Name),
		change: &Change{Action: "imported", Component: t.Name, Detail: title, Href: "/t/" + t.Name, Before: Imported(t.Name, report.IDs)},
	}
}
