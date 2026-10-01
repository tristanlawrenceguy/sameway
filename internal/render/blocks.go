package render

// Blocks are the components that can be placed on a canvas: all but the
// ones Sameway places itself (PageOnly).
func (r *Registry) Blocks() []*Component {
	var out []*Component
	for _, c := range r.Components() {
		if !c.Manifest.PageOnly {
			out = append(out, c)
		}
	}
	return out
}
