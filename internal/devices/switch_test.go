package devices_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/devices"
)

// A device that says it is on or off is turned the other way by the
// message its program expects; anything else gets no guess.
func TestADeviceIsSwitchedTheWayItsProgramExpects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		fields         map[string]any
		topic, payload string
	}{
		{"zigbee2mqtt", map[string]any{"topic": "zigbee2mqtt/kitchen_light", "state": "{\n  \"state\": \"ON\",\n  \"brightness\": 200\n}"}, "zigbee2mqtt/kitchen_light/set", `{"state":"OFF"}`},
		{"tasmota", map[string]any{"topic": "stat/desk_plug/POWER", "state": "ON"}, "cmnd/desk_plug/POWER", "OFF"},
		{"its own set topic", map[string]any{"topic": "home/lamp/state", "state": "ON", "set": "home/lamp/command"}, "home/lamp/command", "OFF"},
		{"plain on with no known way", map[string]any{"topic": "home/lamp/state", "state": "ON"}, "", ""},
		{"a sensor", map[string]any{"topic": "zigbee2mqtt/hall_sensor", "state": `{"temperature": 21}`}, "", ""},
	}
	for _, c := range cases {
		topic, payload, ok := devices.Command(c.fields, false)
		if topic != c.topic || payload != c.payload || ok != (c.topic != "") {
			t.Errorf("%s: %q %q %v, want %q %q", c.name, topic, payload, ok, c.topic, c.payload)
		}
	}
	if on, said := devices.OnOff(map[string]any{"state": `{"state":"OFF"}`}); on || !said {
		t.Errorf("off is read as off: %v %v", on, said)
	}
}
