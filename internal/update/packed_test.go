package update_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// A release carries the program under its own name and again packed for a
// person (Sameway-Linux-x64.tar.gz); an update takes the program, even
// when the packed one comes first.
func TestAnUpdateTakesTheProgramNotTheOnePackedForAPerson(t *testing.T) {
	t.Parallel()
	packed := "Sameway-" + runtime.GOOS + "-" + runtime.GOARCH + ".zip"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.9.0", "assets": []map[string]any{
			{"name": packed, "browser_download_url": "https://example.test/" + packed},
			{"name": update.AssetName("0.9.0"), "browser_download_url": "https://example.test/program"},
			{"name": "checksums.txt", "browser_download_url": "https://example.test/sums"},
		}})
	}))
	defer srv.Close()
	rel, err := update.Updater{Current: "0.3.0", Repo: "o/r", API: srv.URL}.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.Name != update.AssetName("0.9.0") {
		t.Errorf("the program under its own name, not %q", rel.Name)
	}
}
