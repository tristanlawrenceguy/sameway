package chat

// How the tab reads, at the start of a turn, in the same words the model
// gets after every write and an agent gets from look (measured.layout):
// Layout now. It once came only after a write, so a model asked anything
// else on a tab that had gone wrong never knew; then the start had its own
// pieces (rows by span, flags beside blocks, the to-look-at list), which
// said each broken block twice and the rows in other words than after a
// write. One account, said once: the listing above it gives each block's
// facts, this gives how the page reads and what is wrong with it. The
// model may say it when it matters, and fix it when it changes the tab
// anyway; it is not asked to tidy what the person did not ask about.

// tabReads is Layout now for the tab the person is on, with when to act
// on what is to look at.
func (s *Service) tabReads() string {
	r := s.layoutReportOf(s.current) // layout_report.go
	if len(r.problems) == 0 {
		return r.String() + "\n"
	}
	return "Say what is to look at when it bears on what the person asks, and fix it when you change the tab anyway; do not tidy unasked. " + r.String() + "\n"
}
