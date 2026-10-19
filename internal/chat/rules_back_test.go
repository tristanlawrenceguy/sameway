package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// Back after time away, the assistant is told to catch the person up from
// what waits and what is late, and to offer Today's keeping the long late
// with no date; asked to track tablets, to set when each dose reminds.
func TestTheRulesForComingBackAndMedicines(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ said, want string }{
		{"I'm back now, need to catch up on everything", "keep them as tasks with no date"},
		{"Mum has morning and evening pills", "with remind set to each time"},
	} {
		svc, m := withModel(t, &llm.Response{Text: "ok"})
		svc.Send(context.Background(), c.said)
		if !strings.Contains(m.seen[0].System, c.want) {
			t.Errorf("%q brings %q", c.said, c.want)
		}
	}
}

// A pile handed over by someone overwhelmed is given days from today; a
// carer back is told of the habits kept; an appointment only in a note is
// offered its day.
func TestTheRulesForAPileACarerAndAnAppointment(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ said, want string }{
		{"everything is piling up, I want to stop worrying about it", "never a day already past"},
		{"I'm back after two weeks, how is Mum?", "were kept while they were away"},
		{"Mum's dentist appointment moved to Friday", "offer once to put it on its day"},
	} {
		svc, m := withModel(t, &llm.Response{Text: "ok"})
		svc.Send(context.Background(), c.said)
		if !strings.Contains(m.seen[0].System, c.want) {
			t.Errorf("%q brings %q", c.said, c.want)
		}
	}
}

// Got back from somewhere is coming back too; listing the week looks in
// notes for appointments; correcting a day names what still gives the old.
func TestTheRulesForGotBackTheWeekAndCorrections(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ said, want string }{
		{"I just got back from moving house", "When the person is back after time away"},
		{"What reminders do I have this week?", "look in notes too for an appointment"},
		{"the GP visit is Friday, not Thursday, please fix the reminder", "still give the old one and name them"},
	} {
		svc, m := withModel(t, &llm.Response{Text: "ok"})
		svc.Send(context.Background(), c.said)
		if !strings.Contains(m.seen[0].System, c.want) {
			t.Errorf("%q brings %q", c.said, c.want)
		}
	}
}
