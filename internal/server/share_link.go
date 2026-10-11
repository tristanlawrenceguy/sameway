package server

import (
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A link is kept as the page it names, without what was added to follow
// whoever clicked it: the click identifiers Safari's Link Tracking
// Protection takes off, and the campaign tags (utm_) that say which
// newsletter or post it came from. Opened again from a note, the page is
// not told it came from that post, and the same page saved from two
// places is the same link, so it is known as saved before.

// tracking are the query names taken off; utm_ goes by its prefix.
var tracking = map[string]bool{"gclid": true, "gbraid": true, "wbraid": true, "dclid": true, "fbclid": true,
	"msclkid": true, "twclid": true, "igshid": true, "mc_cid": true, "mc_eid": true, "yclid": true,
	"_hsenc": true, "_hsmi": true, "mkt_tok": true}

// keptLink is a shared link checked and without its tracking, or why it
// cannot be kept.
func keptLink(link string) (*url.URL, error) {
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("a link begins http:// or https://, such as https://example.com/recipe")
	}
	if u.RawQuery == "" {
		return u, nil
	}
	var keep []string
	for _, pair := range strings.Split(u.RawQuery, "&") {
		name, _, _ := strings.Cut(pair, "=")
		if n, err := url.QueryUnescape(name); err == nil {
			name = strings.ToLower(n)
		}
		if pair != "" && !tracking[name] && !strings.HasPrefix(name, "utm_") {
			keep = append(keep, pair)
		}
	}
	u.RawQuery = strings.Join(keep, "&")
	u.ForceQuery = false
	return u, nil
}

// savedBefore is the note that already keeps this link, if one does.
func (s *Server) savedBefore(link string) *store.Record {
	recs, err := s.app.Store.List(shareType, store.ListOptions{})
	if err != nil {
		return nil
	}
	in := "](" + link + ")"
	for _, r := range recs {
		body, _ := r.Fields["body"].(string)
		if strings.Contains(body, in) || slices.Contains(strings.Fields(body), link) {
			return r
		}
	}
	return nil
}

// savedOn says the day a note was saved, as a person would.
func (s *Server) savedOn(t time.Time) string {
	now := s.now()
	t = t.In(now.Location())
	if y, m, d := t.Date(); y == now.Year() && m == now.Month() && d == now.Day() {
		return "earlier today"
	}
	if t.Year() == now.Year() {
		return "on " + t.Format("2 January")
	}
	return "on " + t.Format("2 January 2006")
}
