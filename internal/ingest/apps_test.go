package ingest

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

// Each export a person has is read as the app wrote it, into tasks and
// notes with their fields already named.
func TestTheExportsPeopleHaveAreRecognised(t *testing.T) {
	t.Parallel()
	todo := "TYPE,CONTENT,DESCRIPTION,PRIORITY,INDENT,AUTHOR,RESPONSIBLE,DATE,DATE_LANG,TIMEZONE\n" +
		"section,Errands,,,,,,,,\n" +
		"task,Call the bank,About the card,4,1,Me,,tomorrow,en,Europe/Berlin\n" +
		"task,Water plants,,1,1,Me,,every Monday,en,Europe/Berlin\n"
	b, err := Recognise("Inbox.csv", []byte(todo))
	if err != nil || b.From != "Todoist" || b.Count("task") != 2 || b.Kinds["task"].Rows[0]["due"] != "tomorrow" || b.Kinds["task"].Rows[0]["notes"] != "About the card" {
		t.Errorf("Todoist: %v %+v", err, b)
	}

	gt := `{"kind":"tasks#taskLists","items":[{"kind":"tasks#taskList","title":"Groceries","items":[{"title":"Milk","status":"completed","due":"2026-10-09T00:00:00.000Z"},{"title":"Bread","status":"needsAction"}]}]}`
	b, err = Recognise("Tasks.json", []byte(gt))
	if err != nil || b.From != "Google Tasks" || b.Count("task") != 2 || b.Kinds["task"].Rows[0]["done"] != "true" || b.Kinds["task"].Rows[0]["due"] != "2026-10-09" || b.Kinds["task"].Rows[0]["tags"] != "Groceries" {
		t.Errorf("Google Tasks: %v %+v", err, b)
	}

	enex := `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE en-export SYSTEM "http://xml.evernote.com/pub/evernote-export4.dtd"><en-export><note><title>Recipes</title><content><![CDATA[<en-note><div>Pancakes: <b>flour</b>, eggs</div></en-note>]]></content><tag>food</tag><tag>home</tag></note></en-export>`
	b, err = Recognise("My Notes.enex", []byte(enex))
	if err != nil || b.From != "Evernote" || b.Count("note") != 1 || !strings.Contains(b.Kinds["note"].Rows[0]["body"], "**flour**") || b.Kinds["note"].Rows[0]["tags"] != "food, home" {
		t.Errorf("Evernote: %v %+v", err, b)
	}

	b, err = Recognise("takeout.zip", zipOf(map[string]string{
		"Takeout/Keep/Shopping.json": `{"title":"","listContent":[{"text":"Eggs","isChecked":false},{"text":"Tea","isChecked":true}],"labels":[{"name":"home"}],"isTrashed":false}`,
		"Takeout/Keep/Old.json":      `{"title":"Gone","textContent":"x","isTrashed":true}`,
	}))
	if err != nil || b.From != "Google Keep" || b.Count("note") != 1 || !strings.Contains(b.Kinds["note"].Rows[0]["body"], "- [x] Tea") || b.Kinds["note"].Rows[0]["title"] != "Eggs" {
		t.Errorf("Google Keep, the bin left out: %v %+v", err, b)
	}

	b, err = Recognise("Export.zip", zipOf(map[string]string{
		"Garden 2f9c1a0b3d4e5f60718293a4b5c6d7e8.md": "# Garden plan\n\nTomatoes by the wall.",
		"Ideas.md": "Paint the shed.",
	}))
	if err != nil || b.Count("note") != 2 {
		t.Fatalf("Markdown notes: %v %+v", err, b)
	}
	titles := b.Kinds["note"].Rows[0]["title"] + "|" + b.Kinds["note"].Rows[1]["title"]
	if !strings.Contains(titles, "Garden plan") || !strings.Contains(titles, "Ideas") || strings.Contains(titles, "2f9c") {
		t.Errorf("titled by their heading or name, without Notion's id: %s", titles)
	}

	if _, err := Recognise("photo.png", []byte("x")); err == nil || !strings.Contains(err.Error(), "Todoist") {
		t.Errorf("something else says what is read: %v", err)
	}
}

func zipOf(files map[string]string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, _ := zw.Create(name)
		w.Write([]byte(body))
	}
	zw.Close()
	return buf.Bytes()
}
