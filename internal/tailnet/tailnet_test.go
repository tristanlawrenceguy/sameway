package tailnet_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/tailnet"
)

// A workspace that names no machine stays off the tailnet: nothing starts
// and nothing is said.
func TestNoNameMeansOff(t *testing.T) {
	said := []string{}
	err := tailnet.Start(context.Background(), tailnet.Config{Name: "  "}, http.NotFoundHandler(),
		func(s string) { said = append(said, s) })
	if err != nil {
		t.Fatal(err)
	}
	if len(said) != 0 {
		t.Errorf("an unnamed tailnet should say nothing, said %q", said)
	}
}
