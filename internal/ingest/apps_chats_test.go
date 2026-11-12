package ingest

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Conversations exported from ChatGPT and Claude come in as notes, one a
// conversation, by its title, with what was said in turn and the day it
// began, tagged with where they came from: ChatGPT's read along the branch
// it ended on, not one edited away; a chat with nothing said left out.
func TestChatsFromChatGPTAndClaudeComeInAsNotes(t *testing.T) {
	t.Parallel()
	gpt := `[{"title":"Trip to Porto","create_time":1760000000,"current_node":"c","mapping":{
		"root":{"parent":null,"message":null},
		"a":{"parent":"root","message":{"author":{"role":"user"},"content":{"parts":["Plan 3 days in Porto"]}}},
		"old":{"parent":"a","message":{"author":{"role":"assistant"},"content":{"parts":["An answer edited away"]}}},
		"b":{"parent":"a","message":{"author":{"role":"assistant"},"content":{"parts":["Day 1: the river."]}}},
		"c":{"parent":"b","message":{"author":{"role":"user"},"content":{"parts":["Book the train for Friday"]}}}}},
		{"title":"Empty","create_time":1760000000,"current_node":"r","mapping":{"r":{"parent":null,"message":null}}}]`
	for _, name := range []string{"conversations.json", "chatgpt-export.zip"} {
		data := []byte(gpt)
		if strings.HasSuffix(name, ".zip") {
			data = zipOf(map[string]string{"conversations.json": gpt, "chat.html": "<html></html>"})
		}
		b, err := Recognise(name, data)
		if err != nil || b.From != "ChatGPT" || b.Count("note") != 1 {
			t.Fatalf("%s: %v %+v", name, err, b)
		}
		note := b.Kinds["note"].Rows[0]
		body := note["body"]
		if note["title"] != "Trip to Porto" || note["tags"] != "chatgpt" || !strings.HasPrefix(body, "Began ") ||
			!strings.Contains(body, "**You:** Plan 3 days in Porto\n\n**ChatGPT:** Day 1: the river.\n\n**You:** Book the train for Friday") || strings.Contains(body, "edited away") {
			t.Errorf("%s: the conversation as it ended, in turn: %+v", name, note)
		}
	}

	claude := `[{"name":"Meals for the week","created_at":"2026-10-20T08:00:00Z","chat_messages":[
		{"sender":"assistant","text":"Monday: lentil soup.","created_at":"2026-10-20T08:00:05Z"},
		{"sender":"human","text":"Plan dinners, no fish","created_at":"2026-10-20T08:00:01Z"},
		{"sender":"human","text":"And this letter?","created_at":"2026-10-20T08:01:00Z","attachments":[{"file_name":"gp-letter.txt","file_size":60,"file_type":"txt","extracted_content":"Book a blood test before 1 December."}],"files":[]}]}]`
	b, err := Recognise("data.zip", zipOf(map[string]string{"conversations.json": claude, "users.json": "[]"}))
	if err != nil || b.From != "Claude" || b.Count("note") != 1 {
		t.Fatalf("Claude: %v %+v", err, b)
	}
	if body := b.Kinds["note"].Rows[0]["body"]; !strings.Contains(body, "**You:** Plan dinners, no fish\n\n**Claude:** Monday: lentil soup.") || !strings.Contains(body, "Tuesday 20 October 2026") ||
		!strings.Contains(body, "**You:** And this letter?\n\n*Attached: gp-letter.txt*\n\nBook a blood test before 1 December.") {
		t.Errorf("Claude: in the order said, with its day and what was attached: %s", body)
	}

	if _, err := Recognise("conversations.json", []byte(`[{"title":"x","current_node":"r","mapping":{"r":{"parent":null,"message":null}}}]`)); err == nil || !strings.Contains(err.Error(), "no conversations with anything said") {
		t.Errorf("an export with nothing said says so: %v", err)
	}
}

// A very long conversation keeps its start and its end, where what was
// decided is, and stays whole words in any script.
func TestALongConversationKeepsItsEnd(t *testing.T) {
	t.Parallel()
	gpt := `[{"title":"Long","create_time":1760000000,"current_node":"b","mapping":{
		"r":{"parent":null,"message":null},
		"a":{"parent":"r","message":{"author":{"role":"user"},"content":{"parts":["` + strings.Repeat("Καλημέρα. ", 30000) + `"]}}},
		"b":{"parent":"a","message":{"author":{"role":"assistant"},"content":{"parts":["Book the doctor for Tuesday."]}}}}}]`
	b, err := Recognise("conversations.json", []byte(gpt))
	if err != nil || b.Count("note") != 1 {
		t.Fatalf("%v %+v", err, b)
	}
	body := b.Kinds["note"].Rows[0]["body"]
	if !utf8.ValidString(body) || !strings.HasSuffix(body, "Book the doctor for Tuesday.") || !strings.Contains(body, "the middle of this conversation is in the export") || len(body) > maxChatBody+200 {
		t.Errorf("start and end, whole letters: %d bytes, ends %q", len(body), body[len(body)-60:])
	}
}

// A long ChatGPT history comes split, as conversations-000.json and on,
// and is read whole; a voice chat is read as ChatGPT wrote it down.
func TestASplitChatGPTExportAndAVoiceChatComeIn(t *testing.T) {
	t.Parallel()
	conv := func(title, part string) string {
		return `[{"title":"` + title + `","create_time":1760000000,"current_node":"a","mapping":{
			"r":{"parent":null,"message":null},
			"a":{"parent":"r","message":{"author":{"role":"user"},"content":{"content_type":"multimodal_text","parts":[` + part + `]}}}}}]`
	}
	voice := `{"content_type":"audio_transcription","text":"Ring the dentist on Monday","direction":"in"},{"content_type":"audio_asset_pointer","asset_pointer":"sediment://file_0"}`
	b, err := Recognise("export.zip", zipOf(map[string]string{
		"conversations-000.json": conv("Porto", `"Book the train"`),
		"conversations-001.json": conv("Dentist", voice),
		"export_manifest.json":   `{"export_files":[]}`,
	}))
	if err != nil || b.From != "ChatGPT" || b.Count("note") != 2 {
		t.Fatalf("both files of a split export: %v %+v", err, b)
	}
	rows := b.Kinds["note"].Rows
	if body := rows[0]["body"] + rows[1]["body"]; !strings.Contains(body, "**You:** Ring the dentist on Monday") {
		t.Errorf("what was said aloud: %s", body)
	}
}

// What Claude made beside its answer (an artifact) and what ChatGPT wrote
// on a canvas are part of what they said: each comes after its message,
// under its title, as that message left it.
func TestWhatWasMadeInAChatComesUnderItsTitle(t *testing.T) {
	t.Parallel()
	claude := `[{"name":"Meals","created_at":"2026-10-20T08:00:00Z","chat_messages":[
		{"sender":"human","text":"Plan dinners","created_at":"2026-10-20T08:00:01Z"},
		{"sender":"assistant","text":"Here is a list.","created_at":"2026-10-20T08:00:05Z","content":[{"type":"text","text":"Here is a list."},
			{"type":"tool_use","name":"artifacts","input":{"id":"shop","type":"text/markdown","title":"Shopping list","command":"create","content":"- lentils\n- carrots"}}]},
		{"sender":"assistant","text":"Changed.","created_at":"2026-10-20T08:01:00Z","content":[
			{"type":"tool_use","name":"artifacts","input":{"id":"shop","command":"update","old_str":"carrots","new_str":"oat milk"}}]}]}]`
	b, err := Recognise("conversations.json", []byte(claude))
	if err != nil {
		t.Fatal(err)
	}
	if body := b.Kinds["note"].Rows[0]["body"]; !strings.Contains(body, "**Claude:** Here is a list.\n\n*Shopping list*\n\n- lentils\n- carrots\n\n") ||
		!strings.Contains(body, "**Claude:** Changed.\n\n*Shopping list*\n\n- lentils\n- oat milk") {
		t.Errorf("Claude's artifact, made and changed: %s", body)
	}

	gpt := `[{"title":"Letter","create_time":1760000000,"current_node":"d","mapping":{
		"r":{"parent":null,"message":null},
		"a":{"parent":"r","message":{"author":{"role":"user"},"content":{"content_type":"text","parts":["Draft a letter to the council"]}}},
		"b":{"parent":"a","message":{"author":{"role":"assistant"},"recipient":"canmore.create_textdoc","content":{"content_type":"code","text":"{\"name\":\"Council letter\",\"type\":\"document\",\"content\":\"Dear council, the bins.\"}"}}},
		"c":{"parent":"b","message":{"author":{"role":"user"},"content":{"content_type":"text","parts":["Say Tuesday"]}}},
		"d":{"parent":"c","message":{"author":{"role":"assistant"},"recipient":"canmore.update_textdoc","content":{"content_type":"code","text":"{\"updates\":[{\"pattern\":\"the bins\",\"multiple\":false,\"replacement\":\"the bins on Tuesday\"}]}"}}}}}]`
	if b, err = Recognise("conversations.json", []byte(gpt)); err != nil {
		t.Fatal(err)
	}
	if body := b.Kinds["note"].Rows[0]["body"]; !strings.Contains(body, "**ChatGPT:** *Council letter*\n\nDear council, the bins.\n\n**You:** Say Tuesday\n\n**ChatGPT:** *Council letter*\n\nDear council, the bins on Tuesday.") {
		t.Errorf("ChatGPT's canvas, made and changed: %s", body)
	}
}
