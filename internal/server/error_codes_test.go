package server_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every error code the API sends is one describe names, with what it
// means: an agent reads describe to know what it may get back, and a code
// it was never told of is one it cannot act on.
func TestEveryErrorCodeIsDescribed(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var routes map[string]any
	decode(t, get(t, h, "/api/describe/routes"), &routes)
	said, _ := routes["errors"].(string)
	files, _ := filepath.Glob("*.go")
	codes := regexp.MustCompile(`Code: +"([a-z_]+)"`)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, _ := os.ReadFile(f)
		for _, m := range codes.FindAllStringSubmatch(string(src), -1) {
			if !strings.Contains(said, m[1]) {
				t.Errorf("%s sends the error code %s, which describe's errors do not name", f, m[1])
			}
		}
	}
}
