// Package devices is the way to the things in a home: an MQTT broker,
// which is what Zigbee2MQTT, Tasmota, ESPHome, Home Assistant and most
// hubs speak or bridge to. Topics the workspace subscribes to become
// device records, one per topic, with the latest payload as the state
// and when it was seen; an mqtt action publishes to a topic, which is how
// a button turns a light on. Nothing here runs a program.
package devices

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// DeviceType is the content type a subscribed topic lands in.
const DeviceType = "device"

// Config is the mqtt section of workspace.yaml.
type Config struct {
	// Broker is the address, such as tcp://192.168.1.10:1883 or
	// ssl://broker.example.com:8883. Empty means no broker.
	Broker   string `yaml:"broker"`
	ClientID string `yaml:"client_id"`
	// UsernameEnv and PasswordEnv name environment variables; the values
	// never go in the file.
	UsernameEnv string `yaml:"username_env"`
	PasswordEnv string `yaml:"password_env"`
	// Subscribe lists the topics whose messages become devices, with
	// MQTT wildcards: zigbee2mqtt/+ takes every Zigbee device.
	Subscribe []string `yaml:"subscribe"`
}

// Bus is a connection to the broker.
type Bus struct {
	client mqtt.Client
	store  *store.Store
}

// Start connects to the broker and subscribes, when the workspace names
// one; otherwise it returns nil and nothing else changes. The connection
// is kept up for as long as ctx lasts.
func Start(ctx context.Context, cfg Config, st *store.Store) (*Bus, error) {
	if strings.TrimSpace(cfg.Broker) == "" {
		return nil, nil
	}
	opts := mqtt.NewClientOptions().AddBroker(cfg.Broker).SetAutoReconnect(true).SetConnectRetry(true).SetConnectRetryInterval(10 * time.Second)
	id := cfg.ClientID
	if id == "" {
		id = "sameway"
	}
	opts.SetClientID(id)
	if cfg.UsernameEnv != "" {
		opts.SetUsername(os.Getenv(cfg.UsernameEnv))
	}
	if cfg.PasswordEnv != "" {
		opts.SetPassword(os.Getenv(cfg.PasswordEnv))
	}
	b := &Bus{store: st}
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		for _, topic := range cfg.Subscribe {
			if token := c.Subscribe(topic, 0, func(_ mqtt.Client, m mqtt.Message) {
				if err := Apply(st, m.Topic(), string(m.Payload()), time.Now()); err != nil {
					log.Printf("devices: %s: %v", m.Topic(), err)
				}
			}); token.Wait() && token.Error() != nil {
				log.Printf("devices: subscribe %s: %v", topic, token.Error())
			}
		}
	})
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) { log.Printf("devices: connection lost: %v", err) })
	b.client = mqtt.NewClient(opts)
	if token := b.client.Connect(); token.WaitTimeout(10*time.Second) && token.Error() != nil {
		return nil, fmt.Errorf("could not reach the broker at %s: %w", cfg.Broker, token.Error())
	}
	go func() {
		<-ctx.Done()
		b.client.Disconnect(250)
	}()
	return b, nil
}

// Publish sends a payload to a topic, and waits to know it went.
func (b *Bus) Publish(topic, payload string) error {
	if b == nil || b.client == nil {
		return errors.New("no MQTT broker is configured: set mqtt.broker in workspace.yaml")
	}
	token := b.client.Publish(topic, 0, false, payload)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("publishing to %s took too long", topic)
	}
	return token.Error()
}

// Apply lands one message: the device for its topic gets the payload as
// its state and now as when it was seen, made if it is new. A JSON
// payload is kept as written, so a page and the assistant can read its
// fields.
func Apply(st *store.Store, topic, payload string, now time.Time) error {
	if _, ok := st.Types().Get(DeviceType); !ok {
		return fmt.Errorf("this workspace has no %s type", DeviceType)
	}
	payload = strings.TrimSpace(payload)
	if json.Valid([]byte(payload)) {
		var buf bytes.Buffer
		if json.Indent(&buf, []byte(payload), "", "  ") == nil {
			payload = buf.String()
		}
	}
	fields := map[string]any{"state": payload, "seen": now.UTC().Format(time.RFC3339)}
	recs, err := st.List(DeviceType, store.ListOptions{})
	if err != nil {
		return err
	}
	for _, rec := range recs {
		if t, _ := rec.Fields["topic"].(string); t == topic {
			_, err := st.Update(DeviceType, rec.ID, fields)
			return err
		}
	}
	fields["topic"] = topic
	fields["name"] = nameOf(topic)
	_, err = st.Create(DeviceType, fields)
	return err
}

// nameOf is a device named for its topic: the last meaningful part,
// with the words a person would use. zigbee2mqtt/kitchen_light is
// kitchen light; home/upstairs/temp is upstairs temp.
func nameOf(topic string) string {
	parts := strings.Split(strings.Trim(topic, "/"), "/")
	var keep []string
	for i, p := range parts {
		if i == 0 && len(parts) > 1 {
			continue
		}
		if p == "state" || p == "status" || p == "tele" || p == "stat" || p == "" {
			continue
		}
		keep = append(keep, strings.ReplaceAll(strings.ReplaceAll(p, "_", " "), "-", " "))
	}
	if len(keep) == 0 {
		return topic
	}
	if len(keep) > 2 {
		keep = keep[len(keep)-2:]
	}
	return strings.Join(keep, " ")
}
