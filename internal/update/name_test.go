package update

import (
	"runtime"
	"testing"
)

// Which file in a release is the program is decided by its name, and the
// name differs per machine, so the same rule has to be read on every one.
// These cases are written out rather than derived, because the version in
// a release file's name has dots in it and a rule that mistakes one of
// them for a file type rejects the program on the machines whose programs
// have no suffix.
func TestFileTypeTellsAVersionFromASuffix(t *testing.T) {
	for name, want := range map[string]string{
		"sameway_0.4.0_linux_amd64":        "",
		"sameway_0.4.0_darwin_arm64":       "",
		"sameway-0.4.0-linux-amd64":        "",
		"sameway_1_linux_amd64":            "",
		"sameway":                          "",
		"sameway_0.4.0_windows_amd64.exe":  "exe",
		"sameway_0.4.0_linux_amd64.sha256": "sha256",
		"sameway_0.4.0_linux_amd64.sig":    "sig",
		"sameway_0.4.0_linux_amd64.zip":    "zip",
		"checksums.txt":                    "txt",
		"sameway_0.4.0_linux_amd64.tar.gz": "gz",
		"trailing.":                        "",
	} {
		if got := fileType(name); got != want {
			t.Errorf("fileType(%q) = %q, want %q", name, got, want)
		}
	}
}

// The name this machine's release file has is the one the updater picks,
// and what is published beside it is left alone.
func TestTheProgramForThisMachineIsPicked(t *testing.T) {
	plain := AssetName("0.4.0")
	for name, want := range map[string]bool{
		plain:                       true,
		plain + ".zip":              true,
		plain + ".tar.gz":           true,
		plain + ".sha256":           false,
		plain + ".sig":              false,
		"checksums.txt":             false,
		"sameway_0.4.0_plan9_s390x": false,
	} {
		if got := forThisMachine(name); got != want {
			t.Errorf("forThisMachine(%q) = %v on %s/%s, want %v", name, got, runtime.GOOS, runtime.GOARCH, want)
		}
	}
}
