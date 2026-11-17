package chat

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// How people reach the owner, in one answer: each way in and whether it
// is open (the server's, Service.Doors), the standing yes, and what came
// in by each lately: how many requests, how many were answered and how
// soon, and what waits. For "how do people reach me?" and "how quickly do
// I answer?", asked of the assistant, not a page of its own.

var waysInOp = Op{Title: "See how people reach you", Traits: Traits{ReadOnly: true, Idempotent: true},
	Access: ForOwner,
	Words:  []string{"reach me", "ways in", "front desk", "how quickly", "response time", "how fast", "channels"},
	Doing:  saying("Looking at how people reach you"),
	Run:    func(s *Service, _ toolArgs, _ llm.ToolCall) toolResult { return s.waysIn() },
	Tool: llm.Tool{Name: "ways_in",
		Description: "How people reach the person: each way in (email, published pages, booking link, Telegram, their website, other agents) and whether it is open, their standing yes, and the requests of the last 30 days by way in: how many, how many answered, how soon, and what waits.",
		Schema:      obj(map[string]any{})}}

func (s *Service) waysIn() toolResult {
	var lines []string
	if s.Doors != nil {
		lines = append(lines, s.Doors()...)
	}
	if rule := strings.TrimSpace(s.setting(AlwaysSetting)); rule != "" {
		lines = append(lines, "Standing yes: "+rule)
	} else {
		lines = append(lines, "Standing yes: none; every request waits for you")
	}
	since := s.Now().AddDate(0, 0, -30)
	type tally struct {
		asked, answered, waiting int
		took                     []time.Duration
	}
	by := map[string]*tally{}
	recs, _ := s.Store.List(records.ProposalType, store.ListOptions{})
	for _, p := range recs {
		if p.Fields["by_action"] != ByRequest || p.CreatedAt.Before(since) {
			continue
		}
		way := fmt.Sprint(p.Fields["detail"])
		if by[way] == nil {
			by[way] = &tally{}
		}
		t := by[way]
		t.asked++
		if RequestState(p) == "waiting" {
			t.waiting++
			continue
		}
		t.answered++
		t.took = append(t.took, p.UpdatedAt.Sub(p.CreatedAt))
	}
	if len(by) == 0 {
		lines = append(lines, "No requests in the last 30 days.")
	}
	var ways []string
	for w := range by {
		ways = append(ways, w)
	}
	sort.Strings(ways)
	for _, w := range ways {
		t := by[w]
		line := fmt.Sprintf("%s: %d requests in 30 days, %d answered", w, t.asked, t.answered)
		if len(t.took) > 0 {
			sort.Slice(t.took, func(i, j int) bool { return t.took[i] < t.took[j] })
			line += ", usually within " + roughly(t.took[len(t.took)/2])
		}
		if t.waiting > 0 {
			line += fmt.Sprintf(", %d waiting", t.waiting)
		}
		lines = append(lines, line)
	}
	return toolResult{text: strings.Join(lines, "\n")}
}

// roughly is a length of time as a person says it.
func roughly(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "a minute"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes", int(d.Minutes())+1)
	case d < 48*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()+0.5))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24+0.5))
}
