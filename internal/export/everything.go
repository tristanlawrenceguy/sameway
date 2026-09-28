package export

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/content"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Everything is a whole workspace in one zip, to keep, to move, or to
// take elsewhere: every record as the Markdown the content folder keeps
// (which `sameway import` reads back), every file as it was added, and a
// spreadsheet of each kind of record for anything that opens one. It is
// written as it goes, so a workspace of gigabytes is never held whole.
func Everything(w io.Writer, name string, st *store.Store, types *schema.Set, mirror content.Mirror, filesDir string, titles Titles) error {
	z := zip.NewWriter(w)
	now := time.Now()
	add := func(path string, compress bool) (io.Writer, error) {
		method := zip.Deflate
		if !compress {
			method = zip.Store
		}
		return z.CreateHeader(&zip.FileHeader{Name: path, Method: method, Modified: now})
	}
	readme, _ := add("README.txt", true)
	fmt.Fprintf(readme, `%s, taken out of sameway on %s.

content/       every record as Markdown with its fields at the top, one
               folder per kind. To bring them back, put this folder in a
               workspace and run: sameway import
files/         every file as it was added, named as the workspace keeps
               them; the file records in content/file say which is which.
spreadsheets/  each kind of record as a spreadsheet (CSV), to open in
               anything.
`, name, now.Format("2 January 2006, 15:04"))
	for _, t := range types.Types {
		if t.Internal || t.Hidden {
			continue
		}
		recs, err := st.List(t.Name, store.ListOptions{})
		if err != nil {
			return err
		}
		if len(recs) == 0 {
			continue
		}
		if mirror.Mirrored(t.Name) {
			for _, r := range recs {
				data, err := content.Encode(t, r)
				if err != nil {
					return err
				}
				f, err := add("content/"+t.Name+"/"+r.ID+".md", true)
				if err != nil {
					return err
				}
				f.Write(data)
			}
		}
		f, err := add("spreadsheets/"+t.Name+".csv", true)
		if err != nil {
			return err
		}
		if err := writeCSV(f, t, recs, titles); err != nil {
			return err
		}
		if t.Name == "file" {
			for _, r := range recs {
				stored, _ := r.Fields["path"].(string)
				if stored == "" || strings.ContainsAny(stored, `/\`) {
					continue
				}
				if err := addFile(add, filepath.Join(filesDir, stored), "files/"+stored); err != nil {
					return err
				}
			}
		}
	}
	return z.Close()
}

// addFile copies one kept file in, stored as it is when it is already
// compressed (pictures, recordings, videos, documents).
func addFile(add func(string, bool) (io.Writer, error), path, name string) error {
	src, err := os.Open(path)
	if err != nil {
		return nil // a file record whose original has gone: the record says what it was
	}
	defer src.Close()
	ext := strings.ToLower(filepath.Ext(path))
	compress := strings.Contains(".txt.md.csv.tsv.json.yaml.yml.html.htm.xml.svg.ics.vcf.srt.vtt.log", ext) && ext != ""
	w, err := add(name, compress)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}
