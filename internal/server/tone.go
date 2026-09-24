package server

import (
	"regexp"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// toneWords are what a block's tone means, for a person who cannot see
// its tint.
var toneWords = map[string]string{
	"accent": "Highlighted.", "success": "Done or good.", "warning": "Warning.", "danger": "Important.", "info": "For information.",
}

// langCode is a language code a page can say: de, fr, pt-BR.
var langCode = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]{2,8})?$`)

// langOf is the lang attribute of a record written in another language
// than the workspace's, so a screen reader reads it in that voice (WCAG
// 3.1.2); nothing when it does not say.
func langOf(rec *store.Record) string {
	if l, _ := rec.Fields["language"].(string); langCode.MatchString(l) {
		return ` lang="` + l + `"`
	}
	return ""
}
