package server

import "html/template"

// pageOptions is what a page adds around its body: the parts of the shell
// it fills, how its heading and window title read, and its status.
type pageOptions struct {
	QuietTitle bool
	// Said is the window's title when it says more than the heading: the
	// outcome of a search, heard first when the page arrives.
	Said         string
	Shell        string
	Kicker       template.HTML
	Lede         template.HTML
	Dot          int
	Left         template.HTML
	Right        template.HTML
	Header       template.HTML
	Footer       template.HTML
	JSONURL      string
	Focus        string
	FocusLabel   string
	Status       int
	ExtraScripts []template.HTML
}
