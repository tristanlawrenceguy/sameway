package ingest

import (
	"strings"
	"testing"
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
