package server_test

import (
	"strings"
	"testing"
)

// TestProposalRoutesAreDescribed checks that an agent calling GET
// /api/describe/routes can discover the proposal accept and dismiss endpoints.
// These are functional routes registered in server.go but were missing from the
// describe map, so agents had no way to find them programmatically.
func TestProposalRoutesAreDescribed(t *testing.T) {
	_, h := newApp(t)

	var routes map[string]string
	decode(t, get(t, h, "/api/describe/routes"), &routes)

	if routes["proposal_accept"] == "" {
		t.Error("routes should document proposal_accept")
	} else if !strings.Contains(routes["proposal_accept"], "POST") ||
		!strings.Contains(routes["proposal_accept"], "/proposal/{id}/accept") {
		t.Errorf("proposal_accept description should mention POST /proposal/{id}/accept: %q", routes["proposal_accept"])
	}

	if routes["proposal_dismiss"] == "" {
		t.Error("routes should document proposal_dismiss")
	} else if !strings.Contains(routes["proposal_dismiss"], "POST") ||
		!strings.Contains(routes["proposal_dismiss"], "/proposal/{id}/dismiss") {
		t.Errorf("proposal_dismiss description should mention POST /proposal/{id}/dismiss: %q", routes["proposal_dismiss"])
	}
}
