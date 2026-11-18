package chat

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The owner's yes, given ahead in their own words (requests.always:
// "always yes to bookings from clients in working hours"): a request to
// make something that it plainly covers is answered yes as it comes, as the owner would, and the
// owner is told. Whether it covers one is the model's judgement, made on
// what Sameway itself knows of the request, never on a word the asker
// wrote: the kind of change, its times and whether the owner is free
// then, and who asks as the owner's own records have them. So nothing an
// asker writes can talk the judgement round; unsure is no, and the
// request waits as any does.

// AlwaysSetting is where the owner's standing yes is kept.
const AlwaysSetting = "requests.always"

// answerAhead answers a request just made yes when the owner's standing
// yes plainly covers it, and says whether it did.
func (s *Service) answerAhead(p *store.Record) bool {
	rule := strings.TrimSpace(s.setting(AlwaysSetting))
	if rule == "" || s.Provider == nil {
		return false
	}
	facts := s.requestFacts(p)
	if facts == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	resp, err := s.Provider.Complete(ctx, llm.Request{
		System:    "You decide whether the owner's standing rule, in their own words, plainly covers a request, read as the owner meant it: a visit, a meeting or a booking is an event. Answer only yes or no. When it does not plainly cover it, or you are unsure, answer no.",
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "The owner's standing rule: " + rule + "\n\nThe request, as Sameway knows it:\n" + facts + "\n\nDoes the rule plainly cover it?"}},
		MaxTokens: 5, Effort: "none"}) // a model here that thinks first spent its five words thinking, and never said yes
	if err != nil || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(resp.Text)), "yes") {
		return false
	}
	owner := *s
	owner.who = records.Visitor{Access: records.Owner}
	if owner.Accept(p.ID) != nil {
		return false
	}
	summary, _ := p.Fields["summary"].(string)
	// Said as done, with the owner's own words and how to take them back:
	// the request's own line ends in a question it no longer asks.
	said := strings.TrimSuffix(summary, " Make it?") + " Made without asking you, by your standing yes: \"" + rule + "\". To change or stop it, tell the assistant."
	owner.Say(said)
	if s.Automate != nil && s.Automate.Tell != nil {
		s.Automate.Tell("Made by your standing yes", said, "/activity")
	}
	return true
}

// requestFacts is a request as Sameway itself knows it, without a word the
// asker wrote: the kind of thing to make, its times and whether the owner
// is free, who asks as the owner's records have them, and how it came.
// Only a request to make something new is judged: one to change what is
// already here always waits, as a real model said yes to a client moving
// the owner's own meeting under a rule for clients' visits.
func (s *Service) requestFacts(p *store.Record) string {
	action, _ := p.Fields["action"].(map[string]any)
	tool, _ := action["tool"].(string)
	typeName, _ := action["type"].(string)
	fields, _ := action["fields"].(map[string]any)
	t, ok := s.Store.Types().Get(typeName)
	if !ok || tool != "create_record" {
		return ""
	}
	lines := []string{fmt.Sprintf("- To make %s.", withA(t.Name))}
	if t.Description != "" {
		lines = append(lines, "- What "+withA(t.Name)+" is, as the workspace says: "+t.Description)
	}
	var names []string
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	loc := s.Now().Location()
	var from, to time.Time
	for _, name := range names {
		f, ok := t.Field(name)
		v, isText := fields[name].(string)
		if !ok || f.Type != "datetime" || !isText {
			continue
		}
		at, day, ok := when.Stored(v, s.Now())
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %s.", f.Display(), when.Text(when.Store(at, day), loc, false)))
		if !day {
			if from.IsZero() || at.Before(from) {
				from = at
			}
			if at.After(to) {
				to = at
			}
		}
	}
	if !from.IsZero() {
		if !to.After(from) {
			to = from.Add(time.Hour)
		}
		free := "free"
		if len(s.Busy(from, to)) > 0 {
			free = "busy"
		}
		lines = append(lines, "- The owner is "+free+" then.")
	}
	lines = append(lines, "- Asked by: "+s.askerFacts(p)+".", fmt.Sprintf("- How it came: %s.", p.Fields["detail"]))
	return strings.Join(lines, "\n")
}

// askerFacts is who asked as the owner's records have them: a person
// here, with the tags the owner gave them, or someone not in them.
func (s *Service) askerFacts(p *store.Record) string {
	v, ok := RequestAsker(p)
	if !ok || v.Person == "" {
		return "someone not in the owner's records"
	}
	person, err := s.Store.Get(records.PersonType, v.Person)
	if err != nil {
		return "someone not in the owner's records"
	}
	name, _ := person.Fields["name"].(string)
	out := name + ", a person in the owner's records"
	if tags := records.StringList(person.Fields["tags"]); len(tags) > 0 {
		out += ", tagged " + strings.Join(tags, ", ")
	}
	return out
}
