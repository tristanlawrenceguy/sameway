package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/prose"
)

// editedFields reads an inline edit form. A prop-<name> value is the words
// as written. An html-<name> value is what the rich editor holds: the
// rendered text a person changed in place, turned back into Markdown here,
// with headings brought back to the level the source had, which
// level-<name> gives (the level a # line was rendered at).
func editedFields(form url.Values) (map[string]any, error) {
	out := map[string]any{}
	for key, values := range form {
		if len(values) == 0 {
			continue
		}
		if name, ok := strings.CutPrefix(key, "prop-"); ok {
			out[name] = strings.ReplaceAll(values[0], "\r\n", "\n")
			continue
		}
		if name, ok := strings.CutPrefix(key, "html-"); ok {
			level, _ := strconv.Atoi(form.Get("level-" + name))
			md, err := fromHTML(values[0], level)
			if err != nil {
				return nil, fmt.Errorf("could not read the edited %s: %w", name, err)
			}
			out[name] = md
		}
	}
	return out, nil
}

// fromHTML is the rich editor's HTML as Markdown, with headings at the
// level the source had.
func fromHTML(html string, level int) (string, error) {
	md, err := convert.HTMLToMarkdown(html)
	if err != nil {
		return "", err
	}
	return unshiftHeadings(strings.TrimSpace(md), level) + "\n", nil
}

var headingLine = regexp.MustCompile(`(?m)^(#{1,6}) `)

// unshiftHeadings undoes the outline shift the page made when it rendered
// the source: a # line shown as an h3 comes back as a # line.
func unshiftHeadings(md string, base int) string {
	if base <= 1 {
		return md
	}
	return headingLine.ReplaceAllStringFunc(md, func(m string) string {
		n := len(m) - 1 - (base - 1)
		if n < 1 {
			n = 1
		}
		return strings.Repeat("#", n) + " "
	})
}

// apiProse converts either way between Markdown and the HTML the page
// shows for it, so the editor can switch views without losing anything:
// {"markdown": "...", "level": 3} gives html; {"html": "...", "level": 3}
// gives markdown.
func (s *Server) apiProse(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Markdown *string `json:"markdown"`
		HTML     *string `json:"html"`
		Level    int     `json:"level"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "send JSON with markdown or html, and level"})
		return
	}
	switch {
	case in.HTML != nil:
		md, err := fromHTML(*in.HTML, in.Level)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"markdown": md})
	case in.Markdown != nil:
		writeJSON(w, http.StatusOK, map[string]any{"html": string(prose.Render(*in.Markdown, in.Level))})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "send markdown or html"})
	}
}
