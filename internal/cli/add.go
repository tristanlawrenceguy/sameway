package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// addCmd adds files already on this computer to the workspace: copied in
// as they are, a piece at a time, so a video of gigabytes is added with no
// browser and no upload, and read as an upload would be.
func (c *ctx) addCmd() error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	title := fs.String("title", "", "what to call it, when adding one file; its name otherwise")
	var paths []string
	rest := c.args
	for len(rest) > 0 {
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() == 0 {
			break
		}
		paths, rest = append(paths, fs.Arg(0)), fs.Args()[1:]
	}
	if len(paths) == 0 {
		return errors.New("usage: sameway add <file> [more files] [--title words]   for example: sameway add ~/Videos/talk.mp4")
	}
	if *title != "" && len(paths) > 1 {
		return errors.New("--title names one file; add the files one at a time to name each")
	}
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	srv := server.New(a)
	var added []any
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return fmt.Errorf("could not open %s: %w", p, err)
		}
		if st, err := f.Stat(); err == nil && st.IsDir() {
			f.Close()
			return fmt.Errorf("%s is a folder; name the files in it", p)
		}
		rec, err := srv.AddFile(context.Background(), f, filepath.Base(p), *title)
		f.Close()
		if err != nil {
			return fmt.Errorf("could not add %s: %w", p, err)
		}
		added = append(added, rec)
		if !c.JSON {
			t, _ := rec.Fields["title"].(string)
			note, _ := rec.Fields["note"].(string)
			fmt.Fprintf(c.Stdout, "added %s as %s (/t/file/%s)", filepath.Base(p), t, rec.ID)
			if note != "" {
				fmt.Fprintf(c.Stdout, ": %s", note)
			}
			fmt.Fprintln(c.Stdout)
		}
	}
	if c.JSON {
		return json.NewEncoder(c.Stdout).Encode(added)
	}
	return nil
}
