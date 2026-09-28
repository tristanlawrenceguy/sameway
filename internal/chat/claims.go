package chat

import (
	"context"
	"regexp"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// pagePath is a record's page as the model would name it: /t/<type>/<id>.
var pagePath = regexp.MustCompile(`/t/([a-z][a-z0-9_-]*)/([a-z0-9]{8,})`)

// ClaimResult reports whether one URL mentioned in an assistant reply
// points to a real resource. Exists is false when the store has no such
// record, or when the type named in the path is not known to this workspace.
type ClaimResult struct {
	URL    string
	Exists bool
}

// VerifyClaims scans reply for /t/<type>/<id> URLs and checks each one
// against the store. Only types that exist in the schema are checked; a
// made-up type yields Exists: false because the model is wrong about it too.
func VerifyClaims(ctx context.Context, store *store.Store, reply string) ([]ClaimResult, error) {
	var results []ClaimResult
	for _, m := range pagePath.FindAllStringSubmatch(reply, -1) {
		typ, id := m[1], m[2]
		results = append(results, ClaimResult{URL: m[0]})
		if _, ok := store.Types().Get(typ); !ok {
			continue // will be Exists:false by zero value
		}
		if _, err := store.Get(typ, id); err != nil {
			continue // also Exists:false
		}
		results[len(results)-1].Exists = true
	}
	return results, nil
}
