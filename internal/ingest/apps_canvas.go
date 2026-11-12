package ingest

import (
	"encoding/json"
	"regexp"
	"strings"
)

// What Claude or ChatGPT made beside its answer (a meal plan, a letter, a
// packing list) is an artifact in Claude and a canvas in ChatGPT. It is
// part of what was said, so it comes into the note after the message that
// made it, under its title, as that message left it. A change is to one
// made before, so each is kept whole here, by its id.

// canvas is what has been made so far in one conversation: by id, its
// title and its words.
type canvas map[string][2]string

// made is a Claude artifact as its message left it: made, written again,
// or with some words changed.
func (cv canvas) made(id, command, title, words, old, replaced string) string {
	was := cv[id]
	switch command {
	case "create", "rewrite":
		was[1] = words
	case "update":
		was[1] = strings.Replace(was[1], old, replaced, 1)
	default:
		return ""
	}
	if title != "" {
		was[0] = title
	}
	cv[id] = was
	return shown(was)
}

// gptMessage is one message in ChatGPT's export: its parts, or, sent to
// the canvas (canmore), what it wrote there as JSON.
type gptMessage struct {
	Author struct {
		Role string `json:"role"`
	} `json:"author"`
	Recipient string `json:"recipient"`
	Content   struct {
		Parts []json.RawMessage `json:"parts"`
		Text  string            `json:"text"`
	} `json:"content"`
}

// written is a ChatGPT canvas as its message left it: made with a name,
// or changed by patterns over the one made last (the export does not say
// which one an update is to).
func (cv canvas) written(recipient, code string) string {
	var doc struct {
		Name    string `json:"name"`
		Content string `json:"content"`
		Updates []struct {
			Pattern     string `json:"pattern"`
			Multiple    bool   `json:"multiple"`
			Replacement string `json:"replacement"`
		} `json:"updates"`
	}
	if !strings.HasPrefix(recipient, "canmore.") || json.Unmarshal([]byte(code), &doc) != nil {
		return ""
	}
	was := cv[""]
	switch recipient {
	case "canmore.create_textdoc":
		was = [2]string{doc.Name, doc.Content}
	case "canmore.update_textdoc":
		for _, u := range doc.Updates {
			re, err := regexp.Compile("(?s)" + u.Pattern) // ".*" is the whole canvas
			if err != nil {
				continue
			}
			if u.Multiple {
				was[1] = re.ReplaceAllLiteralString(was[1], u.Replacement)
			} else if at := re.FindStringIndex(was[1]); at != nil {
				was[1] = was[1][:at[0]] + u.Replacement + was[1][at[1]:]
			}
		}
	default:
		return ""
	}
	cv[""] = was
	return shown(was)
}

// shown is a thing made, under its title, after the words of its message.
func shown(made [2]string) string {
	title, words := strings.TrimSpace(made[0]), strings.TrimSpace(made[1])
	if words == "" {
		return ""
	}
	if title == "" {
		title = "What was made"
	}
	return "\n\n*" + title + "*\n\n" + words
}
