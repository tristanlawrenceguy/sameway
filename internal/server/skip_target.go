package server

import "strings"

// skipToLatest offers "Skip to latest message" only when that message is
// on the page. A canvas whose chat block was removed, put at icon size or
// on another tab still has a latest message, but not here: the link led
// nowhere, which axe reported on Home in every in-app run of the agent
// evaluation (skip-link: the target should exist).
func skipToLatest(opts *pageOptions, latest, body string) {
	if latest == "" {
		return
	}
	target := `id="` + latest + `"`
	for _, part := range []string{body, string(opts.Left), string(opts.Right), string(opts.Header), string(opts.Footer)} {
		if strings.Contains(part, target) {
			opts.Focus, opts.FocusLabel = latest, "Skip to latest message"
			return
		}
	}
}
