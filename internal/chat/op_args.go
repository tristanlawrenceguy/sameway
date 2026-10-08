package chat

// toolArgs is every argument any op takes, read once from a call, and
// handed to its Run (op.go).
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

// look is the layout a call asks for.
func (a toolArgs) look() look {
	return look{Span: a.Span, Position: a.Position, Frame: a.Frame, Tone: a.Tone, Region: a.Region, Size: a.Size, Canvas: deref(a.Canvas), SetCanvas: a.Canvas != nil}
}
