package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// "I get worse at curating the lists the more overwhelmed I am": email and
// shared messages waited on Today for the person to turn into tasks, the
// curating they cannot do on a bad day. So each is put to the model as it
// comes, one narrow question with no tools: is there something to do, by
// when, does it matter, for whom. What comes back is a suggestion the
// person keeps with a press; nothing is made without them. Urgency is
// not asked: it is the day, which Sameway counts.

// Suggestion is what a message asks of the person, as the model read it.
type Suggestion struct {
	Task      bool   `json:"task"`
	Title     string `json:"title,omitempty"`
	Due       string `json:"due,omitempty"` // stored form: a day, or a moment
	Important bool   `json:"important,omitempty"`
	For       string `json:"for,omitempty"` // a person's name, as written
	Why       string `json:"why,omitempty"` // in a few words, shown beside it
}

const triageSystem = `You read one message the person forwarded or shared to their own workspace and say whether it asks them to do something.
Answer with JSON only, no other words:
{"task": true or false, "title": "...", "when": "...", "important": true or false, "for": "...", "why": "..."}
- task: true when the person has to do something: pay, reply, book, bring, send, sign, renew, attend, buy, call. False for newsletters, adverts, receipts already paid, confirmations needing nothing, things to read some day.
- title: the thing to do, as a short task starting with a verb, in the message's language: "Pay the water bill", "Bring the signed form". Not the message's subject.
- when: the words of the message that say the day it must be done by or happens on, copied exactly, as short as they can be: "Friday", "15 October at 14:30", "the 1st", "31 January 2027". Do not work out a date; Sameway does. Leave it out when no day is said.
- important: true for money owed, health, legal or official deadlines (tax, passport, visa, court, insurance), school, work deadlines, or a person waiting on an answer. False otherwise.
- for: the name of someone else the task is for, only when one of these people is named as doing it: %s. Leave it out otherwise.
- why: four to eight words saying why, for the person: "Bill due, late fee after". `

var triageJSON = regexp.MustCompile(`(?s)\{.*\}`)

// Triage asks the model what a message wants of the person. sent is when
// the message was sent, which a day in it is counted from.
func (s *Service) Triage(ctx context.Context, message string, sent time.Time, people []string) (Suggestion, error) {
	if s.Provider == nil {
		return Suggestion{}, errors.New("no model is connected")
	}
	if sent.IsZero() {
		sent = s.Now()
	}
	names := "none"
	if len(people) > 0 {
		names = strings.Join(people, ", ")
	}
	ask := fmt.Sprintf("Sent %s.\n\n%s", sent.Format("Monday 2 January 2006, 15:04"), clipRunes(message, 6000))
	resp, err := s.Provider.Complete(ctx, llm.Request{System: fmt.Sprintf(triageSystem, names), Messages: []llm.Message{{Role: llm.RoleUser, Content: ask}}, MaxTokens: 2048, Effort: "none"})
	if err != nil {
		return Suggestion{}, err
	}
	raw := triageJSON.FindString(resp.Text)
	if raw == "" {
		return Suggestion{}, fmt.Errorf("the model answered without JSON: %.80s", resp.Text)
	}
	var out struct {
		Suggestion
		Due  any `json:"due"`
		When any `json:"when"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return Suggestion{}, fmt.Errorf("the model's answer could not be read: %v", err)
	}
	sug := out.Suggestion
	sug.Title = strings.TrimSpace(sug.Title)
	if !sug.Task || sug.Title == "" {
		return Suggestion{Task: false, Why: sug.Why}, nil
	}
	// The model says the day in the message's words and Sameway counts it
	// from when it was sent: asked for a date, a small model wrote the 18th
	// for "by Friday" on Monday the 12th, and 30 November for 31 January.
	for _, v := range []any{out.When, out.Due} {
		if d, ok := v.(string); ok && sug.Due == "" {
			if at, day, ok := when.Parse(dayWords(d), sent); ok && (at.Format("2006-01-02") != sent.Format("2006-01-02") || saysToday(d)) {
				sug.Due = when.Store(at, day) // a day it could not read is left out, not guessed
			}
		}
	}
	return sug, nil
}

// saysToday is whether words name the day they were written on: "now" and
// "when you can" are no day, and a small model gave them as one.
func saysToday(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "today") || strings.Contains(s, "tonight") || strings.Contains(s, "this morning") || strings.Contains(s, "this afternoon") || strings.Contains(s, "this evening")
}

// dayWords is a day as a message says it, without what says how it
// stands to the task: "by Friday", "end of day Thursday", "Saturday
// morning" are Friday, Thursday and Saturday.
func dayWords(s string) string {
	s = strings.ToLower(strings.Trim(strings.TrimSpace(s), ".,!"))
	for _, lead := range []string{"by ", "on ", "before ", "until ", "till ", "due ", "end of day ", "the end of ", "end of ", "from ", "no later than "} {
		s = strings.TrimPrefix(s, lead)
	}
	for _, tail := range []string{" morning", " afternoon", " evening", " night", " at the latest", " latest", " eod"} {
		s = strings.TrimSuffix(s, tail)
	}
	return strings.TrimSpace(s)
}

func clipRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
