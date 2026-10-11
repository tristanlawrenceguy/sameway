package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
)

// The owner reads on Help what runs in the background, in words: how
// often, when it last ran and how that went, and when it runs next. An
// agent reads the same from GET /api/jobs; nobody else sees either.
func TestTheOwnerSeesTheBackgroundWork(t *testing.T) {
	t.Parallel()
	f := runner.NewFake(time.Date(2026, 10, 10, 14, 0, 0, 0, time.Local))
	a, h := newAppWith(t, app.Options{Clock: f.Now, After: f.After})
	a.Workspace.Set("ui.clock", "12")
	ran := make(chan bool, 2)
	a.Jobs.Add(
		runner.Job{Name: "Reminders ring", Every: 5 * time.Second, Run: func(context.Context, time.Time) error { ran <- true; return nil }},
		runner.Job{Name: "Daily copy in the cloud folder", Every: time.Hour, Run: func(context.Context, time.Time) error {
			ran <- true
			return errors.New("the folder is not there any more")
		}},
	)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	a.Jobs.Start(ctx)
	<-ran
	<-ran
	f.BlockUntil(2)

	page := get(t, h, "/help").Body.String()
	for _, want := range []string{
		`<h2 id="help-jobs">Work in the background</h2>`,
		"Reminders ring: every 5 seconds. Last ran today at 2pm, and went well. Next today at 2pm.",
		"Daily copy in the cloud folder: every hour. Last ran today at 2pm, and went wrong: the folder is not there any more. Next today at 3pm.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("Help says %q\n%.4000s", want, page)
		}
	}
	editor := records.Visitor{Name: "Hana", Login: "hana@example.com", Access: records.Edit}
	if theirs := as(t, h, editor, http.MethodGet, "/help", "", "").Body.String(); strings.Contains(theirs, "help-jobs") {
		t.Error("only the owner sees the background work")
	}

	var got struct {
		Jobs []struct {
			Name      string  `json:"name"`
			Every     float64 `json:"every_seconds"`
			Runs      int     `json:"runs"`
			LastError string  `json:"last_error"`
			NextRun   string  `json:"next_run"`
		} `json:"jobs"`
	}
	res := get(t, h, "/api/jobs")
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil || len(got.Jobs) != 2 {
		t.Fatalf("GET /api/jobs: %v %s", err, res.Body.String())
	}
	if j := got.Jobs[1]; j.Every != 3600 || j.Runs != 1 || j.LastError != "the folder is not there any more" || j.NextRun == "" {
		t.Errorf("the copy's job as JSON: %+v", j)
	}
	if code := as(t, h, editor, http.MethodGet, "/api/jobs", "", "").Code; code == http.StatusOK {
		t.Error("the background work is the owner's to read")
	}
}
