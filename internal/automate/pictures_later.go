package automate

import (
	"context"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A photo shared while the model was down, or while it could not see,
// was sorted from its file's name and left unread for good. So, as search
// by meaning reads a record again only when it needs it, the pictures
// with no words yet are read once a model can: the record that links
// them is renamed as on its first sorting, then sorted again, so its tags
// follow, each in Activity to undo.

// picturesLater is how often pictures not read yet are tried again; one
// that came more recently is being sorted as it came.
const picturesLater = 10 * time.Minute

// PicturesJob reads the pictures sorting could not read when they came.
func (s *Service) PicturesJob() runner.Job {
	return runner.Job{Name: "Pictures not read yet, read once a model can see", Every: picturesLater, Wait: true, Run: func(ctx context.Context, _ time.Time) error {
		c, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		s.readLater(c)
		return nil
	}}
}

// readLater reads, newest first, the unread pictures of each record an
// action that sorts what is added would have sorted, and runs it again.
// It stops at the first the model could not read: it is down or cannot
// see, and is tried again next time.
func (s *Service) readLater(ctx context.Context) {
	if s.Pictures == nil || s.Model == nil || s.Model() == nil {
		return
	}
	actions, _ := s.Store.List(records.ActionType, store.ListOptions{})
	for _, a := range actions {
		if a.Fields["when"] != "added" || (a.Fields["kind"] != "classify" && a.Fields["kind"] != "suggest") {
			continue
		}
		t, ok := s.Store.Types().Get(str(a.Fields["what"]))
		if !ok {
			continue
		}
		recs, _ := query.Filter(s.Store, t, records.StringList(a.Fields["only"]), "created_at", 0, s.Now())
		for i := len(recs) - 1; i >= 0; i-- {
			rec := recs[i]
			before := s.unreadPictures(rec)
			if len(before) == 0 || rec.CreatedAt.Before(a.CreatedAt) || rec.CreatedAt.After(s.Now().Add(-picturesLater)) {
				continue
			}
			read := s.readPictures(ctx, t, rec)
			if len(s.unreadPictures(rec)) == len(before) {
				if s.canBeSeen(before) {
					return
				}
				continue // a picture not read here (HEIC), whatever the model
			}
			if r := withRead(Result{}, a, t, rec, read); r.Change != nil {
				r.Change.Via = "because its pictures could be read now"
				records.Record(s.Store, "system", *r.Change)
			}
			job := automationJob{action: a.ID, vars: s.recordVars(t, rec), why: "its pictures could be read now", backlog: true}
			if s.auto == nil {
				s.runAutomation(ctx, job)
				continue
			}
			select {
			case s.auto.jobs <- job:
			default:
			}
		}
	}
}

// unreadPictures are the pictures and scans a record links to with no
// words yet, as readPictures reads them.
func (s *Service) unreadPictures(rec *store.Record) []string {
	var out []string
	for _, id := range linkedFiles(rec) {
		f, err := s.Store.Get(records.FileType, id)
		if err == nil && (f.Fields["kind"] == "image" || f.Fields["kind"] == "pdf") && strings.TrimSpace(str(f.Fields["text"])+str(f.Fields["description"])) == "" {
			out = append(out, id)
		}
	}
	return out
}

// canBeSeen says whether any of the files has a picture to show a model.
func (s *Service) canBeSeen(ids []string) bool {
	for _, id := range ids {
		if len(s.Pictures(id)) > 0 {
			return true
		}
	}
	return false
}
