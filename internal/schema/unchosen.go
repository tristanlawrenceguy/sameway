package schema

import "fmt"

// Unchosen says a field holds only what it holds when nobody chose: its
// default, or no for a yes-or-no without one. A page says what was chosen
// about a record, so it leaves such a field out: an action that is not a
// web hook does not say its method, a habit not archived does not say
// Archived no. The record and its editor still have it.
func (f Field) Unchosen(v any) bool {
	if v == nil {
		return true
	}
	if f.Type == "bool" {
		on, _ := v.(bool)
		def, _ := f.Default.(bool)
		return on == def
	}
	return f.Default != nil && fmt.Sprint(v) == fmt.Sprint(f.Default)
}
