package mcp_test

import (
	"os"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/testhome"
)

// TestMain keeps this computer's folders out of the tests (testhome).
func TestMain(m *testing.M) { os.Exit(testhome.Run(m)) }
