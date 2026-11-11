package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
)

// Photos taken on a phone come into Sameway by themselves, through the
// app that already copies them to this computer: iCloud Photos, OneDrive
// or Google Drive puts each in a folder here, and Sameway keeps the new
// ones from that folder as files, sorted like anything shared. Only those
// that arrive after it is turned on: a library of years is not read photo
// by photo by the assistant unasked. A photo still being copied (changed
// in the last half minute) waits for the next look.

// PhotosFrom is the folder new photos are kept from, empty when none.
func (s *Service) PhotosFrom() string { return s.app.Store.Meta("files:from") }

// KeepPhotosFrom starts keeping new photos from a folder: those there now
// are passed over. An empty folder stops it.
func (s *Service) KeepPhotosFrom(dir string) error {
	if dir == "" {
		s.app.Store.SetMeta("files:from", "")
		return nil
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return errNoFolder
	}
	walkPhotos(dir, func(key, _ string) { s.app.Store.SetMeta(key, "there") })
	s.app.Store.SetMeta("files:from", dir)
	return nil
}

type folderError string

func (e folderError) Error() string { return string(e) }

const errNoFolder = folderError("there is no folder there on this computer")

// photosJob looks in the folder each minute for photos come since.
func (s *Service) photosJob() runner.Job {
	return runner.Job{Name: "New photos kept from the phone's folder", Every: time.Minute, Wait: true, Run: func(_ context.Context, now time.Time) error {
		s.KeepNewPhotos(now)
		return nil
	}}
}

// KeepNewPhotos keeps each photo or video come into the folder since it
// was last looked in, and says how many.
func (s *Service) KeepNewPhotos(now time.Time) int {
	dir := s.PhotosFrom()
	if dir == "" {
		return 0
	}
	kept := 0
	walkPhotos(dir, func(key, path string) {
		if s.app.Store.Meta(key) != "" {
			return
		}
		if fi, err := os.Stat(path); err != nil || now.Sub(fi.ModTime()) < 30*time.Second {
			return // still being copied
		}
		f, err := os.Open(path)
		if err != nil {
			return
		}
		name := filepath.Base(path)
		rec, stored, err := s.KeepFile(records.Who{Actor: "human", Via: "photos folder"}, f, name, "", "")
		f.Close()
		s.app.Store.SetMeta(key, "kept")
		if err != nil {
			return
		}
		s.ReadKept(rec.ID, name, stored, false)
		kept++
	})
	if kept > 0 {
		s.Changed()
	}
	return kept
}

// walkPhotos calls each with every photo and video in a folder and the
// folders in it, by a key that is the same while the file is: its place
// in the folder and its size. Hidden files and folders are passed over.
func walkPhotos(dir string, each func(key, path string)) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") && path != dir {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if k := convert.Kind(d.Name()); k != "image" && k != "video" {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		h := sha256.Sum256([]byte(filepath.ToSlash(rel) + "|" + strconv.FormatInt(fi.Size(), 10)))
		each("files:from:"+hex.EncodeToString(h[:12]), path)
		return nil
	})
}

// PhotoFolders are the folders on this computer a phone's photos are
// copied to by an app found here, each with the app's name.
func PhotoFolders() [][2]string {
	home, _ := os.UserHomeDir()
	var out [][2]string
	try := func(name string, parts ...string) {
		if parts[0] == "" {
			return
		}
		dir := filepath.Join(parts...)
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			out = append(out, [2]string{name, dir})
		}
	}
	if home != "" {
		try("iCloud Photos", home, "Pictures", "iCloud Photos", "Photos")
	}
	for _, od := range []string{os.Getenv("OneDriveConsumer"), os.Getenv("OneDrive")} {
		try("OneDrive", od, "Pictures", "Camera Roll")
		if len(out) > 0 && out[len(out)-1][0] == "OneDrive" {
			break
		}
	}
	return out
}
