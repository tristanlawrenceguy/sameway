package records

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A change made from a copy that has since gone out of date is the change
// that loses someone's work: a person with a page open for an hour saves
// over what an agent did meanwhile, or an agent saves what it read before
// the person edited. Every edit can say which version of the record it
// started from, the record's updated_at, and each way in answers the same
// question with it: has this changed since you read it?

// Version is the record's version: when it was last written.
func Version(rec *store.Record) string { return rec.UpdatedAt.UTC().Format(time.RFC3339Nano) }

// SameVersion says whether version names the record as it is now. Quotes
// are allowed, as an HTTP If-Match header has them.
func SameVersion(rec *store.Record, version string) bool {
	at, err := time.Parse(time.RFC3339Nano, strings.Trim(strings.TrimSpace(version), `"`))
	return err == nil && at.Equal(rec.UpdatedAt)
}

// Print is a short fingerprint of one field's value, the same for the same
// value however it came, so a page can say what each field held when it
// was opened without carrying the words twice.
func Print(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha1.Sum(raw)
	return hex.EncodeToString(sum[:6])
}

// Prints is the fingerprint of each of a record's fields.
func Prints(fields map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range fields {
		out[k] = Print(v)
	}
	return out
}
