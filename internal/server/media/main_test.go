package media_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server/servertest"
)

// TestMain keeps this computer's folders out of the tests (servertest.Main).
func TestMain(m *testing.M) { servertest.Main(m) }

// Media's tests use the whole server, as a person does, with the helpers
// the server's own tests use, under the same names.
var (
	newApp        = servertest.New
	newAppWith    = servertest.NewWith
	do            = servertest.Do
	get           = servertest.Get
	postForm      = servertest.PostForm
	as            = servertest.As
	after         = servertest.After
	landed        = servertest.Landed
	multipartFile = servertest.MultipartFile
	wantStatus    = servertest.WantStatus
	truncate      = servertest.Truncate
	waitFor       = servertest.WaitFor
	said          = servertest.Said
)
