package chat

import (
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A person changes a setting by asking for it, so the settings are a tool
// rather than a page of forms. The list comes from the workspace package,
// which owns workspace.yaml, so a setting added there is offered here
// without anything else changing.

var settingTool = llm.Tool{
	Name:        "set_setting",
	Description: "Change one setting of this workspace when the person asks for it, and say so. Most are reversible and happen at once; the few that send the conversation or a secret somewhere else, let a program run, or open the workspace to others cannot be taken back, so calling this puts the question to the person for you and nothing changes until they say yes. The settings: " + workspace.SettingsDoc() + ". A setting that holds a key or a token takes the NAME of the environment variable that holds it, never the key.",
	Schema: obj(map[string]any{
		"key":   map[string]any{"type": "string", "enum": workspace.SettingKeys()},
		"value": map[string]any{"type": "string"},
	}, "key", "value"),
}

// setSetting changes one line of workspace.yaml and says what it now is.
func (s *Service) setSetting(key, value string) toolResult {
	if s.SetSetting == nil {
		return fail("this workspace has no settings file")
	}
	if err := s.SetSetting(key, value); err != nil {
		return fail("%v", err)
	}
	note := ""
	if key == "server.addr" {
		note = ", from the next start"
	}
	return toolResult{text: key + " is now " + value + note, change: &Change{Action: "set", Component: key, Detail: value}}
}
