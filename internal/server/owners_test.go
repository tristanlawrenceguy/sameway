package server_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Whose a kind of record is, the schema says, and a workspace whose copy
// of the schema predates saying so still keeps the owner's conversations
// and log the owner's.
func TestAnOlderWorkspaceKeepsTheOwnersRecordsTheirs(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"activity", "message", "conversation", "proposal"} {
		p := filepath.Join(dir, "schema", name+".yaml")
		raw, _ := os.ReadFile(p)
		os.WriteFile(p, []byte(strings.Replace(string(raw), "owners: true\n", "", 1)), 0o644)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	h := server.New(a)
	viewer := chat.Visitor{Name: "Vi", Login: "vi@example.com", Access: chat.View}
	for _, path := range []string{"/t/activity", "/api/message", "/t/conversation", "/export/activity.csv"} {
		if res := as(t, h, viewer, http.MethodGet, path, "", ""); res.Code != http.StatusForbidden {
			t.Errorf("%s is the owner's in an older workspace too: %d", path, res.Code)
		}
	}
}
