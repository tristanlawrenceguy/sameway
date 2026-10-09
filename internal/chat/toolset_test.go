package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A model on this computer is given the tools a turn needs: the core,
// first and always, and the others its message speaks of or the
// conversation has used; a model elsewhere is given them all.
func TestAModelHereIsGivenTheToolsATurnNeeds(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	all := svc.Tools()
	names := func(ts []llm.Tool) string {
		var n []string
		for _, t := range ts {
			n = append(n, t.Name)
		}
		return "," + strings.Join(n, ",") + ","
	}
	svc.Provider = &llm.OpenAI{BaseURL: "https://openrouter.ai/api/v1", Model: "x"}
	if got := svc.ToolsFor(all, []llm.Message{{Role: llm.RoleUser, Content: "Add a task"}}); len(got) != len(all) {
		t.Errorf("a model elsewhere has all %d, got %d", len(all), len(got))
	}
	svc.Provider = &llm.OpenAI{BaseURL: llm.OllamaURL + "/v1", Model: "qwen"}
	task := names(svc.ToolsFor(all, []llm.Message{{Role: llm.RoleUser, Content: "Add a task for tomorrow: call the dentist"}}))
	if !strings.HasPrefix(task, ",find_records,") && !strings.Contains(task, ",create_record,") || strings.Contains(task, ",set_setting,") || strings.Contains(task, ",write_up_meeting,") {
		t.Errorf("a task needs the core, not the settings or meetings: %s", task)
	}
	if len(task) > 0 && strings.Count(task, ",")-1 >= len(all) {
		t.Errorf("fewer than all: %s", task)
	}
	slower := names(svc.ToolsFor(all, []llm.Message{{Role: llm.RoleUser, Content: "Make things move slower please"}}))
	if !strings.Contains(slower, ",set_setting,") {
		t.Errorf("a message about a setting brings set_setting: %s", slower)
	}
	followUp := names(svc.ToolsFor(all, []llm.Message{
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{Name: "write_up_meeting"}}},
		{Role: llm.RoleUser, Content: "and add Ana to it"},
	}))
	if !strings.Contains(followUp, ",write_up_meeting,") {
		t.Errorf("a tool the conversation used stays: %s", followUp)
	}
}
