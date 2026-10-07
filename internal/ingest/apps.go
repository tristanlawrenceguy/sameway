package ingest

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// A person comes to Sameway with their things somewhere else, and the
// import a list page has asks which kind of record a spreadsheet's
// columns are and which column is which: a question nobody who just
// exported from Todoist can answer. Recognise knows the exports people
// have, by what is in them: Todoist's CSV, Google Tasks and Google Keep
// from Takeout, Evernote's .enex, and any zip of Markdown (Notion,
// Obsidian, Bear), and reads each into the kinds Sameway has, tasks and
// notes, the columns already named as the fields are (apps_zip.go has
// Evernote and the zips).

// Brought is what one app's export holds, by kind.
type Brought struct {
	// From names the app, as the person knows it.
	From string
	// Kinds are tables by the kind of record they become: "task", "note".
	Kinds map[string]*Table
}

// Count is how many of a kind came.
func (b *Brought) Count(kind string) int {
	if t := b.Kinds[kind]; t != nil {
		return len(t.Rows)
	}
	return 0
}

var errUnknown = errors.New("this is not an export Sameway knows: it reads the CSV Todoist exports, Google Tasks and Google Keep from Google Takeout, the .enex Evernote exports, and a zip of Markdown notes from Notion, Obsidian or elsewhere")

// Recognise reads an export by what it holds.
func Recognise(name string, data []byte) (*Brought, error) {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".enex"):
		return evernote(data)
	case strings.HasSuffix(lower, ".zip"):
		return zipped(data)
	case strings.HasSuffix(lower, ".json"):
		return googleJSON(data)
	case strings.HasSuffix(lower, ".csv"):
		return todoist(data)
	}
	return nil, errUnknown
}

func newBrought(from string) *Brought { return &Brought{From: from, Kinds: map[string]*Table{}} }

func (b *Brought) add(kind string, row map[string]string) {
	t := b.Kinds[kind]
	if t == nil {
		t = &Table{Source: b.From}
		b.Kinds[kind] = t
	}
	for col := range row {
		if !contains(t.Columns, col) {
			t.Columns = append(t.Columns, col)
		}
	}
	t.Rows = append(t.Rows, row)
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// todoist reads the CSV Todoist exports: its task rows become tasks, with
// the date as Todoist wrote it ("tomorrow", "every Monday", "22 Oct").
func todoist(data []byte) (*Brought, error) {
	tb, err := ReadCSV(data)
	if err != nil || !contains(lowered(tb.Columns), "content") || !contains(lowered(tb.Columns), "type") {
		return nil, errUnknown
	}
	b := newBrought("Todoist")
	for _, r := range tb.Rows {
		get := func(col string) string { return cellAny(r, col) }
		if strings.ToLower(get("type")) != "task" || get("content") == "" {
			continue
		}
		row := map[string]string{"title": get("content"), "notes": get("description")}
		if d := get("date"); d != "" {
			row["due"] = d
		}
		b.add("task", row)
	}
	if len(b.Kinds) == 0 {
		return nil, errors.New("this Todoist export has no tasks in it")
	}
	return b, nil
}

func lowered(cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = strings.ToLower(c)
	}
	return out
}

func cellAny(row map[string]string, col string) string {
	for k, v := range row {
		if strings.EqualFold(k, col) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// googleJSON reads Google Tasks (the Tasks.json in Takeout) or one note
// from Google Keep.
func googleJSON(data []byte) (*Brought, error) {
	var tasks struct {
		Items []struct {
			Kind  string `json:"kind"`
			Title string `json:"title"`
			Items []struct {
				Title  string `json:"title"`
				Notes  string `json:"notes"`
				Due    string `json:"due"`
				Status string `json:"status"`
			} `json:"items"`
		} `json:"items"`
	}
	if json.Unmarshal(data, &tasks) == nil && len(tasks.Items) > 0 && strings.HasPrefix(tasks.Items[0].Kind, "tasks#") {
		b := newBrought("Google Tasks")
		for _, list := range tasks.Items {
			for _, t := range list.Items {
				if strings.TrimSpace(t.Title) == "" {
					continue
				}
				row := map[string]string{"title": t.Title, "notes": t.Notes, "done": boolWord(t.Status == "completed")}
				if d, err := time.Parse(time.RFC3339, t.Due); err == nil {
					row["due"] = d.Format("2006-01-02")
				}
				if list.Title != "" && list.Title != "My Tasks" {
					row["tags"] = list.Title
				}
				b.add("task", row)
			}
		}
		return b, nil
	}
	b := newBrought("Google Keep")
	if keepNote(b, data) {
		return b, nil
	}
	return nil, errUnknown
}

func boolWord(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// keepNote reads one note from Google Keep into b, saying whether it was
// one; a note in the bin is left out.
func keepNote(b *Brought, data []byte) bool {
	var n struct {
		Title       string  `json:"title"`
		TextContent *string `json:"textContent"`
		ListContent []struct {
			Text      string `json:"text"`
			IsChecked bool   `json:"isChecked"`
		} `json:"listContent"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		IsTrashed bool `json:"isTrashed"`
	}
	if json.Unmarshal(data, &n) != nil || n.TextContent == nil && n.ListContent == nil {
		return false
	}
	if n.IsTrashed {
		return true
	}
	body := ""
	if n.TextContent != nil {
		body = *n.TextContent
	}
	for _, item := range n.ListContent {
		mark := " "
		if item.IsChecked {
			mark = "x"
		}
		body += "\n- [" + mark + "] " + item.Text
	}
	title := n.Title
	if title == "" {
		title = firstLine(body)
	}
	var tags []string
	for _, l := range n.Labels {
		tags = append(tags, l.Name)
	}
	b.add("note", map[string]string{"title": title, "body": strings.TrimSpace(body), "tags": strings.Join(tags, ", ")})
	return true
}

func firstLine(s string) string {
	s = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(s), "#-[] x"))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if r := []rune(s); len(r) > 60 {
		s = string(r[:60]) + "…"
	}
	if s == "" {
		return "Untitled"
	}
	return s
}
