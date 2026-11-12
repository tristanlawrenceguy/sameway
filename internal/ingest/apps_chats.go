package ingest

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// People do their life admin with ChatGPT or Claude now: a trip planned in
// one chat, the meals for a week in another, what the doctor said in a
// third, each gone from view as the next chat starts. Both let a person
// export everything, as conversations.json (in a zip, or alone), and
// Sameway reads it: each conversation becomes a note, by its title, with
// what was said in turn and the day it began, tagged with where it came
// from, so it is found by search and the assistant can make tasks, dates
// and people of it. Chats with nothing said are left out.

// maxChatBody is the most of one conversation a note keeps: its start and
// its end, as what was decided comes last, and the middle said to be in the
// export.
const maxChatBody = 200_000

// chatFile is the name of the conversations in an export's zip: a long
// ChatGPT history comes split, as conversations-000.json and on.
var chatFile = regexp.MustCompile(`^conversations(-\d+)?\.json$`)

// joined is the conversations of a split export read as one.
func joined(a, b *Brought) *Brought {
	if a == nil {
		return b
	}
	if t := b.Kinds["note"]; t != nil {
		for _, row := range t.Rows {
			a.add("note", row)
		}
	}
	return a
}

// chatExport reads conversations.json from ChatGPT or Claude, or says it
// is neither.
func chatExport(data []byte, zone *time.Location) (*Brought, bool) {
	var raw []json.RawMessage
	if json.Unmarshal(data, &raw) != nil || len(raw) == 0 {
		return nil, false
	}
	var probe map[string]json.RawMessage
	if json.Unmarshal(raw[0], &probe) != nil {
		return nil, false
	}
	switch {
	case probe["mapping"] != nil:
		return chatGPT(raw, zone), true
	case probe["chat_messages"] != nil:
		return claudeChats(raw, zone), true
	}
	return nil, false
}

// turn is one thing said in a conversation.
type turn struct{ who, text string }

// chatNote adds one conversation as a note, when anything was said.
func chatNote(b *Brought, title string, began time.Time, turns []turn) {
	var body strings.Builder
	for _, t := range turns {
		if strings.TrimSpace(t.text) == "" {
			continue
		}
		fmt.Fprintf(&body, "**%s:** %s\n\n", t.who, strings.TrimSpace(t.text))
	}
	text := strings.TrimSpace(body.String())
	if text == "" {
		return
	}
	if len(text) > maxChatBody {
		// Cut by bytes, so a letter split in two (Greek, Chinese) is dropped.
		start, end := text[:maxChatBody/2], text[len(text)-maxChatBody/2:]
		text = strings.ToValidUTF8(start, "") + "\n\n… the middle of this conversation is in the export.\n\n" + strings.ToValidUTF8(end, "")
	}
	if strings.TrimSpace(title) == "" {
		title = "A conversation with " + b.From
	}
	if !began.IsZero() {
		text = "Began " + began.Format("Monday 2 January 2006") + ".\n\n" + text
	}
	b.add("note", map[string]string{"title": title, "body": text, "tags": strings.ToLower(b.From)})
}

// chatGPT reads ChatGPT's export: each conversation a tree of messages,
// read from where it ended back to where it began.
func chatGPT(raw []json.RawMessage, zone *time.Location) *Brought {
	b := newBrought("ChatGPT")
	for _, r := range raw {
		var c struct {
			Title       string  `json:"title"`
			CreateTime  float64 `json:"create_time"`
			CurrentNode string  `json:"current_node"`
			Mapping     map[string]struct {
				Parent  *string     `json:"parent"`
				Message *gptMessage `json:"message"`
			} `json:"mapping"`
		}
		if json.Unmarshal(r, &c) != nil {
			continue
		}
		var said []*gptMessage
		for id, seen := c.CurrentNode, 0; id != "" && seen < len(c.Mapping)+1; seen++ {
			node, ok := c.Mapping[id]
			if !ok {
				break
			}
			if m := node.Message; m != nil && (m.Author.Role == "user" || m.Author.Role == "assistant") {
				said = append(said, m)
			}
			id = ""
			if node.Parent != nil {
				id = *node.Parent
			}
		}
		// Read in the order said, so a canvas is changed after it is made.
		var turns []turn
		doc := canvas{}
		for i := len(said) - 1; i >= 0; i-- {
			m := said[i]
			if m.Author.Role == "user" {
				turns = append(turns, turn{"You", partsText(m.Content.Parts)})
			} else {
				turns = append(turns, turn{"ChatGPT", partsText(m.Content.Parts) + doc.written(m.Recipient, m.Content.Text)})
			}
		}
		chatNote(b, c.Title, unixTime(c.CreateTime).In(zone), turns)
	}
	return b
}

// partsText is a ChatGPT message's text: its parts that are words, and
// what was said aloud in a voice chat, as ChatGPT wrote it down.
func partsText(parts []json.RawMessage) string {
	var out []string
	for _, p := range parts {
		var s string
		var said struct {
			ContentType string `json:"content_type"`
			Text        string `json:"text"`
		}
		if json.Unmarshal(p, &s) == nil {
			out = append(out, s)
		} else if json.Unmarshal(p, &said) == nil && said.ContentType == "audio_transcription" {
			out = append(out, said.Text)
		}
	}
	return strings.Join(out, "\n")
}

func unixTime(sec float64) time.Time {
	if sec <= 0 {
		return time.Time{}
	}
	return time.Unix(int64(sec), 0).UTC()
}

// claudeChats reads Claude's export: each conversation its messages in
// order, from the person (human) and from Claude.
func claudeChats(raw []json.RawMessage, zone *time.Location) *Brought {
	b := newBrought("Claude")
	for _, r := range raw {
		var c struct {
			Name      string `json:"name"`
			CreatedAt string `json:"created_at"`
			Messages  []struct {
				Sender    string `json:"sender"`
				Text      string `json:"text"`
				CreatedAt string `json:"created_at"`
				// What was pasted or attached, as Claude read it: a long
				// paste is kept as an attachment, not in the text.
				Attachments []struct {
					Name  string `json:"file_name"`
					Words string `json:"extracted_content"`
				} `json:"attachments"`
				// What Claude made beside its answer, an artifact, is a
				// tool it used.
				Content []struct {
					Type  string `json:"type"`
					Name  string `json:"name"`
					Input struct {
						ID      string `json:"id"`
						Command string `json:"command"`
						Title   string `json:"title"`
						Content string `json:"content"`
						Old     string `json:"old_str"`
						New     string `json:"new_str"`
					} `json:"input"`
				} `json:"content"`
			} `json:"chat_messages"`
		}
		if json.Unmarshal(r, &c) != nil {
			continue
		}
		sort.SliceStable(c.Messages, func(i, j int) bool { return c.Messages[i].CreatedAt < c.Messages[j].CreatedAt })
		var turns []turn
		made := canvas{}
		for _, m := range c.Messages {
			who := "You"
			if m.Sender == "assistant" {
				who = "Claude"
			}
			text := m.Text
			for _, a := range m.Attachments {
				label := "*Attached: " + a.Name + "*"
				if a.Name == "" {
					label = "*Pasted:*"
				}
				if strings.TrimSpace(a.Words) != "" {
					text += "\n\n" + label + "\n\n" + strings.TrimSpace(a.Words)
				}
			}
			for _, u := range m.Content {
				if u.Type == "tool_use" && u.Name == "artifacts" {
					in := u.Input
					text += made.made(in.ID, in.Command, in.Title, in.Content, in.Old, in.New)
				}
			}
			turns = append(turns, turn{who, text})
		}
		began, _ := time.Parse(time.RFC3339, c.CreatedAt)
		chatNote(b, c.Name, began.In(zone), turns)
	}
	return b
}
