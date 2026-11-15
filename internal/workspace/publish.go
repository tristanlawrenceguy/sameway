package workspace

// Publish is what anyone on the internet may reach, people and AI services
// alike, with no login, through Tailscale Funnel at the workspace's
// tailnet address: tabs and content types to read, by name, comma
// separated, and asking for a time with the person. Empty is nothing; see
// server/publish.go.
type Publish struct {
	Tabs  string `yaml:"tabs"`
	Types string `yaml:"types"`
	// Book is on when anyone may ask for a time at /book; each request
	// waits on Today for the person's yes (server/book.go).
	Book string `yaml:"book"`
}
