package devices

import (
	"encoding/json"
	"strings"
)

// A device that says it is on or off can be turned the other way with one
// press, by the convention the program behind its topic follows. Only two
// are known, the two the common subscriptions take in:
//
//   - Zigbee2MQTT: zigbee2mqtt/<name> says {"state":"ON"}; it is changed
//     by {"state":"OFF"} on zigbee2mqtt/<name>/set.
//   - Tasmota: stat/<name>/POWER says ON; it is changed by OFF on
//     cmnd/<name>/POWER.
//
// A device whose set field names a topic is changed there, in the shape
// its state came in. Anything else says nothing, rather than a guess sent
// to the person's hardware.

// Common are the subscriptions that take in the devices of the two
// programs above, saved with a broker found on the home network.
var Common = []string{"zigbee2mqtt/+", "stat/+/POWER"}

// OnOff reads a device's state: whether it is on, and whether it said.
func OnOff(fields map[string]any) (on, said bool) {
	word, _ := stateWord(fields)
	switch word {
	case "ON":
		return true, true
	case "OFF":
		return false, true
	}
	return false, false
}

// Command is the message that turns a device on or off: the topic and the
// payload, or ok false when its state or topic follows no known way.
func Command(fields map[string]any, on bool) (topic, payload string, ok bool) {
	if _, said := OnOff(fields); !said {
		return "", "", false
	}
	word := "OFF"
	if on {
		word = "ON"
	}
	_, isJSON := stateWord(fields)
	if isJSON {
		payload = `{"state":"` + word + `"}`
	} else {
		payload = word
	}
	if set, _ := fields["set"].(string); strings.TrimSpace(set) != "" {
		return strings.TrimSpace(set), payload, true
	}
	from, _ := fields["topic"].(string)
	parts := strings.Split(from, "/")
	switch {
	case isJSON && len(parts) == 2 && parts[0] == "zigbee2mqtt":
		return from + "/set", payload, true
	case !isJSON && len(parts) == 3 && parts[0] == "stat" && parts[2] == "POWER":
		return "cmnd/" + parts[1] + "/POWER", payload, true
	}
	return "", "", false
}

// stateWord is what a device's state says, ON, OFF or anything else, and
// whether it said it as JSON's state field.
func stateWord(fields map[string]any) (word string, isJSON bool) {
	state, _ := fields["state"].(string)
	state = strings.TrimSpace(state)
	if strings.HasPrefix(state, "{") {
		var v struct {
			State string `json:"state"`
		}
		if json.Unmarshal([]byte(state), &v) == nil {
			return v.State, true
		}
		return "", true
	}
	return state, false
}
