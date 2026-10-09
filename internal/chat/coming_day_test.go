package chat_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// Asked on a Tuesday for a walk on Saturday, a small model wrote the
// Saturday after next. It is the weekday named, so the day is not wrong
// as such; the result says which Saturday is the coming one, and a day
// that is the coming one is only said back.
func TestADayAWeekOutSaysWhichIsTheComingOne(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Clock = func() time.Time { return time.Date(2026, 10, 6, 19, 0, 0, 0, time.UTC) } // a Tuesday
	m := &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Walk", "starts": "2026-10-17 09:00"}}),
		call("create_record", map[string]any{"type": "event", "fields": map[string]any{"title": "Walk again", "starts": "2026-10-10 09:00"}}),
	}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "Plan my weekend: a walk on Saturday morning"); err != nil {
		t.Fatal(err)
	}
	far := lastToolResult(m.seen[1]).Content
	if !strings.Contains(far, "Saturday 17 October is not the coming Saturday, which is 10 October") {
		t.Errorf("a day a week out says which is the coming one: %s", far)
	}
	near := lastToolResult(m.seen[2]).Content
	if strings.Contains(near, "not the coming") {
		t.Errorf("the coming one is only said back: %s", near)
	}
}
