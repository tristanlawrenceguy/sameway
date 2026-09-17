package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/search"
)

// searchCmd is the same search as the page, from the command line.
func (c *ctx) searchCmd() error {
	if len(c.args) == 0 {
		return errors.New("usage: sameway search <words>")
	}
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	q := strings.Join(c.args, " ")
	hits := search.Find(a.Store, a.Types, q)
	c.print(map[string]any{"query": q, "count": len(hits), "hits": hits}, func() {
		if len(hits) == 0 {
			fmt.Fprintf(c.Stdout, "nothing has %q in it\n", q)
		}
		for _, h := range hits {
			fmt.Fprintf(c.Stdout, "%-8s %s  %s\n", h.Type, h.Href, h.Title)
			if h.Snippet != "" {
				fmt.Fprintf(c.Stdout, "         %s\n", h.Snippet)
			}
		}
	})
	return nil
}
