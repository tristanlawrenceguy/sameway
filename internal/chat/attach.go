package chat

import (
	"fmt"
	"strings"
)

// FileType is the content type a person's files become; see the server's
// upload. The chat only reads it.
const FileType = "file"

// attachmentChars caps how much of a file's text goes to the model with
// one message; the rest is on the file's page, which the model can name.
const attachmentChars = 8000

// attachment is what the model is told about a file that came with a
// message: what it is, where it is filed, and its text.
func (s *Service) attachment(id string) string {
	rec, err := s.Store.Get(FileType, id)
	if err != nil {
		return "\n\n[The attached file is no longer here.]"
	}
	title, _ := rec.Fields["title"].(string)
	kind, _ := rec.Fields["kind"].(string)
	text, _ := rec.Fields["text"].(string)
	status, _ := rec.Fields["status"].(string)
	description, _ := rec.Fields["description"].(string)
	var b strings.Builder
	fmt.Fprintf(&b, "\n\n[Attached: %s, a %s, filed at /t/%s/%s.", title, kind, FileType, rec.ID)
	switch {
	case status == "converting":
		b.WriteString(" Its text is still being read; say so, and that it will be on its page.]")
	case status == "failed":
		note, _ := rec.Fields["note"].(string)
		fmt.Fprintf(&b, " Its text could not be read: %s]", note)
	case kind == "image":
		if description == "" {
			b.WriteString(" It is a picture with no description yet; ask for one.]")
		} else {
			fmt.Fprintf(&b, " Described as: %s]", description)
		}
	case strings.TrimSpace(text) == "":
		b.WriteString(" It has no text.]")
	default:
		if len([]rune(text)) > attachmentChars {
			text = string([]rune(text)[:attachmentChars]) + "\n…(the rest is on its page)"
		}
		b.WriteString(" Its contents:]\n" + text)
	}
	return b.String()
}
