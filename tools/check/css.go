package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// A component's colours are tokens (var(--sw-...)), so light, dark and
// forced colours reach every component by the tokens alone; and no
// component takes away the focus outline. It used to be a test in
// internal/render that read the stylesheets through the registry.
var rawColour = regexp.MustCompile(`(?i)#[0-9a-f]{3,8}\b|rgba?\(|hsla?\(`)

// checkComponentCSS fails a raw colour or a removed focus outline in one
// component's style.css.
func checkComponentCSS(name, file string) []string {
	src, err := os.ReadFile(file)
	if err != nil {
		return nil // a missing style.css is said by checkComponents
	}
	var problems []string
	if m := rawColour.FindString(string(src)); m != "" {
		problems = append(problems, fmt.Sprintf("component %s: style.css uses the raw colour %q; use a --sw- token", name, m))
	}
	if strings.Contains(string(src), "outline: none") || strings.Contains(string(src), "outline:none") {
		problems = append(problems, fmt.Sprintf("component %s: style.css removes the focus outline", name))
	}
	return problems
}
