package chat

import "regexp"

// pagePath is a record's page as the model would name it: /t/<type>/<id>.
var pagePath = regexp.MustCompile(`/t/([a-z][a-z0-9_-]*)/([a-z0-9]{8,})`)

// claimedMissing returns the first page a reply names that does not exist,
// or "" when every page it names is real. A model that says "it's at
// /t/note/abc" without having called create_record has made nothing; the
// store knows, and the reply need not be taken at its word. Only types the
// workspace has count: a made-up type is a slip of another kind.
func (s *Service) claimedMissing(reply string) string {
	for _, m := range pagePath.FindAllStringSubmatch(reply, -1) {
		if _, ok := s.Store.Types().Get(m[1]); !ok {
			continue
		}
		if _, err := s.Store.Get(m[1], m[2]); err != nil {
			return m[0]
		}
	}
	return ""
}
