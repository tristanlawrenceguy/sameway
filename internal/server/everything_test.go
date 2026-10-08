package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// The whole workspace is its owner's to take out, and offered where the
// workspace is described; nobody else can have it.
func TestOnlyTheOwnerTakesEverything(t *testing.T) {
	_, h := newApp(t)
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, `href="/export/workspace.zip"`) {
		t.Error("the workspaces page offers everything in one zip")
	}
	res := get(t, h, "/export/workspace.zip")
	if res.Code != http.StatusOK || res.Header().Get("Content-Type") != "application/zip" || !strings.HasPrefix(res.Body.String(), "PK") {
		t.Errorf("the owner gets a zip: %d %q", res.Code, res.Header().Get("Content-Type"))
	}
	hana := records.Visitor{Name: "Hana", Login: "hana@example.com", Access: records.Edit}
	if res := as(t, h, hana, http.MethodGet, "/export/workspace.zip", "", ""); strings.HasPrefix(res.Body.String(), "PK") {
		t.Error("someone else does not")
	}
}
