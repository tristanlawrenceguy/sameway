package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ReadBody is a request's JSON object of fields, at most 4 MB; an empty
// body is no fields.
func ReadBody(r *http.Request) (map[string]any, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var fields map[string]any
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, errors.New("body must be a JSON object of fields: " + JSONTrouble(err))
	}
	return fields, nil
}

// Attachment says a response is a file to keep, by its name: filename for
// every browser, and filename* for a name beyond ASCII (RFC 6266).
func Attachment(w http.ResponseWriter, name string) {
	plain := strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' || r == '/' {
			return '_'
		}
		return r
	}, name)
	v := fmt.Sprintf(`attachment; filename="%s"`, plain)
	if plain != name {
		v += "; filename*=UTF-8''" + url.PathEscape(strings.ReplaceAll(name, "/", "_"))
	}
	w.Header().Set("Content-Disposition", v)
}
