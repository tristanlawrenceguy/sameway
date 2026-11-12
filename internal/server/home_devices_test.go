package server_test

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/devices"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The home's devices from the phone, with no broker in the test: a hub
// found on the Wi-Fi is offered on Workspaces in one press, saved with
// the common subscriptions and connected at once; a light that says it
// is on gets Turn off on its page, which sends Zigbee2MQTT's set message.
func TestAHubFoundIsOnePressAndALightOneMore(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var sent []string
	var connected devices.Config
	a, h := newAppWith(t, app.Options{Home: devices.Home{Connect: func(_ context.Context, cfg devices.Config, _ *store.Store, _ devices.Write) (func(string, string) error, error) {
		mu.Lock()
		connected = cfg
		mu.Unlock()
		return func(topic, payload string) error {
			mu.Lock()
			sent = append(sent, topic+" "+payload)
			mu.Unlock()
			return nil
		}, nil
	}}})
	srv := h.(*server.Server)
	if page := get(t, h, "/workspaces").Body.String(); strings.Contains(page, "Your home's devices") {
		t.Fatal("with no hub set or found, Workspaces says nothing of devices")
	}
	srv.LookForHubs(func(context.Context) ([]devices.Broker, error) {
		return []devices.Broker{{Name: "Home Assistant", Addr: "tcp://192.168.1.20:1883"}}, nil
	})
	for _, j := range srv.BackgroundJobs() {
		if j.Name == "A hub for the home's devices looked for" {
			if err := j.Run(context.Background(), time.Now()); err != nil {
				t.Fatal(err)
			}
		}
	}
	page := get(t, h, "/workspaces").Body.String()
	if !strings.Contains(page, "Home Assistant at 192.168.1.20:1883") || !strings.Contains(page, "Use this hub") {
		t.Fatalf("the hub found is offered: %.3000s", page)
	}

	// Not served yet: saved, and said to connect at the next start.
	wantStatus(t, postForm(t, h, "/devices/hub", url.Values{"broker": {"tcp://192.168.1.20:1883"}}), 303)
	if cfg := a.Workspace.Config.MQTT; cfg.Broker != "tcp://192.168.1.20:1883" || strings.Join(cfg.Subscribe, ",") != "zigbee2mqtt/+,stat/+/POWER" {
		t.Fatalf("the broker and the common subscriptions are saved: %+v", cfg)
	}
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, "connects the next time Sameway starts") {
		t.Errorf("not served, it says when it connects")
	}

	// Served: the next change connects at once.
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	if _, err := a.ConnectDevices(ctx); err != nil {
		t.Fatal(err)
	}
	wantStatus(t, postForm(t, h, "/devices/hub", url.Values{"broker": {"192.168.1.21"}}), 303)
	mu.Lock()
	if connected.Broker != "tcp://192.168.1.21:1883" {
		t.Errorf("a typed address is a broker with the usual port, connected at once: %q", connected.Broker)
	}
	mu.Unlock()

	light, err := a.Store.Create(devices.DeviceType, map[string]any{"name": "kitchen light", "topic": "zigbee2mqtt/kitchen_light", "state": "{\n  \"state\": \"ON\"\n}"})
	if err != nil {
		t.Fatal(err)
	}
	temp, _ := a.Store.Create(devices.DeviceType, map[string]any{"name": "upstairs temp", "topic": "home/upstairs/temp", "state": "21.5"})
	if page := get(t, h, "/t/device/"+temp.ID).Body.String(); strings.Contains(page, "Turn on") || strings.Contains(page, "Turn off") {
		t.Error("a sensor has no switch")
	}
	if page := get(t, h, "/t/device/"+light.ID).Body.String(); !strings.Contains(page, "It is on.") || !strings.Contains(page, "Turn off") {
		t.Fatalf("a light that is on offers Turn off: %.3000s", page)
	}
	wantStatus(t, postForm(t, h, "/devices/"+light.ID+"/switch", url.Values{"to": {"off"}}), 303)
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(sent, "|") != `zigbee2mqtt/kitchen_light/set {"state":"OFF"}` {
		t.Errorf("Zigbee2MQTT's set message is sent: %q", sent)
	}
}
