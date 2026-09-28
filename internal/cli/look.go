package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// lookCmd prints a page as a screen reader gets it, through the same
// handler the API serves, so a script and an agent read the same outline.
// --scripts reads it in a headless browser with its scripts run.
func (c *ctx) lookCmd() error {
	fs := flag.NewFlagSet("look", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	scripts := fs.Bool("scripts", false, "read the page with its scripts run, in the Chrome, Edge or Chromium on this machine, with the real Tab order")
	only := fs.String("only", "", "keep only these sections: landmarks, headings, controls, live, components, comma separated")
	kind := fs.String("kind", "", "keep only controls of this kind: link, button, textbox, checkbox, ...")
	name := fs.String("name", "", "keep only controls with these words in their name")
	var args []string
	rest := c.args
	for len(rest) > 0 {
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() == 0 {
			break
		}
		args, rest = append(args, fs.Arg(0)), fs.Args()[1:]
	}
	if len(args) != 1 {
		return errors.New("usage: sameway look <path> [--scripts] [--only controls] [--kind button] [--name words]   for example: sameway look /t/note --scripts")
	}
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	q := url.Values{"path": {args[0]}}
	if *scripts {
		q.Set("scripts", "1")
	}
	for k, v := range map[string]string{"only": *only, "kind": *kind, "name": *name} {
		if v != "" {
			q.Set(k, v)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/look?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	server.New(a).ServeHTTP(rec, req)
	if rec.Code >= 400 {
		return errors.New(rec.Body.String())
	}
	var out json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		return err
	}
	enc := json.NewEncoder(c.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
