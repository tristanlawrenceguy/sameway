package schema

// Only is the types keep says yes to, as a set of their own: what one
// person may reach, for a search or a tool working for them.
func (s *Set) Only(keep func(*Type) bool) *Set {
	out := &Set{byName: map[string]*Type{}}
	for _, t := range s.Types {
		if keep(t) {
			out.Types = append(out.Types, t)
			out.byName[t.Name] = t
		}
	}
	return out
}

// NotOwners is every type but those that are the owner's alone (Owners):
// what someone else the owner let in may reach.
func (s *Set) NotOwners() *Set { return s.Only(func(t *Type) bool { return !t.Owners }) }
