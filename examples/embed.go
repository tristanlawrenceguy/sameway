// Package examples embeds the example workspaces so `sameway init` can copy
// one without network access.
package examples

import "embed"

// FS holds workspaces/<name>/... for every bundled preset.
//
//go:embed workspaces
var FS embed.FS

// StarterRoot is the path inside FS of the default preset.
const StarterRoot = "workspaces/starter"
