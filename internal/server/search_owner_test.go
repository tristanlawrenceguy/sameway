package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A search finds what the one searching may read, and nothing the system
// keeps for itself: not the titles of the owner's conversations, for
// anyone, and so not for someone let in to look.
func TestASearchDoesNotFindTheOwnersConversations(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Store.Create(records.ConversationType, map[string]any{"title": "Divorce lawyer questions"})
	a.Store.Create("note", map[string]any{"title": "Lawyer for the house"})
	viewer := records.Visitor{Name: "Vi", Login: "vi@example.com", Access: records.View}
	for _, path := range []string{"/search?q=lawyer", "/api/search?q=lawyer"} {
		body := as(t, h, viewer, http.MethodGet, path, "", "").Body.String()
		if strings.Contains(body, "Divorce") {
			t.Errorf("%s shows a viewer the owner's conversation", path)
		}
		if !strings.Contains(body, "for the house") {
			t.Errorf("%s still finds the note", path)
		}
	}
}
