package chat

import (
	"context"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// toolArgs is every argument any tool takes, read once from a call.
type toolArgs struct {
	File        string         `json:"file"`
	Mapping     map[string]any `json:"mapping"`
	Component   string         `json:"component"`
	ID          string         `json:"id"`
	Props       map[string]any `json:"props"`
	Span        *int           `json:"span"`
	Position    *int           `json:"position"`
	Frame       string         `json:"frame"`
	Tone        string         `json:"tone"`
	Region      string         `json:"region"`
	Size        string         `json:"size"`
	Canvas      *string        `json:"canvas"`
	Name        string         `json:"name"`
	Summary     string         `json:"summary"`
	Tool        string         `json:"tool"`
	Type        string         `json:"type"`
	Fields      map[string]any `json:"fields"`
	Query       string         `json:"query"`
	Where       []string       `json:"where"`
	Order       string         `json:"order"`
	Limit       int            `json:"limit"`
	Page        int            `json:"page"`
	Key         string         `json:"key"`
	Version     string         `json:"version"`
	Install     bool           `json:"install"`
	Value       string         `json:"value"`
	Kind        string         `json:"kind"`
	Description string         `json:"description"`
	Values      []string       `json:"values"`
	To          string         `json:"to"`
	Required    bool           `json:"required"`
	Default     any            `json:"default"`
	Properties  []fieldDef     `json:"properties"`
	Fills       map[string]any `json:"fills"`
	Event       string         `json:"event"`
	Recording   string         `json:"recording"`
	Decisions   []meetingItem  `json:"decisions"`
	Tasks       []meetingItem  `json:"tasks"`
	How         string         `json:"how"`
	Piece       string         `json:"piece"`
	Parts       []string       `json:"parts"`
	Material    []materialItem `json:"material"`
	Field       string         `json:"field"`
	Edits       []suggested    `json:"edits"`
}

// toolHandlers is what each tool does, by its name: one table, where a
// switch on the name was once kept in step with the list of tools by
// hand. TestEveryToolHasAHandler holds the two together. A function, not a
// variable, because some of these lead back to running a tool.
func toolHandlers() map[string]func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
	return map[string]func(s *Service, a toolArgs, call llm.ToolCall) toolResult{
		"propose_change": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.proposeByModel(a.Summary, call.Args)
		},
		"look_at_page": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.lookAtPage(call.Args)
		},
		"create_canvas": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.createCanvas(a.Name)
		},
		"remove_canvas": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.removeCanvas(a.ID)
		},
		"create_record": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.createRecord(a.Type, a.Fields)
		},
		"organise_writing": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.organiseWriting(a.Type, a.Piece, a.Parts, a.Material)
		},
		"suggest_edits": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.suggestEdits(a.Type, a.ID, a.Field, a.Edits)
		},
		"record_meeting": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			now := time.Now()
			if s.Now != nil {
				now = s.Now()
			}
			return s.askToRecord(a.Event, a.How, now)
		},
		"write_up_meeting": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.writeUpMeeting(a.Event, a.Recording, a.Summary, a.Decisions, a.Tasks)
		},
		"import_records": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.importRecords(a.Type, a.File, a.Mapping)
		},
		"update_record": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.updateRecord(a.Type, a.ID, a.Fields, a.Version)
		},
		"find_records": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.findRecords(a.Type, a.Query, a.Where, a.Order, a.Limit)
		},
		"get_record": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.getRecord(a.Type, a.ID)
		},
		"add_field": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.addField(a.Type, fieldDef{Name: a.Name, Kind: a.Kind, Description: a.Description, Values: a.Values, To: a.To, Required: a.Required, Default: a.Default})
		},
		"add_type": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.addType(a.Name, a.Description, a.Properties)
		},
		"add_component": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.addComponent(a.Component, a.Props, look{Span: a.Span, Position: a.Position, Frame: a.Frame, Tone: a.Tone, Region: a.Region, Size: a.Size, Canvas: deref(a.Canvas), SetCanvas: a.Canvas != nil})
		},
		"update_component": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.updateComponent(a.ID, a.Props, look{Span: a.Span, Position: a.Position, Frame: a.Frame, Tone: a.Tone, Region: a.Region, Size: a.Size, Canvas: deref(a.Canvas), SetCanvas: a.Canvas != nil})
		},
		"remove_component": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.removeBlock(a.ID)
		},
		"undo_change": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.undo(a.ID)
		},
		"add_arrangement": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.addArrangement(a.Name, a.Fills)
		},
		"search": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.search(a.Query, a.Type, a.Page)
		},
		"run_action": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			// What cannot be taken back is asked first; see consent.go.
			if r, ask := s.askFirst("run_action", a.ID, "", ""); ask {
				return r
			}
			return s.Run(context.Background(), a.ID, s.current)
		},
		"accept_action": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.acceptAction(context.Background(), a.ID)
		},
		"update_sameway": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.updateSameway(a.Install)
		},
		"let_in": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.letInCall(call.Args)
		},
		"set_setting": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			if r, ask := s.askFirst("set_setting", "", a.Key, a.Value); ask {
				return r
			}
			return s.setSetting(a.Key, a.Value)
		},
		"clear_canvas": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.clearCanvas()
		},
		"arrange_canvas": func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.arrangeCall(call.Args)
		},
		"clear_conversation": func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		"write_down":         func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		"add_workspace":      func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		"open_workspace":     func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		"restore_workspace":  func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
		"take_agent_away":    func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.homeTool(call) },
	}
}

// HandledTools names every tool with a handler, for the test that each
// tool offered has one and each handler is for a tool.
func HandledTools() []string {
	var names []string
	for name := range toolHandlers() {
		names = append(names, name)
	}
	return names
}
