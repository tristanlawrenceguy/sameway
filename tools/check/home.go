package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// A test must never read or write the person's own folders: the config
// folder, where the known list, snapshots and pasted keys are kept, and
// the home folder. A package whose tests can reach code that finds them
// starts its tests with testhome.Run (or servertest.Main, which calls
// it), so they point at a temp folder for the whole run. Without it, a
// test that forgets an override writes into the developer's real ones.

const modulePath = "github.com/tristanlawrenceguy/sameway/"

// homeRead is code that finds the person's own folders.
var homeRead = regexp.MustCompile(`os\.User(Config|Home)Dir\(|Getenv\("(APPDATA|LOCALAPPDATA|USERPROFILE|HOME|XDG_CONFIG_HOME|XDG_DATA_HOME)"\)`)

// homeGuard is a TestMain that moves them.
var homeGuard = regexp.MustCompile(`testhome\.Run\(|servertest\.Main\(`)

// goPackage is what checkTestHome needs of one folder of Go files.
type goPackage struct {
	imports, testImports []string
	reads, tests, guard  bool
}

func checkTestHome(root string) report {
	pkgs := map[string]*goPackage{}
	fset := token.NewFileSet()
	walkSources(root, func(rel, full string) {
		if !strings.HasSuffix(rel, ".go") {
			return
		}
		src, err := os.ReadFile(full)
		if err != nil {
			return
		}
		f, err := parser.ParseFile(fset, full, src, parser.ImportsOnly)
		if err != nil {
			return
		}
		dir := path.Dir(rel)
		p := pkgs[dir]
		if p == nil {
			p = &goPackage{}
			pkgs[dir] = p
		}
		test := strings.HasSuffix(rel, "_test.go")
		for _, im := range f.Imports {
			v, _ := strconv.Unquote(im.Path.Value)
			if dep, ok := strings.CutPrefix(v, modulePath); ok {
				if test {
					p.testImports = append(p.testImports, dep)
				} else {
					p.imports = append(p.imports, dep)
				}
			}
		}
		if test {
			p.tests = true
			p.guard = p.guard || homeGuard.Match(src)
		} else {
			p.reads = p.reads || homeRead.Match(src)
		}
	})
	reaches := map[string]bool{}
	var reach func(dir string) bool
	reach = func(dir string) bool {
		if r, ok := reaches[dir]; ok {
			return r
		}
		reaches[dir] = false // an import cycle does not compile anyway
		p := pkgs[dir]
		r := p != nil && p.reads
		for i := 0; p != nil && !r && i < len(p.imports); i++ {
			r = reach(p.imports[i])
		}
		reaches[dir] = r
		return r
	}
	var r report
	dirs := make([]string, 0, len(pkgs))
	for dir := range pkgs {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		p := pkgs[dir]
		if !p.tests || p.guard {
			continue
		}
		hit := reach(dir)
		for i := 0; !hit && i < len(p.testImports); i++ {
			hit = reach(p.testImports[i])
		}
		if hit {
			r.problems = append(r.problems, fmt.Sprintf("%s: its tests can reach this computer's own folders (os.UserConfigDir, os.UserHomeDir); add a main_test.go with `func TestMain(m *testing.M) { os.Exit(testhome.Run(m)) }`", dir))
		}
	}
	return r
}
