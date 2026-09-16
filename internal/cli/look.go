package cli

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// lookCmd prints a page as a screen reader gets it, through the same
// handler the API serves, so a script and an agent read the same outline.
func (c *ctx) lookCmd() error {
	if len(c.args) != 1 {
		return errors.New("usage: sameway look <path>   for example: sameway look /t/note")
	}
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	req := httptest.NewRequest(http.MethodGet, "/api/look?path="+c.args[0], nil)
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
