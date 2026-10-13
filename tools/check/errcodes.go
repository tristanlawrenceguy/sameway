package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// Every error code the API sends is one describe names, with what it
// means: an agent reads GET /api/describe/routes to know what it may get
// back, and a code it was never told of is one it cannot act on. It used
// to be a test in internal/server that read only that folder, so the codes
// sent from api, exchange and web were never looked at.

// describePath is the file whose "errors" entry says every code.
const describePath = "internal/app/app.go"

// checkErrorCodes fails an APIError Code under internal that describe's
// errors do not name.
func checkErrorCodes(root string) report {
	var r report
	said, err := describedErrors(path.Join(root, describePath))
	if err != nil {
		r.problems = append(r.problems, err.Error())
		return r
	}
	sent := sentCodes(root)
	for _, code := range sortedKeys(sent) {
		if !regexp.MustCompile(`\b` + code + `\b`).MatchString(said) {
			r.problems = append(r.problems, fmt.Sprintf("%s sends the error code %s, which the \"errors\" entry in %s does not name; say what it means there", sent[code], code, describePath))
		}
	}
	return r
}

// describedErrors is the text of describe's "errors" entry.
func describedErrors(file string) (string, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %v", describePath, err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), file, src, parser.SkipObjectResolution)
	if err != nil {
		return "", fmt.Errorf("cannot parse %s: %v", describePath, err)
	}
	said := ""
	ast.Inspect(f, func(n ast.Node) bool {
		if kv, ok := n.(*ast.KeyValueExpr); ok && stringLit(kv.Key) == "errors" {
			said = stringLit(kv.Value)
		}
		return said == ""
	})
	if said == "" {
		return "", fmt.Errorf("%s has no \"errors\" entry saying the API's error codes", describePath)
	}
	return said, nil
}

// sentCodes is every Code an APIError literal under internal is given,
// with the first file that sends it.
func sentCodes(root string) map[string]string {
	sent := map[string]string{}
	walkSources(path.Join(root, "internal"), func(rel, full string) {
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return
		}
		src, err := os.ReadFile(full)
		if err != nil {
			return
		}
		f, err := parser.ParseFile(token.NewFileSet(), rel, src, parser.SkipObjectResolution)
		if err != nil {
			return
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isAPIError(lit.Type) {
				return true
			}
			for _, el := range lit.Elts {
				if kv, ok := el.(*ast.KeyValueExpr); ok {
					if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "Code" && stringLit(kv.Value) != "" {
						if _, seen := sent[stringLit(kv.Value)]; !seen {
							sent[stringLit(kv.Value)] = "internal/" + rel
						}
					}
				}
			}
			return true
		})
	})
	return sent
}

// isAPIError says whether a literal's type is web.APIError or a package's
// own name for it.
func isAPIError(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.Ident:
		return strings.EqualFold(e.Name, "apiError")
	case *ast.SelectorExpr:
		return e.Sel.Name == "APIError"
	}
	return false
}

// stringLit is a string literal's value, or "".
func stringLit(e ast.Expr) string {
	b, ok := e.(*ast.BasicLit)
	if !ok || b.Kind != token.STRING {
		return ""
	}
	s, _ := strconv.Unquote(b.Value)
	return s
}
