// Package update keeps sameway current. A release published on GitHub is
// found, checked against the sha256 sums published beside it, and put
// where the running binary is; it runs from the next start.
//
// A person chooses how that happens. update.mode in workspace.yaml is
// "auto", which installs a new version on its own and says so in the
// activity log, or "manual", which only says a new version is there and
// waits to be asked: by telling the assistant to update, or by running
// `sameway update`.
//
// A build that cannot say which version it is — one built from source,
// which says "dev" — never replaces itself. It would be replacing work
// newer than the release with the release.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Version is this build's version, stamped by a release build with
//
//	-ldflags "-X github.com/tristanlawrenceguy/sameway/internal/update.Version=0.4.0"
//
// A build from source leaves it as it is.
var Version = "dev"

// DefaultRepo is where sameway's own releases are published.
const DefaultRepo = "tristanlawrenceguy/sameway"

const defaultAPI = "https://api.github.com"

// The ways a new version can arrive.
const (
	Auto   = "auto"
	Manual = "manual"
)

// Modes are the ways a new version can arrive, for the setting.
var Modes = []string{Auto, Manual}

// Release is one published version of sameway.
type Release struct {
	Version string `json:"version"`
	Notes   string `json:"notes,omitempty"`
	// Page is the release for a person to read.
	Page string `json:"page,omitempty"`
	// Asset is the download for this operating system and processor, and
	// Name is what the sums file calls it. Sums is checksums.txt from the
	// same release; without it nothing is installed.
	Asset string `json:"asset,omitempty"`
	Name  string `json:"name,omitempty"`
	Sums  string `json:"sums,omitempty"`
	Size  int64  `json:"size,omitempty"`
}

// Outcome is what a check or an install came to. Says is the whole of it
// in one sentence, which is what a person, a log entry and the assistant
// all need.
type Outcome struct {
	Current   string `json:"current"`
	Latest    string `json:"latest,omitempty"`
	Newer     bool   `json:"newer"`
	Installed bool   `json:"installed"`
	Notes     string `json:"notes,omitempty"`
	Page      string `json:"page,omitempty"`
	// Path is the binary that was replaced.
	Path string `json:"path,omitempty"`
	Says string `json:"says"`
}

// Updater finds and installs releases.
type Updater struct {
	// Current is the running version; empty means this build's.
	Current string
	// Repo is owner/name on GitHub; empty means sameway's own.
	Repo string
	// API is where releases are read from; empty means GitHub. Tests and
	// forks point it elsewhere.
	API string
	// Exe is the binary to replace; empty means the running one.
	Exe string
	// HTTP fetches releases; empty means a client with a timeout.
	HTTP *http.Client
}

func (u Updater) current() string {
	if u.Current != "" {
		return u.Current
	}
	return Version
}

func (u Updater) client() *http.Client {
	if u.HTTP != nil {
		return u.HTTP
	}
	return &http.Client{Timeout: 5 * time.Minute}
}

// Latest is the newest release published, with the download for this
// machine picked out of it.
func (u Updater) Latest(ctx context.Context) (*Release, error) {
	repo, api := u.Repo, u.API
	if repo == "" {
		repo = DefaultRepo
	}
	if api == "" {
		api = defaultAPI
	}
	url := strings.TrimSuffix(api, "/") + "/repos/" + repo + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "sameway/"+u.current())
	resp, err := u.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach %s to look for a new version: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s has published no releases yet, so there is nothing to update to", repo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s when asked for the latest release", url, resp.Status)
	}
	var body struct {
		Tag    string `json:"tag_name"`
		Notes  string `json:"body"`
		Page   string `json:"html_url"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%s answered something that is not a release: %w", url, err)
	}
	if strings.TrimSpace(body.Tag) == "" {
		return nil, fmt.Errorf("%s answered a release with no tag", url)
	}
	rel := &Release{Version: strings.TrimPrefix(strings.TrimSpace(body.Tag), "v"), Notes: strings.TrimSpace(body.Notes), Page: body.Page}
	for _, a := range body.Assets {
		switch {
		case strings.EqualFold(a.Name, sumsFile):
			rel.Sums = a.URL
		case rel.Asset == "" && forThisMachine(a.Name):
			rel.Asset, rel.Name, rel.Size = a.URL, a.Name, a.Size
		}
	}
	return rel, nil
}

// sumsFile is the list of sha256 sums a release publishes beside its
// downloads, in the form `shasum -a 256` writes: the sum, spaces, the
// file name.
const sumsFile = "checksums.txt"

// Run looks for a new version and installs it when told to. It comes back
// with what it found either way, and with an error only when something
// went wrong: nothing newer is not an error.
func (u Updater) Run(ctx context.Context, install bool) (Outcome, error) {
	rel, err := u.Latest(ctx)
	if err != nil {
		return Outcome{Current: u.current(), Says: err.Error()}, err
	}
	out := u.compare(rel)
	if !install || !out.Newer {
		return out, nil
	}
	where, err := u.install(ctx, rel)
	if err != nil {
		return out, err
	}
	out.Installed, out.Path, out.Says = true, where, Installed(rel.Version)
	return out, nil
}

// Check is Run without installing anything.
func (u Updater) Check(ctx context.Context) (Outcome, error) { return u.Run(ctx, false) }

// Installed is the sentence a finished install reads as, in one place
// because the log entry, the terminal and the assistant all say it.
func Installed(version string) string {
	return "sameway " + version + " is installed; it runs from the next start"
}

// AskFor is what a person does about a new version themselves, added to
// the sentence when the mode is manual and nothing was installed for them.
const AskFor = "; ask the assistant to update, or run `sameway update`"

// compare says where this build stands against a release.
func (u Updater) compare(rel *Release) Outcome {
	have := u.current()
	out := Outcome{Current: have, Latest: rel.Version, Notes: rel.Notes, Page: rel.Page, Newer: Newer(have, rel.Version)}
	switch {
	case !Known(have):
		out.Says = fmt.Sprintf("this build says %q, so it cannot tell whether it is current and will not replace itself; sameway %s is the latest release", have, rel.Version)
	case out.Newer:
		out.Says = "sameway " + rel.Version + " is out; this is " + have
	default:
		out.Says = "sameway " + have + " is the latest"
	}
	return out
}

// AssetName is what sameway's own releases call the download for this
// machine: sameway_<version>_<system>_<processor>, with .exe on Windows.
// The release build names its files this way; forThisMachine is looser
// than this on purpose, so a differently named release still works.
func AssetName(version string) string {
	name := "sameway_" + strings.TrimPrefix(version, "v") + "_" + runtime.GOOS + "_" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// forThisMachine says whether a release file is the program for this
// operating system and processor. A release may name the rest of the file
// however it likes; the words for the system have to be in it, and it has
// to be the program or an archive holding one, not something published
// beside it such as a checksum or a signature.
func forThisMachine(name string) bool {
	lower := strings.ToLower(name)
	if !anyOf(lower, systems()) || !anyOf(lower, processors()) {
		return false
	}
	for _, archive := range []string{".tar.gz", ".tgz", ".zip", ".exe"} {
		if strings.HasSuffix(lower, archive) {
			return true
		}
	}
	// Otherwise it has to be the program itself, which carries no file
	// type at all.
	return fileType(lower) == ""
}

// fileType is the suffix that says what kind of file this is: a dot, then
// letters and digits to the end of the name. The dots in a version are
// not one, because the rest of the name follows them — which is why
// sameway_0.4.0_linux_amd64 is a program and sameway_0.4.0_linux_amd64.sha256
// is not. path.Ext cannot tell the two apart.
func fileType(name string) string {
	dot := strings.LastIndexByte(name, '.')
	if dot < 0 {
		return ""
	}
	suffix := name[dot+1:]
	if suffix == "" || strings.IndexFunc(suffix, notTypeRune) >= 0 {
		return ""
	}
	return suffix
}

func notTypeRune(r rune) bool {
	return !('a' <= r && r <= 'z') && !('0' <= r && r <= '9')
}

// systems and processors are the names releases use for this machine.
func systems() []string {
	if runtime.GOOS == "darwin" {
		return []string{"darwin", "macos", "mac"}
	}
	return []string{runtime.GOOS}
}

func processors() []string {
	switch runtime.GOARCH {
	case "amd64":
		return []string{"amd64", "x86_64", "x64"}
	case "arm64":
		return []string{"arm64", "aarch64"}
	}
	return []string{runtime.GOARCH}
}

func anyOf(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
