package devices_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/devices"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A message on a topic becomes the device for that topic: made the first
// time with a name a person would use, and from then on its state and
// when it was seen. JSON is kept readable.
func TestAMessageBecomesADeviceAndKeepsItCurrent(t *testing.T) {
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	first := time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)
	if err := devices.Apply(st, "zigbee2mqtt/kitchen_light", `{"state":"ON","brightness":200}`, first); err != nil {
		t.Fatal(err)
	}
	if err := devices.Apply(st, "home/upstairs/temp", "21.5", first); err != nil {
		t.Fatal(err)
	}
	if err := devices.Apply(st, "zigbee2mqtt/kitchen_light", `{"state":"OFF"}`, first.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	recs, _ := st.List(devices.DeviceType, store.ListOptions{})
	if len(recs) != 2 {
		t.Fatalf("one device per topic, got %d", len(recs))
	}
	var light, temp map[string]any
	for _, r := range recs {
		switch r.Fields["topic"] {
		case "zigbee2mqtt/kitchen_light":
			light = r.Fields
		case "home/upstairs/temp":
			temp = r.Fields
		}
	}
	if light["name"] != "kitchen light" || !strings.Contains(light["state"].(string), `"state": "OFF"`) || light["seen"] != "2026-09-21T08:01:00Z" {
		t.Errorf("the light is named for its topic and holds the latest state, readable: %v", light)
	}
	if temp["name"] != "upstairs temp" || temp["state"] != "21.5" {
		t.Errorf("a plain payload is the state as it came: %v", temp)
	}
}

// Without a broker there is no bus, and publishing says what to set.
func TestNoBrokerIsNoBus(t *testing.T) {
	var b *devices.Bus
	if err := b.Publish("a/b", "on"); err == nil || !strings.Contains(err.Error(), "mqtt.broker") {
		t.Errorf("publishing with no broker says how to set one: %v", err)
	}
}
