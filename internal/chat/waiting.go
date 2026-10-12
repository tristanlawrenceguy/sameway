package chat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What waits for the person is gathered here, once, for whoever shows it:
// Today puts each kind in its own place, the morning brief counts them and
// an agent reads the same list. Each reader used to find its own, so Today,
// the brief and an agent could each say something else. Each kind keeps
// its own words and its own answers; what is shared is which things wait,
// and so how many.

// The kinds of thing that wait, in the order they are listed.
const (
	WaitQuestion   = "question"   // the assistant asks before it acts (propose.go)
	WaitSort       = "sort"       // an email or a shared message not yet dealt with
	WaitReply      = "reply"      // an email that asked the person something (records/turn.go)
	WaitSuggestion = "suggestion" // a record an action suggests (suggest_records.go)
	WaitTag        = "tag"        // a tag an action gave, to keep or change (classify.go)
)

// ToSort is the tag on what came in and is not yet dealt with.
const ToSort = "to sort"

// tagsShown is how many tags to check wait at once, newest first: a full
// inbox is a few questions at a time, not a wall of them.
const tagsShown = 20

// Waiting is one thing waiting for the person.
type Waiting struct {
	Kind   string        `json:"kind"`
	Record *store.Record `json:"-"` // what waits: the question, the email, the tag given
	// About is the record it is about, as type/id: the email a suggestion
	// or a tag came from.
	About   string  `json:"about,omitempty"`
	Words   string  `json:"words"`
	Link    string  `json:"link"`
	Presses []Press `json:"presses,omitempty"`
}

// Press is one answer, as the page gives it: a form posted to Post, with
// the id when it takes one, and the one more field Takes names, if any.
type Press struct {
	Label string `json:"label"`
	Post  string `json:"post"`
	ID    string `json:"id,omitempty"`
	Takes string `json:"takes,omitempty"`
}

// Waiting is everything waiting for the person this service speaks for.
// The assistant's questions are the owner's alone, as in the chat.
func (s *Service) Waiting() []Waiting {
	var out []Waiting
	if s.IsOwner() {
		for _, p := range s.Proposals() {
			out = append(out, Waiting{Kind: WaitQuestion, Record: p, Words: fmt.Sprint(p.Fields["summary"]), Link: "/t/" + records.ProposalType + "/" + p.ID, Presses: QuestionPresses(p)})
		}
	}
	alone := s.AloneTags()
	for _, r := range s.ToSort() {
		if AloneOn(r, alone) == "" {
			out = append(out, Waiting{Kind: WaitSort, Record: r, Words: s.name(r), Link: "/t/" + r.Type + "/" + r.ID,
				Presses: []Press{{Label: "Make it a task", Post: "/mail/task", ID: r.ID}, {Label: "Done with it", Post: "/mail/sorted", ID: r.ID}}})
		}
	}
	out = append(out, s.yourTurn()...)
	for _, p := range s.Suggestions() {
		from := fmt.Sprint(p.Fields["from"])
		out = append(out, Waiting{Kind: WaitSuggestion, Record: p, About: from, Words: SuggestionWords(p), Link: "/t/" + from,
			Presses: []Press{{Label: "Make it", Post: "/suggested/yes", ID: p.ID}, {Label: "Change", Post: "/suggested/change", ID: p.ID}, {Label: "No", Post: "/suggested/no", ID: p.ID}}})
	}
	return append(out, s.tagsToCheck(alone)...)
}

// WaitingOf is the things of one kind in a list, in its order.
func WaitingOf(list []Waiting, kind string) []Waiting {
	var out []Waiting
	for _, w := range list {
		if w.Kind == kind {
			out = append(out, w)
		}
	}
	return out
}

// ToSort are the emails and shared notes not yet dealt with, oldest
// first, those that need nothing as well. An email is a record of its
// own; one from before was a note.
func (s *Service) ToSort() []*store.Record {
	var out []*store.Record
	for _, typ := range []string{records.EmailType, "note"} {
		if _, ok := s.Store.Types().Get(typ); !ok {
			continue
		}
		recs, _ := s.Store.List(typ, store.ListOptions{OrderBy: "created_at"})
		for _, r := range recs {
			if Tagged(r, ToSort) && (typ == records.EmailType || Tagged(r, "email") || Tagged(r, "shared")) {
				out = append(out, r)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// Tagged is whether a record has a tag, as it was given.
func Tagged(r *store.Record, tag string) bool {
	tags, _ := r.Fields["tags"].([]any)
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// yourTurn are the emails whose latest asked the person something,
// oldest first; each is taken off with a press when its turn is theirs
// to change.
func (s *Service) yourTurn() []Waiting {
	t, ok := s.Store.Types().Get(records.EmailType)
	if !ok {
		return nil
	}
	emails, _ := s.Store.List(records.EmailType, store.ListOptions{OrderBy: "created_at"})
	f, _ := t.Field("turn")
	var out []Waiting
	for _, e := range emails {
		if e.Fields["turn"] != "yours" {
			continue
		}
		w := Waiting{Kind: WaitReply, Record: e, Words: s.name(e), Link: "/t/" + records.EmailType + "/" + e.ID}
		if f != nil && !f.ReadOnly {
			w.Presses = []Press{{Label: "No reply needed", Post: "/mail/answered", ID: e.ID}}
		}
		out = append(out, w)
	}
	return out
}

// tagsToCheck are the tags an action suggested, newest first, a few at a
// time; a tag given only when no other fits is not asked about, and one
// on a record since gone is not either.
func (s *Service) tagsToCheck(alone map[string]bool) []Waiting {
	if _, ok := s.Store.Types().Get(ClassificationType); !ok {
		return nil
	}
	recs, _ := s.Store.List(ClassificationType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 100})
	// A tag is changed to another of the person's tags, so Change is
	// there when there is more than one to choose from.
	other := 0
	for _, d := range s.tagDefs(nil) {
		if !d.alone {
			other++
		}
	}
	var out []Waiting
	for _, c := range recs {
		tag, about := fmt.Sprint(c.Fields["tag"]), fmt.Sprint(c.Fields["record"])
		if c.Fields["state"] != "suggested" || alone[strings.ToLower(tag)] || len(out) == tagsShown {
			continue
		}
		typ, id, _ := strings.Cut(about, "/")
		rec, err := s.Store.Get(typ, id)
		if err != nil {
			continue
		}
		w := Waiting{Kind: WaitTag, Record: c, About: about, Words: tag + " on " + s.name(rec), Link: "/t/" + about,
			Presses: []Press{{Label: "Keep", Post: "/tags/keep", ID: c.ID}, {Label: "Take it off", Post: "/tags/off", ID: c.ID}}}
		if other > 1 {
			w.Presses = append(w.Presses, Press{Label: "Change", Post: "/tags/change", ID: c.ID, Takes: "to"})
		}
		out = append(out, w)
	}
	return out
}

// AloneTags are the tags given only when no other fits, by name in lower
// case.
func (s *Service) AloneTags() map[string]bool {
	out := map[string]bool{}
	for _, d := range s.tagDefs(nil) {
		if d.alone {
			out[strings.ToLower(d.name)] = true
		}
	}
	return out
}

// AloneOn is the tag a record has that is given only when no other fits,
// or "".
func AloneOn(rec *store.Record, alone map[string]bool) string {
	for _, t := range records.StringList(rec.Fields["tags"]) {
		if alone[strings.ToLower(t)] {
			return t
		}
	}
	return ""
}

// name is what a record is called, whatever its type.
func (s *Service) name(rec *store.Record) string {
	t, ok := s.Store.Types().Get(rec.Type)
	if !ok {
		return rec.ID
	}
	return records.Name(s.Store, t, rec)
}

// SuggestionWords is what a suggestion asks, and why.
func SuggestionWords(p *store.Record) string {
	summary, _ := p.Fields["summary"].(string)
	why, _ := p.Fields["detail"].(string)
	if why = strings.TrimSpace(strings.TrimSuffix(why, ".")); why != "" {
		summary += " " + strings.ToUpper(why[:1]) + why[1:] + "."
	}
	return summary
}

// QuestionPresses are a question's answers, wherever it is asked: yes and
// no, each saying what it does, and another way first when there is one
// (hiding instead of deleting), as the one that keeps everything.
func QuestionPresses(p *store.Record) []Press {
	at := "/proposal/" + p.ID
	yes, no := "Yes, do it", "No, leave it"
	if l := acceptLabel(p.Fields["action"]); l != "" {
		yes = l
	}
	// A question Sameway put itself says what each answer does in its own
	// words.
	if l, _ := p.Fields["yes"].(string); l != "" {
		yes = l
	}
	if l, _ := p.Fields["no"].(string); l != "" {
		no = l
	}
	var out []Press
	if l, _ := p.Fields["instead"].(string); l != "" {
		out = append(out, Press{Label: l, Post: at + "/instead"})
	}
	return append(out, Press{Label: yes, Post: at + "/accept"}, Press{Label: no, Post: at + "/dismiss"})
}

// acceptLabel is what yes does to the canvas or a command, in words.
func acceptLabel(action any) string {
	a, _ := action.(map[string]any)
	switch tool, _ := a["tool"].(string); tool {
	case "add_component":
		return "Yes, add it"
	case "remove_component":
		return "Yes, remove it"
	case "update_component":
		return "Yes, change it"
	case "accept_action":
		return "Yes, run it"
	}
	return ""
}
