package chat

// What is wrong with the tab, at the start of a turn. Layout now, with
// what to look at (rows half empty, a block scrolling inside itself, one
// that cannot be shown or repeats another, a phone's first screen), came
// only after the model wrote a block. Asked anything else on a tab that
// had gone wrong, it never knew: the crew's Home kept its broken boxes
// through days of conversations on it. Given at the start, the model can
// say so when it matters, and fix it when it is changing the tab anyway;
// it is not asked to tidy what the person did not ask about.

// lookAtNow is the tab's to-look-at list from Layout now, or "".
func (s *Service) lookAtNow() string {
	r := s.layoutReportOf(s.current) // layout_report.go
	if len(r.problems) == 0 {
		return ""
	}
	return "On this tab, to look at (say it when it bears on what the person asks, and fix it when you change the tab anyway; do not tidy unasked):" + r.lookAt() + "\n"
}
