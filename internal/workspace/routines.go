package workspace

// Routines are what Sameway does for the person on a schedule of their
// choosing, each set from Help or Workspaces and kept in workspace.yaml.
type Routines struct {
	// Brief.At is when the morning brief is sent, "07:30"; "" sends none.
	Brief struct {
		At string `yaml:"at"`
	} `yaml:"brief"`
	// Backup.Folder is a cloud folder a daily whole copy goes to; "" none.
	Backup struct {
		Folder string `yaml:"folder"`
	} `yaml:"backup"`
	// Review.On is the weekday the weekly review says it is ready, in the
	// evening, "sunday"; "" says nothing (internal/server review.go).
	Review struct {
		On string `yaml:"on"`
	} `yaml:"review"`
}
