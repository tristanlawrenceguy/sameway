// The component registry: the built-in components and arrangements, the
// workspace's own, and the base stylesheets and scripts every page carries.

package app

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// NewRegistry loads the built-in components and then the workspace ones.
func NewRegistry(workspaceComponents string) (*render.Registry, error) {
	reg := render.New()
	tokensCSS, err := fs.ReadFile(design.FS, "tokens/tokens.css")
	if err != nil {
		return nil, fmt.Errorf("design tokens missing; run `go run ./tools/tokens`: %w", err)
	}
	baseCSS, err := baseStyles()
	if err != nil {
		return nil, err
	}
	reg.SetBase(string(tokensCSS), baseCSS)
	baseJS, err := baseScripts()
	if err != nil {
		return nil, err
	}
	reg.SetBaseJS(baseJS)
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		return nil, err
	}
	if err := reg.LoadArrangementsFS(design.FS, "arrangements", "builtin"); err != nil {
		return nil, err
	}
	if workspaceComponents != "" {
		if err := reg.LoadDir(workspaceComponents, "workspace"); err != nil {
			return nil, err
		}
		// A workspace may keep arrangements of its own beside its components.
		if err := reg.LoadArrangementsDir(filepath.Join(filepath.Dir(workspaceComponents), "arrangements"), "workspace"); err != nil {
			return nil, err
		}
	}
	return reg, nil
}

// baseStyles concatenates every stylesheet in design/base in filename
// order. The files are numbered so the order is deterministic and each one
// covers a single concern.
func baseStyles() (string, error) {
	return concatBase(".css")
}

func concatBase(ext string) (string, error) {
	entries, err := fs.ReadDir(design.FS, "base")
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ext) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		src, err := fs.ReadFile(design.FS, "base/"+n)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "\n/* base: %s */\n%s", n, src)
	}
	return b.String(), nil
}

// baseScripts concatenates every script in design/base, the same way its
// stylesheets are concatenated.
func baseScripts() (string, error) {
	return concatBase(".js")
}
