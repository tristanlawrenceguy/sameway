package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/testhome"
)

// TestMain keeps this computer's folders out of the tests (testhome). The
// model is Claude Code, signed in as the person running the bench: it
// keeps its sign-in in their home, which testhome moves, and every turn
// was "Not logged in". It is told where its own folder is first.
func TestMain(m *testing.M) {
	if os.Getenv("CLAUDE_CONFIG_DIR") == "" {
		if home, err := os.UserHomeDir(); err == nil {
			os.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
		}
	}
	os.Exit(testhome.Run(m))
}
