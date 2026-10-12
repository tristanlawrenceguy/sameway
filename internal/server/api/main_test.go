package api_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server/servertest"
)

// TestMain keeps this computer's folders out of the tests (servertest.Main).
func TestMain(m *testing.M) { servertest.Main(m) }

// The API's tests use the whole server, as an agent does, with the helpers
// the server's own tests use, under the same names.
var (
	newApp      = servertest.New
	do          = servertest.Do
	get         = servertest.Get
	postForm    = servertest.PostForm
	postJSON    = servertest.PostJSON
	parse       = servertest.Parse
	decode      = servertest.Decode
	as          = servertest.As
	after       = servertest.After
	wantStatus  = servertest.WantStatus
	truncate    = servertest.Truncate
	said        = servertest.Said
	written     = servertest.Written
	apiAs       = servertest.Headed
	newestEntry = servertest.NewestEntry
)
