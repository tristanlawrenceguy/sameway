package chat

import (
	"fmt"
	"sort"
	"strings"
)

// What the person sees of a tab on their phone. Layout now said each row's
// widths and heights as a desktop draws them, and a tab that read well
// there was, on a phone, a column of blocks the person scrolled through to
// find what they came for: the crew's Home ran to dozens of screens, the
// first two of them a note and a test message. Where the person's phone
// measured the tab, Layout now says what its first screen shows and how
// long the tab runs there, so what matters can be put first.

// phoneScreen is about how tall a phone's first screen is, in CSS pixels.
const phoneScreen = 700

// phoneSaid is the line on the phone, or "" where none measured the tab.
func phoneSaid(all []placed) string {
	type seen struct {
		name string
		top  int
	}
	var on []seen
	end := 0
	for _, p := range all {
		r, ok := p.seen["phone"]
		if !ok {
			continue
		}
		on = append(on, seen{p.called(), r.Top})
		end = max(end, r.Top+r.Height)
	}
	if len(on) == 0 {
		return ""
	}
	sort.SliceStable(on, func(i, j int) bool { return on[i].top < on[j].top })
	var first []string
	for _, s := range on {
		if s.top < phoneScreen {
			first = append(first, s.name)
		}
	}
	screens := (end + phoneScreen - 1) / phoneScreen
	line := fmt.Sprintf(" On the phone, the first screen (about %dpx) shows %s; the tab runs %spx there, about %d screens.", phoneScreen, strings.Join(first, ", "), px(end), screens)
	if screens > 3 {
		line += " Put what the person comes for first, and take away what they do not."
	}
	return line
}
