package ui

// The words a prop takes from a fixed set, one type per prop. Enums lists
// every one against its manifest prop, and manifest_test.go fails when the
// manifest gains or loses a word that is not here.

// ButtonType is a button's type.
type ButtonType string

// Variant is how loud a button is.
type Variant string

// LinkLook is whether a link reads as text or as a button.
type LinkLook string

// AlertKind is how serious an alert is.
type AlertKind string

// FieldType is what a text field takes.
type FieldType string

// StatusState is where work under way has got to.
type StatusState string

const (
	TypeButton ButtonType = "button"
	Submit     ButtonType = "submit"
	Reset      ButtonType = "reset"

	Primary   Variant = "primary"
	Secondary Variant = "secondary"
	Danger    Variant = "danger"
	Quiet     Variant = "quiet"

	LookText   LinkLook = "text"
	LookButton LinkLook = "button"

	Info    AlertKind = "info"
	Success AlertKind = "success"
	Warning AlertKind = "warning"
	Problem AlertKind = "danger"

	Text     FieldType = "text"
	Email    FieldType = "email"
	URL      FieldType = "url"
	Search   FieldType = "search"
	Number   FieldType = "number"
	Password FieldType = "password"

	Idle    StatusState = "idle"
	Working StatusState = "working"
	Done    StatusState = "done"
	Failed  StatusState = "error"
)

// Enums is every fixed set above, by component and prop.
var Enums = map[[2]string][]string{
	{"button", "type"}:     {string(TypeButton), string(Submit), string(Reset)},
	{"button", "variant"}:  {string(Primary), string(Secondary), string(Danger), string(Quiet)},
	{"link", "look"}:       {string(LookText), string(LookButton)},
	{"alert", "kind"}:      {string(Info), string(Success), string(Warning), string(Problem)},
	{"text-field", "type"}: {string(Text), string(Email), string(URL), string(Search), string(Number), string(Password)},
	{"status", "state"}:    {string(Idle), string(Working), string(Done), string(Failed)},
}
