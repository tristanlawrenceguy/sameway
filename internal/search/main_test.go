package search

import (
	"os"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/testhome"
)

func TestMain(m *testing.M) {
	os.Exit(testhome.Run(m))
}
