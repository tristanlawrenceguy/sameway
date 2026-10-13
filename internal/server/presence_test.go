package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

var hana = records.Visitor{Name: "Hana", Login: "hana@example.com", Access: records.Edit}

func presenceLine(page string) string {
	i := strings.Index(page, `data-component="presence"`)
	if i < 0 {
		return ""
	}
	line := page[i:]
	return line[:strings.Index(line, "</p>")]
}

// Someone on the page the reader is on is "on this page", marked so the
// page can say so once when they arrive; elsewhere, where they are.
func TestSomeoneOnThisPageIsSaidToBe(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Records.Owner = records.Visitor{Access: records.Owner, Login: "me@example.com", Name: "Me"}
	as(t, h, hana, http.MethodGet, "/t/note", "", "")
	line := presenceLine(get(t, h, "/t/note").Body.String())
	if !strings.Contains(line, "Hana</bdi></span>, on this page") || !strings.Contains(line, "data-here") {
		t.Errorf("on the same page, she is on this page: %s", line)
	}
	line = presenceLine(get(t, h, "/").Body.String())
	if !strings.Contains(line, "on Notes") || strings.Contains(line, "data-here") {
		t.Errorf("elsewhere, she is on Notes: %s", line)
	}
}

// Where the owner is on a part of the workspace that is theirs alone is
// not said: its title is not the others' to read.
func TestWhereTheOwnerIsAloneIsNotSaid(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Records.Owner = records.Visitor{Access: records.Owner, Login: "me@example.com", Name: "Me"}
	c, err := a.Store.Create(records.ConversationType, map[string]any{"title": "A surprise for Hana"})
	if err != nil {
		t.Fatal(err)
	}
	get(t, h, "/t/conversation/"+c.ID)
	line := presenceLine(as(t, h, hana, http.MethodGet, "/", "", "").Body.String())
	if !strings.Contains(line, "Me</bdi></span>") {
		t.Errorf("Hana sees the owner is here: %s", line)
	}
	if strings.Contains(line, "surprise") || strings.Contains(line, ", on") {
		t.Errorf("but not where: %s", line)
	}
}

// A published page does not carry who is in the workspace, even hidden.
func TestAPublishedPageSaysNothingOfWhoIsHere(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Records.Owner = records.Visitor{Access: records.Owner, Login: "me@example.com", Name: "Me"}
	n, _ := a.Store.Create("note", map[string]any{"title": "Sourdough"})
	a.Workspace.Config.Publish.Types = "note"
	as(t, h, hana, http.MethodGet, "/t/note", "", "")
	page := public(t, h.(*server.Server).Public(nil), http.MethodGet, "/t/note/"+n.ID, "").Body.String()
	if !strings.Contains(page, "Sourdough") || strings.Contains(page, "Hana") || strings.Contains(page, "sw-present") {
		t.Errorf("the internet is not told who is here:\n%s", truncate(page))
	}
}

// A page that says its person has been idle keeps following but no longer
// makes them here; nor does a page fetching itself to follow a change.
func TestIdleIsNotHere(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Records.Owner = records.Visitor{Access: records.Owner, Login: "me@example.com", Name: "Me"}
	follow := func(path string) {
		ctx, cancel := context.WithTimeout(records.WithVisitor(context.Background(), hana), 1500*time.Millisecond)
		defer cancel()
		req := httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctx)
		req.Header.Set("Referer", "http://example.com/t/note")
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
	follow("/events?idle=1")
	req := httptest.NewRequest(http.MethodGet, "/t/note", nil)
	req.Header.Set("X-Requested-With", "sameway-live")
	h.ServeHTTP(httptest.NewRecorder(), req.WithContext(records.WithVisitor(req.Context(), hana)))
	if line := presenceLine(get(t, h, "/").Body.String()); line != "" {
		t.Errorf("an idle page and a live fetch do not make her here: %s", line)
	}
	follow("/events")
	if line := presenceLine(get(t, h, "/").Body.String()); !strings.Contains(line, "Hana") {
		t.Errorf("a page she is using does: %s", line)
	}
}

// An arrival on this page said once, politely, and leaving never:
// behave-components.mjs. A page untouched for ten minutes: behave-connect.mjs.
