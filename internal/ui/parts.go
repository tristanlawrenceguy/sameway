package ui

// The builders, one per component, in the order a page most often uses
// them. Each field's prop tag is the prop's name in the component's
// manifest; manifest_test.go holds them to it. What a prop means is said
// in the manifest, not repeated here.

// Button is design/components/button: an action on this page.
type Button struct {
	Label    string     `prop:"label"`
	Context  string     `prop:"context"`
	Type     ButtonType `prop:"type"`    // Submit in a Form
	Variant  Variant    `prop:"variant"` // Primary when left out
	Name     string     `prop:"name"`
	Value    string     `prop:"value"`
	ID       string     `prop:"id"`
	Disabled bool       `prop:"disabled"`
	Busy     bool       `prop:"busy"`
	Pressed  *bool      `prop:"pressed"`
	Action   string     `prop:"action"`
	From     string     `prop:"from"`
}

// Link is design/components/link: going somewhere.
type Link struct {
	Href    string   `prop:"href"`
	Label   string   `prop:"label"`
	Context string   `prop:"context"`
	Current bool     `prop:"current"`
	Within  bool     `prop:"within"`
	Look    LinkLook `prop:"look"`
	ID      string   `prop:"id"`
}

// Alert is design/components/alert: something the person should know.
type Alert struct {
	Message string    `prop:"message"`
	Kind    AlertKind `prop:"kind"`
	Title   string    `prop:"title"`
	Level   int       `prop:"level"`
	Live    bool      `prop:"live"`
	ID      string    `prop:"id"`
	Icon    string    `prop:"icon"`
	Dismiss bool      `prop:"dismiss"`
}

// TextField is design/components/text-field: one line of text in a form.
type TextField struct {
	Label        string    `prop:"label"`
	Name         string    `prop:"name"`
	ID           string    `prop:"id"`
	Value        string    `prop:"value"`
	Type         FieldType `prop:"type"`
	Spellcheck   *bool     `prop:"spellcheck"` // on when left out
	Hint         string    `prop:"hint"`
	MaxLength    int       `prop:"maxlength"`
	Error        string    `prop:"error"`
	Required     bool      `prop:"required"`
	Autocomplete string    `prop:"autocomplete"`
}

// Empty is design/components/empty: a list or page with nothing in it yet.
type Empty struct {
	Message string `prop:"message"`
	Title   string `prop:"title"`
	Level   int    `prop:"level"`
	Quiet   bool   `prop:"quiet"`
	Live    bool   `prop:"live"`
	// Action is the one thing to do next, a link after the sentence.
	Action *EmptyAction `prop:"action"`
}

// EmptyAction is the empty state's action prop.
type EmptyAction struct {
	Href  string `prop:"href"`
	Label string `prop:"label"`
}

// Status is design/components/status: work under way, or how it ended.
type Status struct {
	Message string      `prop:"message"`
	State   StatusState `prop:"state"`
	Said    string      `prop:"said"`
	ID      string      `prop:"id"`
}

func (Button) Component() string    { return "button" }
func (Link) Component() string      { return "link" }
func (Alert) Component() string     { return "alert" }
func (TextField) Component() string { return "text-field" }
func (Empty) Component() string     { return "empty" }
func (Status) Component() string    { return "status" }

func (p Button) Props() map[string]any    { return props(p) }
func (p Link) Props() map[string]any      { return props(p) }
func (p Alert) Props() map[string]any     { return props(p) }
func (p TextField) Props() map[string]any { return props(p) }
func (p Empty) Props() map[string]any     { return props(p) }
func (p Status) Props() map[string]any    { return props(p) }
