package automate

import (
	"context"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/devices"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An mqtt action publishes a payload to a topic on the broker workspace.yaml
// names, which is how a button reaches a light, a plug or a hub without a
// program ever running. The broker is the person's, set by them, so the
// action needs no acceptance; every press is logged like any other run.

// mqttAction publishes what the action says.
func (s *Service) mqttAction(_ context.Context, rec *store.Record, title string) Result {
	topic, _ := rec.Fields["topic"].(string)
	payload, _ := rec.Fields["payload"].(string)
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fail("action %s has no topic to publish to", title)
	}
	if s.Publish == nil {
		return fail("action %s: your hub is not connected: Workspaces offers the one Sameway finds on your Wi-Fi, or set mqtt.broker", title)
	}
	if err := s.Publish(topic, payload); err != nil {
		return fail("action %s: %v", title, err)
	}
	return Result{
		Text:   fmt.Sprintf("%s published %q to %s", title, payload, topic),
		Change: &records.Change{Action: "ran", Component: records.ActionType, ID: rec.ID, Detail: title + " (" + topic + ")", Href: relate.Page(records.ActionType, rec.ID)},
	}
}

// SwitchDevice turns a device on or off by the way its topic is changed
// (devices/switch.go), and says what was asked. The device says its new
// state itself, as it does after any change, so nothing is written here.
func (s *Service) SwitchDevice(id string, on bool) (string, error) {
	rec, err := s.Store.Get(devices.DeviceType, id)
	if err != nil {
		return "", fmt.Errorf("there is no device %q", id)
	}
	name, _ := rec.Fields["name"].(string)
	topic, payload, ok := devices.Command(rec.Fields, on)
	if !ok {
		return "", fmt.Errorf("%s does not say it switches on and off in a way Sameway knows; an mqtt action can send it what its maker says", name)
	}
	if s.Publish == nil {
		return "", fmt.Errorf("your hub is not connected: Workspaces offers the one Sameway finds on your Wi-Fi")
	}
	if err := s.Publish(topic, payload); err != nil {
		return "", fmt.Errorf("%s was not reached: %v", name, err)
	}
	word := "off"
	if on {
		word = "on"
	}
	return fmt.Sprintf("Asked %s to turn %s", name, word), nil
}
