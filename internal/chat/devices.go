package chat

import (
	home "github.com/tristanlawrenceguy/sameway/internal/devices"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A device that says it is on or off is turned the other way by asking,
// as by its page's button: the same publish to the home's hub
// (automate.SwitchDevice), the owner's alone, as the hub is.

var deviceOp = Op{Title: "Turn a device on or off", Traits: Traits{Idempotent: true, OpenWorld: true},
	Access: ForOwner,
	Words:  []string{"turn on", "turn off", "switch", "light", "lamp", "plug", "device"},
	Doing:  saying("Switching a device"),
	Offered: func(s *Service, _ *llm.Tool) bool {
		_, ok := s.Store.Types().Get(home.DeviceType)
		return ok
	},
	Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
		if a.To != "on" && a.To != "off" {
			return fail("to is on or off, not %q", a.To)
		}
		said, err := s.Automate.SwitchDevice(a.ID, a.To == "on")
		if err != nil {
			return fail("%v", err)
		}
		return toolResult{text: said + ". Its state changes when it answers; get_record reads it."}
	},
	Tool: llm.Tool{
		Name:        "switch_device",
		Description: "Turn one of the person's devices (a light, a plug) on or off now, through their home's hub. It works for a device whose state says ON or OFF and whose topic follows Zigbee2MQTT or Tasmota, or that names its own set topic; any other says so, and an mqtt action can send it what its maker says.",
		Schema: obj(map[string]any{
			"id": map[string]any{"type": "string", "description": "The device record's id, from find_records on device."},
			"to": map[string]any{"type": "string", "enum": []string{"on", "off"}},
		}, "id", "to"),
	}}
