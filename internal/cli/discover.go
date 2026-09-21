package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/discover"
)

// discoverCmd lists what the local network announces: brokers, hubs,
// speakers, printers, anything with mDNS. It needs no workspace; it is
// the first look around before setting mqtt.broker or a webhook.
func (c *ctx) discoverCmd() error {
	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	wait := fs.Duration("wait", 3*time.Second, "how long to listen for answers")
	if err := fs.Parse(c.args); err != nil {
		return err
	}
	found, err := discover.Browse(context.Background(), *wait, fs.Args()...)
	if err != nil {
		return err
	}
	if c.JSON {
		return json.NewEncoder(c.Stdout).Encode(found)
	}
	if len(found) == 0 {
		fmt.Fprintln(c.Stdout, "Nothing announced itself in time. Try --wait 8s, or name a type: sameway discover _mqtt._tcp")
		return nil
	}
	for _, s := range found {
		fmt.Fprintln(c.Stdout, discover.Describe(s))
	}
	fmt.Fprintln(c.Stdout, "\nAn MQTT broker goes in workspace.yaml as mqtt.broker: tcp://<address>:<port>.")
	return nil
}
