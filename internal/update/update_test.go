package update_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/update"
)

func TestNewerComparesVersionsAndRefusesWhatItCannotRead(t *testing.T) {
	cases := []struct {
		have, want string
		newer      bool
	}{
		{"0.3.0", "0.4.0", true},
		{"0.3.0", "v0.4.0", true},
		{"0.4.0", "0.4.1", true},
		{"0.9.0", "1.0.0", true},
		{"0.4", "0.4.1", true},
		{"0.4.0", "0.4.0", false},
		{"0.4.0", "0.3.9", false},
		{"1.0.0", "0.9.9", false},
		{"0.4.0-rc1", "0.4.0", true},
		{"0.4.0", "0.4.0-rc1", false},
		{"0.4.0-rc1", "0.4.0-rc2", true},
		// A build that cannot say what it is never counts as behind, and a
		// release that cannot say what it is is never ahead.
		{"dev", "0.4.0", false},
		{"0.4.0", "nightly", false},
		{"", "0.4.0", false},
	}
	for _, c := range cases {
		if got := update.Newer(c.have, c.want); got != c.newer {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.have, c.want, got, c.newer)
		}
	}
	if !update.Known("0.4.0") || update.Known("dev") {
		t.Error("Known should accept a version and refuse dev")
	}
}

func TestCheckSaysWhereThisBuildStands(t *testing.T) {
	srv := releases(t, release{version: "0.4.0", notes: "Habits."})
	for _, c := range []struct {
		have, contains string
		newer          bool
	}{
		{"0.3.0", "sameway 0.4.0 is out; this is 0.3.0", true},
		{"0.4.0", "sameway 0.4.0 is the latest", false},
		{"0.5.0", "sameway 0.5.0 is the latest", false},
		{"dev", "will not replace itself", false},
	} {
		out, err := update.Updater{Current: c.have, Repo: "o/r", API: srv.URL}.Check(context.Background())
		if err != nil {
			t.Fatalf("check as %s: %v", c.have, err)
		}
		if out.Newer != c.newer || !strings.Contains(out.Says, c.contains) {
			t.Errorf("as %s: newer=%v says %q, want newer=%v containing %q", c.have, out.Newer, out.Says, c.newer, c.contains)
		}
		if out.Latest != "0.4.0" || out.Notes != "Habits." {
			t.Errorf("as %s: latest=%q notes=%q", c.have, out.Latest, out.Notes)
		}
		if out.Installed {
			t.Errorf("as %s: a check installed something", c.have)
		}
	}
}

func TestNoReleasesYetSaysSo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	_, err := update.Updater{Current: "0.3.0", Repo: "o/r", API: srv.URL}.Check(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no releases yet") {
		t.Errorf("expected a clear error about no releases, got %v", err)
	}
}

// release is what a test publishes: one version, its download for this
// machine, and the sums beside it.
type release struct {
	version string
	notes   string
	// name is the download's file name; empty means one named for this
	// machine, which is what a real release publishes.
	name string
	// body is the download's bytes; empty means a plain program.
	body []byte
	// sums replaces the checksums file; nil means the right one, and
	// noSums leaves it out of the release altogether.
	sums   []byte
	noSums bool
}

// releases serves the GitHub release API for one release, with its files
// pointing back at the same server, the way a real release does.
func releases(t *testing.T, rel release) *httptest.Server {
	t.Helper()
	if rel.name == "" {
		rel.name = update.AssetName(rel.version)
	}
	if rel.body == nil {
		rel.body = []byte("#!/sameway " + rel.version + "\n")
	}
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/download/"+rel.name, func(w http.ResponseWriter, r *http.Request) { w.Write(rel.body) })
	mux.HandleFunc("/download/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		if rel.sums != nil {
			w.Write(rel.sums)
			return
		}
		sum := sha256.Sum256(rel.body)
		fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), rel.name)
	})
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		assets := []map[string]any{{"name": rel.name, "browser_download_url": srv.URL + "/download/" + rel.name, "size": len(rel.body)}}
		if !rel.noSums {
			assets = append(assets, map[string]any{"name": "checksums.txt", "browser_download_url": srv.URL + "/download/checksums.txt"})
		}
		json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v" + rel.version,
			"body":     rel.notes,
			"html_url": "https://example.test/releases/v" + rel.version,
			"assets":   assets,
		})
	})
	return srv
}
