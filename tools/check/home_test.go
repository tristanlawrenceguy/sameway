package main

import (
	"reflect"
	"strings"
	"testing"
)

// A package whose tests reach the person's folders, directly, through
// what it imports or through what only its tests import, fails without
// testhome; one with it, or one that cannot reach them, passes.
func TestTestsThatCanReachTheHomeFolderNeedTesthome(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	im := func(dep string) string { return "import _ \"" + modulePath + dep + "\"\n" }
	write(t, root, "internal/known/k.go", "package known\nimport \"os\"\nfunc p() { os.UserConfigDir() }\n")
	write(t, root, "internal/known/k_test.go", "package known\n")
	write(t, root, "internal/app/a.go", "package app\n"+im("internal/known"))
	write(t, root, "internal/app/a_test.go", "package app\n")
	write(t, root, "internal/page/p.go", "package page\n")
	write(t, root, "internal/page/p_test.go", "package page_test\n"+im("internal/app"))
	write(t, root, "internal/guarded/g.go", "package guarded\n"+im("internal/known"))
	write(t, root, "internal/guarded/main_test.go", "package guarded\nfunc TestMain(m *testing.M) { os.Exit(testhome.Run(m)) }\n")
	write(t, root, "internal/plain/p.go", "package plain\n")
	write(t, root, "internal/plain/p_test.go", "package plain\n")
	var got []string
	for _, p := range checkTestHome(root).problems {
		dir, _, _ := strings.Cut(p, ":")
		got = append(got, dir)
	}
	want := []string{"internal/app", "internal/known", "internal/page"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("failed %q, want %q", got, want)
	}
}
