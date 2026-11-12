package exchange

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/ingest"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Bring your things: one page for what a person has in another app. It
// says in a line for each app how to get its export, takes the file, and
// brings in what it holds (ingest.Recognise), tasks and notes, each kind
// as one change with its Undo, saying how many came from where. Nothing is
// asked about columns: the exports are known.

const bringLimit = 200 << 20

// bringHow is how to get each export, in the words of the app's menus.
var bringHow = []struct{ app, how string }{
	{"Your conversations with ChatGPT", "Settings, then Data controls, then Export data. A zip comes by email; bring it as it comes."},
	{"Your conversations with Claude", "Settings, then Privacy, then Export data. A zip comes by email; bring it as it comes."},
	{"Todoist", "Settings, then Backups, or a project's menu and Export as a CSV file."},
	{"Google Tasks or Google Keep", "takeout.google.com: choose Tasks or Keep, then Export once. Bring the zip as it comes."},
	{"Evernote", "select notes, then File and Export notes, as an .enex file."},
	{"Notion", "Settings, then Export all workspace content, as Markdown and CSV. Bring the zip."},
	{"Obsidian, Bear or any notes in Markdown", "zip the folder they are in, and bring the zip."},
}

func (s *Service) bringPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	b.WriteString(`<p>Tasks and notes from another app come in with their titles, words, days, tags and what is done. Undo takes them away again.</p><h2>Getting the file</h2><dl class="sw-stack">`)
	for _, h := range bringHow {
		b.WriteString(`<dt><strong>` + template.HTMLEscapeString(h.app) + `</strong></dt><dd>` + template.HTMLEscapeString(h.how) + `</dd>`)
	}
	b.WriteString(`</dl><h2>Bringing it in</h2>`)
	b.WriteString(string(s.Form(ui.Form{Action: "/bring", Enctype: "multipart/form-data", Class: "sw-stack",
		Body: `<label for="bring-file">The file from the app</label><input type="file" id="bring-file" name="file" accept=".csv,.json,.enex,.zip" required>` +
			`<label class="sw-cluster"><input type="checkbox" name="suggest" value="yes"> For conversations from ChatGPT or Claude: then look through each for what is still to do, and suggest it on Today to make, change or turn down. The assistant reads each conversation.</label>`,
		Button: &ui.Button{Label: "Bring them in"}})))
	s.Page(w, r, "Bring your things", template.HTML(b.String()), web.PageOptions{Lede: "From ChatGPT, Claude, Todoist, Google, Evernote, Notion or Obsidian."})
}

func (s *Service) bring(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, bringLimit)
	f, hdr, err := r.FormFile("file")
	if err != nil {
		why := errors.New("choose the file the app made")
		// A ChatGPT export with years of pictures in it is often bigger.
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			why = errors.New("this file is bigger than 200 MB, more than Sameway takes at once. From ChatGPT or Claude, open the zip and bring the conversations.json in it")
		}
		s.Failed(w, r, "Nothing brought in", why, "/bring")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		s.Failed(w, r, "Nothing brought in", err, "/bring")
		return
	}
	got, err := ingest.Recognise(hdr.Filename, data, s.Now().Location())
	if err != nil {
		s.Failed(w, r, "Nothing brought in", err, "/bring")
		return
	}
	fromChat := got.From == "ChatGPT" || got.From == "Claude"
	suggesting := r.FormValue("suggest") == "yes" && fromChat
	if suggesting {
		s.suggestFromChats(r, strings.ToLower(got.From)) // bring_chats.go: before, so each note sets it off
	}
	var said []string
	to, already, chats, carried := "/", 0, 0, 0
	for _, kind := range []string{"task", "note"} {
		tb := got.Kinds[kind]
		t, ok := s.app.Types.Get(kind)
		if tb == nil || !ok {
			continue
		}
		already += tb.Unseen(s.app.Store, t)
		if kind == "note" && fromChat {
			carried = s.carryOn(t, tb.Continued(s.app.Store, t), got.From) // bring_chats.go
		}
		report := ingest.Import(s.app.Store, t, tb, ingest.Guess(t, tb.Columns))
		if report.Made == 0 {
			continue
		}
		records.Record(s.app.Store, "human", records.Change{Action: "imported", Component: t.Name,
			Detail: fmt.Sprintf("%d %s from %s", report.Made, schema.Plural(t.Name), got.From), Href: "/t/" + t.Name, Ops: records.Made(s.app.Store, t.Name, report.IDs)})
		said = append(said, fmt.Sprintf("%d %s", report.Made, schema.Plural(t.Name)))
		if kind == "note" {
			chats = report.Made
		}
		if to == "/" {
			to = "/t/" + t.Name
		}
	}
	if carried > 0 {
		said = append(said, fmt.Sprintf("%d carried on since, each in the note brought before", carried))
	}
	if len(said) == 0 && already > 0 {
		s.TellAt(w, r, web.Outcome{Title: "Nothing new from " + got.From, Text: "Everything in the file was brought in before."}, "/bring")
		return
	}
	if len(said) == 0 {
		s.Failed(w, r, "Nothing brought in", errors.New("the file from "+got.From+" had nothing in it to bring"), "/bring")
		return
	}
	text := strings.Join(said, " and ") + ". Undo in Activity takes each kind away again."
	if already > 0 {
		text += fmt.Sprintf(" %d brought in before were left out.", already)
	}
	if fromChat {
		more, list := s.aboutChats(got.From, suggesting, chats)
		text, to = text+more, list
	}
	s.TellAt(w, r, web.Outcome{Title: "Brought in from " + got.From, Text: text}, to)
}

// aboutChats is what the outcome says of conversations brought, and the
// list of them. Each is a note tagged with where it came from; what was
// planned in them is still words until the assistant makes it tasks, days
// and people (chat/rules_topical.go).
func (s *Service) aboutChats(from string, suggesting bool, chats int) (string, string) {
	tag := strings.ToLower(from)
	text := " Each conversation is a note tagged " + tag + ". Ask the assistant to find what is still to do in them, and it makes the tasks, reminders and people."
	if suggesting && chats > 0 {
		text += s.lookingThrough(chats)
	}
	return text, "/t/note?where=tags%3D" + tag
}

// lookingThrough says what asking to look through the conversations sets
// going: how many, and that a model on this computer takes a while over
// them. It is asked for plainly and Undo takes it back, so it is said, not
// confirmed or capped.
func (s *Service) lookingThrough(chats int) string {
	text := fmt.Sprintf(" The assistant looks through each of the %d conversations for what is still to do, and puts it on Today to make, change or turn down.", chats)
	if chats == 1 {
		text = " The assistant looks through the conversation for what is still to do, and puts it on Today to make, change or turn down."
	}
	switch cfg := s.app.LLMConfig(); {
	case cfg.Provider == "" || cfg.Provider == "none":
		return text + " It starts once an AI model is connected."
	case cfg.OnThisComputer():
		return text + " The model is on this computer, so this takes a while: a long conversation can take minutes."
	}
	return text + " This uses the AI account you connected, so they are read there."
}

// ImportLinks are a list's ways in from a file: Import for any file, and
// for tasks and notes, the apps Bring your things knows.
func (s *Service) ImportLinks(t *schema.Type) string {
	if !Importable(t) {
		return ""
	}
	out := `<p class="sw-quiet-row">` + string(s.Part(ui.Link{Href: "/t/" + t.Name + "/import", Label: "Import", Context: schema.Plural(t.Name), Look: ui.LookButton}))
	if t.Name == "task" || t.Name == "note" {
		out += " " + string(s.Part(ui.Link{Href: "/bring", Label: "Bring yours from another app"}))
	}
	return out + `</p>`
}
