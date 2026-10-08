package server_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// TestEveryAPIRouteIsDescribed: an agent learns the API from
// /api/describe, so every /api route the server answers is there, with its
// method. A route added without a line in describe is one no agent finds
// (backlog 0190, 0402).
func TestEveryAPIRouteIsDescribed(t *testing.T) {
	pattern := regexp.MustCompile(`^([A-Z]+) (/api/.*)$`)
	var routes [][]string
	for _, r := range server.Routes() {
		if m := pattern.FindStringSubmatch(r.Pattern); m != nil {
			routes = append(routes, m)
		}
	}
	if len(routes) == 0 {
		t.Fatal("found no /api routes in the route table")
	}

	_, h := newApp(t)
	var described map[string]any
	decode(t, get(t, h, "/api/describe/routes"), &described)
	for _, r := range routes {
		method, path := r[1], r[2]
		// A {placeholder} in the route stands for any one path segment in
		// the description, which may name it differently.
		segs := strings.Split(regexp.QuoteMeta(path), "/")
		for i, s := range segs {
			if strings.HasPrefix(s, `\{`) {
				segs[i] = `[^/\s]+`
			}
		}
		want := regexp.MustCompile(strings.Join(segs, "/") + `(\s|$|[,;:.)?])`)
		found := false
		for _, v := range described {
			line := fmt.Sprint(v)
			if strings.Contains(line, method) && want.MatchString(line) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s %s is served but /api/describe/routes does not say so; add it to the routes in internal/app/app.go", method, path)
		}
	}
}
