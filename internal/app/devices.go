package app

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/tristanlawrenceguy/sameway/internal/devices"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The home's broker, connected while the workspace is served: at the
// start, and again whenever an mqtt setting changes (the press on
// Workspaces, the assistant's set_setting, an undo), so a broker set by
// either is used at once rather than from the next start.

// deviceLink is the broker connection, kept while the program serves.
type deviceLink struct {
	mu     sync.Mutex
	served context.Context    // how long the program serves; nil when it does not
	stop   context.CancelFunc // ends the connection there is
	err    error              // why the last one did not connect
}

// ConnectDevices reaches the broker workspace.yaml names, for as long as
// ctx lasts, and again with each change to its settings. It says where
// it connected, or "" when no broker is set.
func (a *App) ConnectDevices(ctx context.Context) (string, error) {
	a.devices.mu.Lock()
	a.devices.served = ctx
	a.devices.mu.Unlock()
	return a.reconnectDevices()
}

// DevicesLive says whether a change to the broker connects at once (the
// program is serving), and why the last connection failed, if it did.
func (a *App) DevicesLive() (bool, error) {
	a.devices.mu.Lock()
	defer a.devices.mu.Unlock()
	return a.devices.served != nil, a.devices.err
}

// reconnectDevices ends the connection there is and makes the one the
// settings say now; nothing while the program does not serve.
func (a *App) reconnectDevices() (string, error) {
	a.devices.mu.Lock()
	defer a.devices.mu.Unlock()
	if a.devices.served == nil {
		return "", nil
	}
	if a.devices.stop != nil {
		a.devices.stop()
		a.devices.stop = nil
	}
	a.Automate.Publish, a.devices.err = nil, nil
	cfg := a.Workspace.Config.MQTT
	if strings.TrimSpace(cfg.Broker) == "" {
		return "", nil
	}
	ctx, stop := context.WithCancel(a.devices.served)
	publish, err := a.opts.Home.Reach(ctx, cfg, a.Store, a.Records.Keeper(devices.DeviceType))
	if err != nil {
		stop()
		a.devices.err = err
		return "", err
	}
	a.devices.stop, a.Automate.Publish = stop, publish
	return fmt.Sprintf("%s, %d topic(s)", cfg.Broker, len(cfg.Subscribe)), nil
}

// settingChanged takes up a change to the model or the broker at once: a
// new model setting is a new model, so the next message goes to it; a
// new broker setting connects again, a failure kept for Workspaces.
func (a *App) settingChanged(key string) {
	if strings.HasPrefix(key, "llm.") {
		a.Chat.Provider, a.Chat.ProviderErr = llm.New(a.LLMConfig())
	}
	if strings.HasPrefix(key, "mqtt.") {
		if _, err := a.reconnectDevices(); err != nil {
			log.Printf("devices: %v", err)
		}
	}
}
