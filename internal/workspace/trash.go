package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Deleting a workspace used to remove its folder for good, with everything
// in it: the database, the content, the files, any git history, and
// whatever else the person kept in that folder. Now it goes to Sameway's
// trash, beside the list of known workspaces, and can be put back. The
// owner's rule: everything reversible.

// Trashed is a workspace in the trash: where it was, where it is now, and
// when it went.
type Trashed struct {
	Name string    `json:"name"`
	From string    `json:"from"`
	Now  string    `json:"now"`
	At   time.Time `json:"at"`
}

// TrashDir is where deleted workspaces go, beside the known list.
func TrashDir() string { return filepath.Join(filepath.Dir(KnownPath()), "trash") }

func trashIndex() string { return filepath.Join(TrashDir(), "trash.json") }

// Trash moves a workspace folder to the trash and says where it went.
// When the trash is on another drive, it goes into a hidden folder beside
// where it was instead, which a rename can always reach.
func Trash(dir, name string) (Trashed, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Trashed{}, err
	}
	stamp := time.Now().Format("20060102-150405")
	base := filepath.Base(abs) + "-" + stamp
	places := []string{filepath.Join(TrashDir(), base), filepath.Join(filepath.Dir(abs), ".sameway-trash", base)}
	var last error
	for _, to := range places {
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			last = err
			continue
		}
		// A file the server just let go of can stay locked a moment on
		// Windows: try a few times before trying the next place.
		for try := 0; try < 10; try++ {
			if last = os.Rename(abs, to); last == nil {
				t := Trashed{Name: name, From: abs, Now: to, At: time.Now()}
				list := TrashedWorkspaces()
				list = append(list, t)
				return t, writeTrash(list)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	return Trashed{}, fmt.Errorf("could not move the workspace to the trash, so nothing was deleted: %w", last)
}

// TrashedWorkspaces lists what is in the trash, newest first, leaving out
// anything that is no longer where the trash put it.
func TrashedWorkspaces() []Trashed {
	raw, err := os.ReadFile(trashIndex())
	if err != nil {
		return nil
	}
	var list []Trashed
	json.Unmarshal(raw, &list)
	var out []Trashed
	for _, t := range list {
		if _, err := os.Stat(t.Now); err == nil {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out
}

// Untrash puts a workspace back where it was, and among the known ones.
func Untrash(now string) (Trashed, error) {
	list := TrashedWorkspaces()
	for i, t := range list {
		if t.Now != now {
			continue
		}
		if _, err := os.Stat(t.From); err == nil {
			return t, errors.New("there is already a folder at " + t.From + "; move it away first")
		}
		if err := os.MkdirAll(filepath.Dir(t.From), 0o755); err != nil {
			return t, err
		}
		if err := os.Rename(t.Now, t.From); err != nil {
			return t, err
		}
		Remember(t.From, "")
		return t, writeTrash(append(list[:i], list[i+1:]...))
	}
	return Trashed{}, errors.New("that workspace is not in the trash")
}

func writeTrash(list []Trashed) error {
	if err := os.MkdirAll(TrashDir(), 0o755); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(trashIndex(), raw, 0o644)
}
