// Package design embeds the Sameway design system so the binary ships with it.
//
// The same folder is published standalone as an npm package; nothing in here
// depends on Go. See design/README.md.
package design

import "embed"

// FS holds tokens, base styles, every built-in component folder, and the
// built-in arrangements.
//
//go:embed tokens base components arrangements
var FS embed.FS
