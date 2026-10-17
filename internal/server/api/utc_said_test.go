package api_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server/servertest"
)

// A time written in UTC that the person reads as another hour is said in
// their time with the write, with how to write the time meant; one written
// in their own time says nothing of it.
func TestATimeWrittenInUTCIsSaidInThePersonsTime(t *testing.T) {
	t.Parallel()
	_, h := servertest.At(t, time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
	postForm(t, h, "/zone", url.Values{"zone": {"Asia/Tokyo"}}) // nine hours ahead, all year
	res := postJSON(t, h, http.MethodPost, "/api/event", map[string]any{"title": "Cleanup Day", "starts": "2026-10-25T09:00:00Z"})
	if body := res.Body.String(); !strings.Contains(body, "was written in UTC (09:00), which for the person is") || !strings.Contains(body, "6pm") || !strings.Contains(body, "write 2026-10-25T09:00") {
		t.Errorf("the time is said in the person's time: %s", body)
	}
	res = postJSON(t, h, http.MethodPost, "/api/event", map[string]any{"title": "Picnic", "starts": "2026-10-26T09:00"})
	if strings.Contains(res.Body.String(), "written in UTC") {
		t.Errorf("a time in the person's own time says nothing of it: %s", res.Body.String())
	}
}
