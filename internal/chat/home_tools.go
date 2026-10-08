package chat

import (
	"encoding/json"
	"errors"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// What a person can do from a page the assistant can do when asked:
// start the conversation afresh, have a recording written down, and make,
// open or restore a workspace. Each was a page's alone until a test made
// every page action name its tool (server/tools_parity_test.go).

// Home is what the server does for the assistant beyond the workspace's
// records: the machine's speech-to-text and the other workspaces on it.
type Home interface {
	WriteDown(fileID string) (string, error)
	MakeWorkspace(name string, copy bool) (string, error)
	OpenWorkspace(name string) (string, error)
	RestoreWorkspace(name string) (string, error)
}

// workspaceName is a workspace, as an argument.
var workspaceName = map[string]any{"type": "string", "description": "The workspace's name, as the person calls it."}

var homeOps = []Op{
	{Title: "Clear the conversation", Traits: Traits{Destructive: true, Idempotent: true},
		Words: []string{"clear the chat", "clear the conversation", "forget"},
		Doing: saying("Clearing the conversation"),
		Run:   func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		Tool: llm.Tool{Name: "clear_conversation", Description: "Start this conversation afresh: its messages go and the canvas, its blocks and the other chats stay. Only when the person asks to start over or clear the chat. It can be undone, which puts the messages back.",
			Schema: obj(map[string]any{})}},
	{Title: "Write a recording down",
		Words: []string{"recording", "transcri", "audio", "voice note"},
		Doing: saying("Writing down a recording"),
		Run:   func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		Tool: llm.Tool{Name: "write_down", Description: "Have a recording (an audio or video file) written down as a transcript on this computer. The words arrive in its text as they are heard; the answer says whether it started or what the person must do instead.",
			Schema: obj(map[string]any{"file": map[string]any{"type": "string", "description": "The file record's id."}}, "file")}},
	{Title: "Make a workspace",
		Access: ForOwner,
		Words:  []string{"workspace"},
		Doing:  saying("Making a workspace"),
		Run:    func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		Tool: llm.Tool{Name: "add_workspace", Description: "Make a new workspace beside this one, blank or as a copy of this one, and open it in a window of its own. Only when the person asks for one.",
			Schema: obj(map[string]any{"name": workspaceName, "copy": map[string]any{"type": "boolean", "description": "True for a copy of this workspace with everything in it; false or left out for a blank one."}}, "name")}},
	{Title: "Open a workspace", Traits: Traits{Idempotent: true},
		Access: ForOwner,
		Words:  []string{"workspace"},
		Doing:  saying("Opening a workspace"),
		Run:    func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		Tool: llm.Tool{Name: "open_workspace", Description: "Open another workspace on this computer, starting it if it is not running, and say its address. An unknown name answers with the names there are.",
			Schema: obj(map[string]any{"name": workspaceName}, "name")}},
	{Title: "Restore a deleted workspace",
		Access: ForOwner,
		Words:  []string{"workspace", "trash"},
		Doing:  saying("Bringing a workspace back"),
		Run:    func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		Tool: llm.Tool{Name: "restore_workspace", Description: "Put a deleted workspace back from Sameway's trash, where it was, ready to open.",
			Schema: obj(map[string]any{"name": workspaceName}, "name")}},
	{Title: "Take an agent's key away", Traits: Traits{Destructive: true, Idempotent: true},
		Access: ForOwner,
		Words:  []string{"agent", "access", "key"},
		Doing:  saying("Changing who can use this"),
		Run:    func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		Tool: llm.Tool{Name: "take_agent_away", Description: "Take an agent's key away, so it can no longer reach this workspace, when the person asks. Keys are made at the command line with sameway agent add, never here, so a key never passes through the conversation. Undoing this lets the agent back in with the same key.",
			Schema: obj(map[string]any{"name": map[string]any{"type": "string", "description": "The agent's name, as the log calls it."}}, "name")}},
}

// homeTool runs one of the tools above, or says the tool is unknown.
func (s *Service) homeTool(call llm.ToolCall) toolResult {
	var args struct {
		File string `json:"file"`
		Name string `json:"name"`
		Copy bool   `json:"copy"`
	}
	if len(call.Args) > 0 {
		if err := json.Unmarshal(call.Args, &args); err != nil {
			return fail("%s", ArgsTrouble(err))
		}
	}
	if call.Name == "clear_conversation" {
		c, err := s.clearing()
		if err != nil {
			return fail("could not clear the conversation: %v", err)
		}
		return toolResult{text: "the conversation is cleared; the canvas and the other chats stayed, and undo_change puts the messages back", change: &c}
	}
	if call.Name == "take_agent_away" {
		c, err := records.TakeAgentAway(s.Store, args.Name)
		if err != nil {
			return fail("%v", err)
		}
		return toolResult{text: args.Name + "'s key no longer works; undo_change lets it back in", change: &c}
	}
	did := map[string]func() (string, error){
		"write_down":        func() (string, error) { return s.home().WriteDown(args.File) },
		"add_workspace":     func() (string, error) { return s.home().MakeWorkspace(args.Name, args.Copy) },
		"open_workspace":    func() (string, error) { return s.home().OpenWorkspace(args.Name) },
		"restore_workspace": func() (string, error) { return s.home().RestoreWorkspace(args.Name) },
	}[call.Name]
	if did == nil {
		return fail("unknown tool %s", call.Name)
	}
	said, err := did()
	if err != nil {
		return fail("%v", err)
	}
	return toolResult{text: said}
}

// home is the server's side, or one that says there is none: a workspace
// used without a server, from the command line, has no pages to open.
func (s *Service) home() Home {
	if s.Home != nil {
		return s.Home
	}
	return noHome{}
}

type noHome struct{}

var errNoHome = errors.New("this is not a running workspace, so there is nothing here to do that with")

func (noHome) WriteDown(string) (string, error)           { return "", errNoHome }
func (noHome) MakeWorkspace(string, bool) (string, error) { return "", errNoHome }
func (noHome) OpenWorkspace(string) (string, error)       { return "", errNoHome }
func (noHome) RestoreWorkspace(string) (string, error)    { return "", errNoHome }
