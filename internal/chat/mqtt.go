package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An mqtt action publishes a payload to a topic on the broker workspace.yaml
// names, which is how a button reaches a light, a plug or a hub without a
// program ever running. The broker is the person's, set by them, so the
// action needs no acceptance; every press is logged like any other run.

// mqttAction publishes what the action says.
func (s *Service) mqttAction(_ context.Context, rec *store.Record, title string) toolResult {
	topic, _ := rec.Fields["topic"].(string)
	payload, _ := rec.Fields["payload"].(string)
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fail("action %s has no topic to publish to", title)
	}
	if s.Publish == nil {
		return fail("action %s: no MQTT broker is configured; set mqtt.broker in workspace.yaml and start sameway again", title)
	}
	if err := s.Publish(topic, payload); err != nil {
		return fail("action %s: %v", title, err)
	}
	return toolResult{
		text:   fmt.Sprintf("%s published %q to %s", title, payload, topic),
		change: &Change{Action: "ran", Component: ActionType, ID: rec.ID, Detail: title + " (" + topic + ")", Href: "/t/" + ActionType + "/" + rec.ID},
	}
}
