package server_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server/servertest"
)

// TestMain points this computer's folders (the known list, copies,
// deleted workspaces, pasted keys) at a temp folder for the whole run, so
// an app a test opens with app.Load, which takes the person's own, never
// reads or writes theirs. newApp gives each test a folder of its own.
func TestMain(m *testing.M) { servertest.Main(m) }
