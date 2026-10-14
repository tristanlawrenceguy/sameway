package schema

import "sort"

// Words Sameway once gave a value of its own and has since put right. A
// reminder that rang and was never dismissed said "Ringing" on its page
// eight days after its time (crew finding 0727): ringing is now, and the
// clock says what is ringing now; the state says it rang. A workspace made
// before keeps its copy of the type, so a word still as Sameway gave it is
// brought up to date; a word the person chose is theirs and is kept.

// retiredLabel is a value's old built-in name.
type retiredLabel struct{ typ, field, value, old string }

var retiredLabels = []retiredLabel{
	{"reminder", "state", "rang", "Ringing"},
}

// retireLabels gives each retired name still in use the built-in one.
func (s *Set) retireLabels(builtin *Set) {
	for _, r := range retiredLabels {
		t, ok := s.byName[r.typ]
		b, bok := builtin.byName[r.typ]
		if !ok || !bok {
			continue
		}
		bf, ok := b.Field(r.field)
		if !ok || bf.Labels[r.value] == "" || bf.Labels[r.value] == r.old {
			continue
		}
		for i := range t.Fields {
			if f := &t.Fields[i]; f.Name == r.field && f.Labels[r.value] == r.old {
				f.Labels[r.value] = bf.Labels[r.value]
			}
		}
	}
}

// settle ends Complete: retired names brought up to date, then the types
// in order of name.
func (s *Set) settle(builtin *Set) {
	s.retireLabels(builtin)
	sort.Slice(s.Types, func(i, j int) bool { return s.Types[i].Name < s.Types[j].Name })
}
