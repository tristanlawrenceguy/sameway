package main

import "testing"

const describeSrc = `package app

var routes = map[string]string{"errors": "code bad_request (400) or not_found (404)"}
`

// A code describe names passes, from any package under internal and
// under either name for the error; one it does not name fails, once.
func TestAnErrorCodeDescribeDoesNotNameFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "internal/app/app.go", describeSrc)
	write(t, root, "internal/server/a.go", "package server\n\nvar a = apiError{Code: \"bad_request\"}\nvar b = apiError{Code: \"gone\"}\n")
	write(t, root, "internal/server/api/b.go", "package api\n\nvar c = web.APIError{Code: \"not_found\"}\nvar d = web.APIError{Code: \"gone\"}\nvar e = web.APIError{Code: \"bad\"}\n")
	write(t, root, "internal/server/a_test.go", "package server\n\nvar f = apiError{Code: \"only_in_a_test\"}\n")
	r := checkErrorCodes(root)
	if len(r.problems) != 2 || !has(r.problems, "sends the error code gone") || !has(r.problems, "error code bad,") {
		t.Errorf("problems = %q, want gone and bad (bad_request does not name bad)", r.problems)
	}
}

func TestNoErrorsEntryFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "internal/app/app.go", "package app\n")
	if r := checkErrorCodes(root); len(r.problems) != 1 || !has(r.problems, "no \"errors\" entry") {
		t.Errorf("problems = %q", r.problems)
	}
}
