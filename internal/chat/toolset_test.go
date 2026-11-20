package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A model is given the tools a turn needs: the core, first and always,
// and the others the assistant asked for or the conversation has used; one
// elsewhere the same as one on this computer.
func TestAModelIsGivenTheToolsATurnNeeds(t *testing.T) {
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
	if got := names(svc.ToolsFor(all, []llm.Message{{Role: llm.RoleUser, Content: "Add a task"}})); strings.Contains(got, ",set_setting,") || !strings.Contains(got, ",create_record,") {
		t.Errorf("a model elsewhere is given what the turn needs too: %s", got)
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
	if strings.Contains(slower, ",set_setting,") {
		t.Errorf("the person's words bring no tool, in any language: %s", slower)
	}
	asked := names(svc.ToolsFor(all, []llm.Message{
		{Role: llm.RoleUser, Content: "Que tout bouge plus lentement"},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{Name: "more_tools", Args: []byte(`{"name":"set_setting"}`)}}},
	}))
	if !strings.Contains(asked, ",set_setting,") {
		t.Errorf("asked for by name, set_setting is given: %s", asked)
	}
	followUp := names(svc.ToolsFor(all, []llm.Message{
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{Name: "write_up_meeting"}}},
		{Role: llm.RoleUser, Content: "and add Ana to it"},
	}))
	if !strings.Contains(followUp, ",write_up_meeting,") {
		t.Errorf("a tool the conversation used stays: %s", followUp)
	}
}
