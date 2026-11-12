//go:build !windows

package server

import "io/fs"

// markedHidden is whether a file is marked hidden beside its name. Only
// Windows marks them; elsewhere a hidden name starts with a dot.
func markedHidden(fs.FileInfo) bool { return false }
