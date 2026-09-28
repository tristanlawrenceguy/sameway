package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestCancelOnARecordJustAddedTakesItBack: Add a person, then Cancel before
// saving, and there is no new person; once saved, Cancel leaves it alone.
func TestCancelOnARecordJustAddedTakesItBack(t *testing.T) {
	a, h := newApp(t)
	added := postForm(t, h, "/t/person/add", url.Values{})
	where := added.Header().Get("Location")
	if !strings.Contains(where, "?added=") || !strings.HasSuffix(where, "#edit") {
		t.Fatalf("Add should open the new record's editor naming the adding, got %q", where)
	}
	page := get(t, h, strings.TrimSuffix(where, "#edit")).Body.String()
	i := strings.Index(page, `data-discard="`)
	if i < 0 {
		t.Fatalf("a record just added should say Cancel takes it back")
	}
	discard := page[i+len(`data-discard="`):]
	discard = strings.ReplaceAll(discard[:strings.Index(discard, `"`)], "&amp;", "&")
	res := postForm(t, h, discard, url.Values{})
	if res.Code != http.StatusSeeOther || res.Header().Get("Location") != "/t/person" {
		t.Errorf("Cancel should go back to the list, got %d %q", res.Code, res.Header().Get("Location"))
	}
	if recs, _ := a.Store.List("person", store.ListOptions{}); len(recs) != 0 {
		t.Errorf("Cancel should leave no new person, found %d", len(recs))
	}

	// Saved once, it is the person's, and Cancel does not take it away.
	added = postForm(t, h, "/t/person/add", url.Values{})
	where = strings.TrimSuffix(added.Header().Get("Location"), "#edit")
	id := strings.TrimPrefix(where[:strings.Index(where, "?")], "/t/person/")
	postForm(t, h, "/t/person/"+id+"/props", url.Values{"prop-name": {"Sandra"}})
	if page := get(t, h, where).Body.String(); strings.Contains(page, "data-discard") {
		t.Errorf("a record saved since it was added should not offer to be taken back")
	}
}
