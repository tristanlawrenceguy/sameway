package devices

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/discover"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Broker is a broker found on the home network, said for a person: what
// it is and where.
type Broker struct {
	Name string // Home Assistant, or as it announced itself
	Addr string // tcp://192.168.1.10:1883
}

// mqttPort is where a broker listens unless told otherwise.
const mqttPort = "1883"

// Find looks on the home network for a broker, three ways: one that
// announces itself as one (_mqtt._tcp); Home Assistant, whose Mosquitto
// add-on, the broker most homes with a hub run, listens on 1883 beside
// it; and one on this computer. A port that does not answer is left out,
// so what is offered is there now.
func Find(ctx context.Context, wait time.Duration) ([]Broker, error) {
	found, err := discover.Browse(ctx, wait, "_mqtt._tcp", "_home-assistant._tcp")
	var out []Broker
	seen := map[string]bool{}
	add := func(name, host, port string) {
		addr := net.JoinHostPort(host, port)
		if host == "" || seen[addr] || !answers(addr) {
			return
		}
		seen[addr] = true
		out = append(out, Broker{Name: name, Addr: "tcp://" + addr})
	}
	for _, s := range found {
		host := s.Addr
		if host == "" {
			host = s.Host
		}
		switch s.Type {
		case "_mqtt._tcp":
			add(s.Name, host, strconv.Itoa(s.Port))
		case "_home-assistant._tcp":
			add("Home Assistant", host, mqttPort)
		}
	}
	add("This computer", "127.0.0.1", mqttPort)
	if len(out) > 0 {
		return out, nil
	}
	return nil, err
}

// answers says whether something listens at addr.
func answers(addr string) bool {
	c, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// Home is how the program reaches the home's broker. The zero value is
// the real network; a test gives its own Connect, so nothing it runs
// needs a broker. Finding one is not here: listening on the network is
// the real program's alone (cli wires Find into the server), so a test
// never sets off the computer's firewall.
type Home struct {
	// Connect reaches a broker for as long as ctx lasts, with subscribed
	// topics landing through write, and returns what publishes to it;
	// nil is Start.
	Connect func(ctx context.Context, cfg Config, st *store.Store, write Write) (publish func(topic, payload string) error, err error)
}

// Reach is Connect, or Start.
func (h Home) Reach(ctx context.Context, cfg Config, st *store.Store, write Write) (func(topic, payload string) error, error) {
	if h.Connect != nil {
		return h.Connect(ctx, cfg, st, write)
	}
	bus, err := Start(ctx, cfg, st, write)
	if err != nil || bus == nil {
		return nil, err
	}
	return bus.Publish, nil
}
