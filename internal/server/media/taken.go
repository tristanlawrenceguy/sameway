package media

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A photo or video says when it was taken (convert/taken.go), and that is
// kept on its file when it is kept, so photos are sorted and narrowed by
// the day they are of, not the day they were added. The files kept before
// are read for it once.

// keepTaken reads when a kept photo or video was taken, and keeps it.
func (s *Service) keepTaken(id, kind, path string) {
	if kind != "image" && kind != "video" {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	var at time.Time
	zoned, ok := true, false
	if kind == "image" {
		head := make([]byte, convert.TakenHead)
		n, _ := io.ReadFull(f, head)
		at, zoned, ok = convert.PhotoTaken(head[:n])
	} else if st, err := f.Stat(); err == nil {
		at, ok = convert.VideoTaken(f, st.Size())
	}
	if !ok {
		return
	}
	v := at.Format(time.RFC3339)
	if !zoned { // the camera's clock where it was: read in the person's zone
		v = at.Format("2006-01-02 15:04")
	}
	s.fileSays(id, map[string]any{"taken": v})
}

// takenJob reads, once, when the photos and videos kept before were taken.
func (s *Service) takenJob() runner.Job {
	return runner.Job{Name: "When photos already kept were taken, read once", Run: func(context.Context, time.Time) error {
		if s.app.Store.Meta("files:taken") != "" {
			return nil
		}
		if t, ok := s.app.Types.Get(records.FileType); !ok {
			return nil
		} else if _, has := t.Field("taken"); !has {
			return nil
		}
		recs, err := s.app.Store.List(records.FileType, store.ListOptions{})
		if err != nil {
			return err
		}
		for _, rec := range recs {
			stored, _ := rec.Fields["path"].(string)
			if kind, _ := rec.Fields["kind"].(string); stored != "" && rec.Fields["taken"] == nil {
				s.keepTaken(rec.ID, kind, filepath.Join(s.app.Workspace.FilesDir(), stored))
			}
		}
		s.app.Store.SetMeta("files:taken", "read")
		s.Changed()
		return nil
	}}
}
