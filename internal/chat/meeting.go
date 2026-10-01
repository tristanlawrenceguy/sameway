package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A meeting is an event, and its recording is a file with a transcript.
// Writing one up is a summary, what was decided, and the tasks that came
// up, each decision and task linked to the line of the transcript where
// it was said, so anyone can go back and hear it. It is written in one
// go and taken back in one go: one entry in the log, one Undo.

// EventType is the content type of a meeting.
const EventType = "event"

type meetingItem struct {
	Text  string `json:"text"`
	Title string `json:"title"`
	At    string `json:"at"`
	Due   string `json:"due"`
	For   string `json:"for"`
}

func (s *Service) meetingTools() []llm.Tool {
	if _, ok := s.Store.Types().Get(EventType); !ok {
		return nil
	}
	at := map[string]any{"type": "string", "description": "Where in the recording it was said, as the transcript's [m:ss] or [h:mm:ss] gives it, such as 12:03. Leave out when it was not."}
	return []llm.Tool{{Name: "write_up_meeting",
		Description: "Write up a meeting from its recording's transcript: a short summary, what was decided, and the tasks that came up, each with where in the recording it was said. Read the recording with get_record on file first. Give the event when the meeting is one already, the recording when it is not and a meeting is made for it, or both to join them. It is written in one go and undone in one go; the decisions and tasks link to the line they came from.",
		Schema: obj(map[string]any{
			"event":     map[string]any{"type": "string", "description": "The event id of the meeting, when there is one."},
			"recording": map[string]any{"type": "string", "description": "The file id of its recording."},
			"summary":   map[string]any{"type": "string", "description": "What was said, in short: a few sentences or a short list, in Markdown."},
			"decisions": map[string]any{"type": "array", "description": "What was decided, one each.", "items": obj(map[string]any{
				"text": map[string]any{"type": "string", "description": "The decision, in a sentence."}, "at": at}, "text")},
			"tasks": map[string]any{"type": "array", "description": "What someone is to do, one each; each becomes a task.", "items": obj(map[string]any{
				"title": map[string]any{"type": "string", "description": "The task, as a short thing to do."},
				"at":    at,
				"due":   map[string]any{"type": "string", "description": "When it is due, when one was said, as a date."},
				"for":   map[string]any{"type": "string", "description": "The person id it is for, when it was said who; find_records on person."},
			}, "title")},
		}, "summary")}}
}

func (s *Service) writeUpMeeting(eventID, fileID, summary string, decisions, tasks []meetingItem) toolResult {
	if eventID == "" && fileID == "" {
		return fail("give the event of the meeting, or the recording to make one for; find_records on event or on file")
	}
	var batch []BatchItem
	var event map[string]any
	if eventID != "" {
		rec, err := s.Store.Get(EventType, eventID)
		if err != nil {
			return fail("no event %s; find_records on event, or give only the recording to make one", eventID)
		}
		event = rec.Fields
		if fileID == "" {
			fileID, _ = event["recording"].(string)
		}
	}
	var heard []convert.Cue
	var recordingTitle string
	if fileID != "" {
		file, err := s.Store.Get(FileType, fileID)
		if err != nil {
			return fail("no file %s; find_records on file for the recording", fileID)
		}
		recordingTitle, _ = file.Fields["title"].(string)
		text, _ := file.Fields["text"].(string)
		heard = convert.FromTranscript(text)
	}
	said := func(at string) string { return moment(fileID, heard, at) }

	var lines []string
	for _, d := range decisions {
		if t := strings.TrimSpace(d.Text); t != "" {
			lines = append(lines, "- "+t+said(d.At))
		}
	}
	fields := map[string]any{"summary": strings.TrimSpace(summary), "decisions": strings.Join(lines, "\n")}
	if fileID != "" && (event == nil || event["recording"] == "" || event["recording"] == nil) {
		fields["recording"] = fileID
	}
	if event == nil {
		fields["title"] = strings.TrimSpace("Meeting: " + recordingTitle)
		rec, _, err := Write(s.Store, "created", EventType, "", fields)
		if err != nil {
			return fail("the meeting could not be made: %v", err)
		}
		eventID, event = rec.ID, rec.Fields
		batch = append(batch, BatchItem{Type: EventType, ID: rec.ID})
	} else {
		if _, _, err := Write(s.Store, "updated", EventType, eventID, fields); err != nil {
			return fail("the meeting could not be written up: %v", err)
		}
		batch = append(batch, BatchItem{Type: EventType, ID: eventID, Before: event})
	}

	made := 0
	for _, t := range tasks {
		title := strings.TrimSpace(t.Title)
		if title == "" {
			continue
		}
		f := map[string]any{"title": title, "event": eventID}
		if note := said(t.At); note != "" {
			f["notes"] = "Came up" + note + "."
		}
		if t.Due != "" {
			f["due"] = t.Due
		}
		if t.For != "" {
			f["for"] = t.For
		}
		rec, _, err := Write(s.Store, "created", "task", "", f)
		if err != nil {
			// What was written stays one batch, so Undo still takes it all.
			s.recordBatch(eventID, event, made, len(lines), batch)
			return fail("the meeting is written up, but the task %q could not be made: %v; make it with create_record", title, err)
		}
		batch = append(batch, BatchItem{Type: "task", ID: rec.ID})
		made++
	}
	c := s.recordBatch(eventID, event, made, len(lines), batch)
	title, _ := event["title"].(string)
	return toolResult{text: fmt.Sprintf("wrote up %s, with its tasks shown, at %s: the summary, %s and %s, each linked to where it was said.",
		title, c.Href, count(len(lines), "decision"), count(made, "task")), change: &c}
}

// recordBatch is the write-up as one change, for the log and its Undo.
func (s *Service) recordBatch(eventID string, event map[string]any, tasks, decisions int, batch []BatchItem) Change {
	title, _ := event["title"].(string)
	// Its page shows the tasks that came up at it: a write-up asks for them.
	return Change{Action: "wrote up", Component: EventType, ID: eventID, Href: "/t/" + EventType + "/" + eventID + "?show=points-here:task.event",
		Detail: fmt.Sprintf("%s, %s and %s", title, count(decisions, "decision"), count(tasks, "task")), Before: Batch(batch)}
}

// moment is " (at 12:03)" linked to that line of the recording's
// transcript, or "" when nothing says where. The line is the one being
// said at that time; with no transcript to find it in, the time itself.
func moment(fileID string, heard []convert.Cue, at string) string {
	at = strings.Trim(strings.TrimSpace(at), "[]")
	if fileID == "" || at == "" {
		return ""
	}
	secs, ok := clockSeconds(at)
	if !ok {
		return ""
	}
	start := secs
	for _, c := range heard {
		if int(c.Start) <= secs {
			start = int(c.Start)
		}
	}
	return fmt.Sprintf(" ([at %s](/t/%s/%s#media-%s-at-%d))", convert.Clock(float64(start)), FileType, fileID, fileID, start)
}

// clockSeconds reads 12:03 or 1:02:03 as seconds.
func clockSeconds(at string) (int, bool) {
	total := 0
	parts := strings.Split(at, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	for _, p := range parts {
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err != nil || n < 0 {
			return 0, false
		}
		total = total*60 + n
	}
	return total, true
}

func count(n int, one string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %ss", n, one)
}
