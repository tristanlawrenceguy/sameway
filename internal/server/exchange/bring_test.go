package exchange_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A person brings their tasks from Todoist with the file the app made: no
// question about columns, the tasks come in with their words and days,
// it says how many from where, and Undo takes them away again.
func TestThingsAreBroughtFromAnotherApp(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	if page := get(t, h, "/bring").Body.String(); !strings.Contains(page, "Todoist") || !strings.Contains(page, `name="file"`) {
		t.Fatalf("the page says how to get each export and takes the file: %s", truncate(page))
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "Inbox.csv")
	fw.Write([]byte("TYPE,CONTENT,DESCRIPTION,PRIORITY,INDENT,AUTHOR,RESPONSIBLE,DATE,DATE_LANG,TIMEZONE\ntask,Call the bank,About the card,4,1,Me,,2026-10-12,en,UTC\ntask,Water plants,,1,1,Me,,,en,UTC\n"))
	mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/bring", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Origin", "http://example.com")
	r.Host = "example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	page := after(t, h, w).Body.String()
	if !strings.Contains(page, "Brought in from Todoist") || !strings.Contains(page, "2 tasks") {
		t.Fatalf("it says how many from where: %d %s", w.Code, truncate(page))
	}
	tasks, _ := a.Store.List("task", store.ListOptions{})
	if len(tasks) != 2 {
		t.Fatalf("the tasks came in: %d", len(tasks))
	}
	log, _ := a.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if len(log) != 1 || !strings.Contains(records.Sentence(a.Store, log[0].Fields), "2 tasks from Todoist") {
		t.Fatalf("as one change: %v", log)
	}
	postForm(t, h, "/activity/"+log[0].ID+"/undo", nil)
	if left, _ := a.Store.List("task", store.ListOptions{}); len(left) != 0 {
		t.Errorf("Undo takes them away again: %d left", len(left))
	}
}

// Conversations exported from ChatGPT come in as notes, one each, tagged
// chatgpt, and the person is told the assistant can make what is still to
// do in them into tasks, reminders and people.
func TestConversationsAreBroughtFromChatGPT(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	if page := get(t, h, "/bring").Body.String(); !strings.Contains(page, "Your conversations with ChatGPT") || !strings.Contains(page, "Your conversations with Claude") {
		t.Fatalf("the page says how to get the chats: %s", truncate(page))
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "conversations.json")
	fw.Write([]byte(`[{"title":"Moving house","create_time":1760000000,"current_node":"b","mapping":{"a":{"parent":null,"message":{"author":{"role":"user"},"content":{"parts":["Remind me to book the van for the 14th"]}}},"b":{"parent":"a","message":{"author":{"role":"assistant"},"content":{"parts":["Noted: book the van."]}}}}}]`))
	mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/bring", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Origin", "http://example.com")
	r.Host = "example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	page := after(t, h, w).Body.String()
	if !strings.Contains(page, "Brought in from ChatGPT") || !strings.Contains(page, "Ask the assistant to find what is still to do in them") {
		t.Fatalf("it says what came and what next: %d %s", w.Code, truncate(page))
	}
	notes, _ := a.Store.List("note", store.ListOptions{})
	if len(notes) != 1 || notes[0].Fields["title"] != "Moving house" || !strings.Contains(notes[0].Fields["body"].(string), "book the van") {
		t.Errorf("the conversation is a note: %+v", notes)
	}
}

// bringChats posts a ChatGPT export of one conversation, said, to Bring
// your things, asking for suggestions or not.
func bringChats(t *testing.T, h http.Handler, suggest bool, said string) string {
	t.Helper()
	return bringExport(t, h, suggest, `[{"title":"Moving house","create_time":1760000000,"current_node":"a","mapping":{"a":{"parent":null,"message":{"author":{"role":"user"},"content":{"parts":["`+said+`"]}}}}}]`)
}

// bringExport posts conversations.json to Bring your things.
func bringExport(t *testing.T, h http.Handler, suggest bool, export string) string {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if suggest {
		mw.WriteField("suggest", "yes")
	}
	fw, _ := mw.CreateFormFile("file", "conversations.json")
	fw.Write([]byte(export))
	mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/bring", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Origin", "http://example.com")
	r.Host = "example.com"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return after(t, h, w).Body.String()
}

// Asked to, Sameway looks through conversations brought for what is still
// to do: an action set up once, before they come in, that suggests a task
// from each on Today; not asked, no action.
func TestConversationsBroughtCanBeLookedThroughForWhatIsToDo(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	if page := get(t, h, "/bring").Body.String(); !strings.Contains(page, `name="suggest"`) {
		t.Fatalf("the page offers to look through them: %s", truncate(page))
	}
	if page := bringChats(t, h, false, "Book the van"); strings.Contains(page, "puts it on Today") {
		t.Errorf("not asked, nothing is said of it: %s", truncate(page))
	}
	suggesting := func() int {
		n := 0
		acts, _ := a.Store.List(records.ActionType, store.ListOptions{})
		for _, x := range acts {
			if x.Fields["kind"] == "suggest" && x.Fields["what"] == "note" {
				n++
			}
		}
		return n
	}
	if suggesting() != 0 {
		t.Fatal("not asked, no action")
	}
	// Asked, it is not confirmed: it says how many are looked through, and
	// where what is to do comes.
	if page := bringChats(t, h, true, "Ring the vet"); !strings.Contains(page, "looks through the conversation for what is still to do, and puts it on Today") {
		t.Errorf("asked, it says what is looked through and where the suggestions come: %s", truncate(page))
	}
	bringChats(t, h, true, "Pay the gas bill")
	if n := suggesting(); n != 1 {
		t.Errorf("one action, set up once: %d", n)
	}
}

// An export holds all of a person's conversations each time: brought
// again, only what is new comes in, and when nothing is, it says so.
func TestConversationsBroughtAgainComeInOnce(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	bringChats(t, h, false, "Book the van")
	if page := bringChats(t, h, false, "Book the van"); !strings.Contains(page, "Nothing new from ChatGPT") || !strings.Contains(page, "Everything in the file was brought in before") {
		t.Errorf("brought again, it says nothing was new: %s", truncate(page))
	}
	if notes, _ := a.Store.List("note", store.ListOptions{}); len(notes) != 1 {
		t.Errorf("one note, not two: %d", len(notes))
	}
}

// A conversation carried on since it was brought is longer in the next
// export: the note brought before takes the new words, as one change in
// Activity that Undo takes back, not a second note beside it.
func TestAConversationCarriedOnUpdatesItsNote(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	bringChats(t, h, false, "Book the van")
	page := bringExport(t, h, false, `[{"title":"Moving house","create_time":1760000000,"current_node":"b","mapping":{
		"a":{"parent":null,"message":{"author":{"role":"user"},"content":{"parts":["Book the van"]}}},
		"b":{"parent":"a","message":{"author":{"role":"assistant"},"content":{"parts":["Done: the van is for the 14th."]}}}}}]`)
	if !strings.Contains(page, "1 carried on since, each in the note brought before") {
		t.Errorf("it says the conversation was carried on: %s", truncate(page))
	}
	notes, _ := a.Store.List("note", store.ListOptions{})
	if len(notes) != 1 || !strings.Contains(notes[0].Fields["body"].(string), "the van is for the 14th") {
		t.Fatalf("one note, with the new words: %+v", notes)
	}
	log, _ := a.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	if len(log) != 1 || !strings.Contains(records.Sentence(a.Store, log[0].Fields), "carried on in ChatGPT") {
		t.Fatalf("in Activity: %v", log)
	}
	postForm(t, h, "/activity/"+log[0].ID+"/undo", nil)
	if notes, _ := a.Store.List("note", store.ListOptions{}); len(notes) != 1 || strings.Contains(notes[0].Fields["body"].(string), "14th") {
		t.Errorf("Undo puts the words back as they were: %+v", notes)
	}
	// Another conversation by the same title is a note of its own.
	bringChats(t, h, false, "Sell the sofa")
	if notes, _ := a.Store.List("note", store.ListOptions{}); len(notes) != 2 {
		t.Errorf("a new conversation by the same title is a second note: %d", len(notes))
	}
}
