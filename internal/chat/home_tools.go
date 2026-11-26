package chat

import (
	"encoding/json"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// What a person can do from a page the assistant can do when asked:
// start the conversation afresh, have a recording written down, and make,
// open or restore a workspace. Each was a page's alone until a test made
// every page action name its tool (server/tools_parity_test.go).

// Home is what the server does for the assistant beyond the workspace's
// records: the machine's speech-to-text, the other workspaces on it and
// the person's files there.
type Home interface {
	WriteDown(fileID string) (string, error)
	MakeWorkspace(name string, copy bool) (string, error)
	OpenWorkspace(name string) (string, error)
	RestoreWorkspace(name string) (string, error)
	FindFiles(words string) (string, error)
}

// workspaceName is a workspace, as an argument.
var workspaceName = map[string]any{"type": "string", "description": "The workspace's name, as the person calls it."}

var homeOps = []Op{
	{Title: "Clear the conversation", Traits: Traits{Destructive: true, Idempotent: true},
		Doing: saying("Clearing the conversation"),
		Run:   runHome,
		Tool: llm.Tool{Name: "clear_conversation", Description: "Start this conversation afresh, only when the person asks: its messages go, the canvas and the other chats stay. It can be undone.",
			Schema: obj(map[string]any{})}},
	{Title: "Write a recording down",
		Doing: saying("Writing down a recording"),
		Run:   runHome,
		Tool: llm.Tool{Name: "write_down", Description: "Have a recording (an audio or video file) written down as a transcript on this computer. The words arrive in its text as they are heard.",
			Schema: obj(map[string]any{"file": map[string]any{"type": "string", "description": "The file record's id."}}, "file")}},
	{Title: "Make a workspace", Traits: Traits{OpenWorld: true},
		Access: ForOwner,
		Doing:  saying("Making a workspace"),
		Run:    runHome,
		Tool: llm.Tool{Name: "add_workspace", Description: "Make a new workspace beside this one, blank or as a copy of this one, and open it in a window of its own. Only when the person asks for one.",
			Schema: obj(map[string]any{"name": workspaceName, "copy": map[string]any{"type": "boolean", "description": "True for a copy of this one; otherwise blank."}}, "name")}},
	{Title: "Open a workspace", Traits: Traits{Idempotent: true, OpenWorld: true},
		Access: ForOwner,
		Doing:  saying("Opening a workspace"),
		Run:    runHome,
		Tool: llm.Tool{Name: "open_workspace", Description: "Open another workspace on this computer, starting it if it is not running, and say its address.",
			Schema: obj(map[string]any{"name": workspaceName}, "name")}},
	{Title: "Restore a deleted workspace", Traits: Traits{OpenWorld: true},
		Access: ForOwner,
		Doing:  saying("Bringing a workspace back"),
		Run:    runHome,
		Tool: llm.Tool{Name: "restore_workspace", Description: "Put a deleted workspace back from Sameway's trash, where it was, ready to open.",
			Schema: obj(map[string]any{"name": workspaceName}, "name")}},
	{Title: "Take an agent's key away", Traits: Traits{Destructive: true, Idempotent: true},
		Access: ForOwner,
		Doing:  saying("Changing who can use this"),
		Run:    runHome,
		Tool: llm.Tool{Name: "take_agent_away", Description: "Take an agent's key away, so it can no longer reach this workspace, when the person asks. Keys are made at the command line with sameway agent add, never here. It can be undone.",
			Schema: obj(map[string]any{"name": map[string]any{"type": "string", "description": "The agent's name, as the log calls it."}}, "name")}},
	{Title: "Find a file on this computer", Traits: Traits{ReadOnly: true, Idempotent: true},
		Access: ForOwner,
		Doing:  saying("Looking on this computer"),
		Run:    runHome,
		Tool: llm.Tool{Name: "find_on_computer", Description: "Find files on this computer that are not in Sameway yet, by words in their names, in the person's Documents, Downloads and Desktop. Only the person brings one in, on the search page the answer links: pass the link on.",
			Schema: obj(map[string]any{"words": map[string]any{"type": "string", "description": "Words in the file's name, such as boiler warranty."}}, "words")}},
}

func runHome(s *Service, _ toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) }

// homeTool runs one of the tools above, or says the tool is unknown.
func (s *Service) homeTool(call llm.ToolCall) toolResult {
	var args struct {
		File  string `json:"file"`
		Name  string `json:"name"`
		Copy  bool   `json:"copy"`
		Words string `json:"words"`
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
	if s.Home == nil { // used without a server, from the command line
		return fail("this is not a running workspace, so there is nothing here to do that with")
	}
	did := map[string]func() (string, error){
		"write_down":        func() (string, error) { return s.Home.WriteDown(args.File) },
		"add_workspace":     func() (string, error) { return s.Home.MakeWorkspace(args.Name, args.Copy) },
		"open_workspace":    func() (string, error) { return s.Home.OpenWorkspace(args.Name) },
		"restore_workspace": func() (string, error) { return s.Home.RestoreWorkspace(args.Name) },
		"find_on_computer":  func() (string, error) { return s.Home.FindFiles(args.Words) },
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
