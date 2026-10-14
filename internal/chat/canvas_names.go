package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// A tab's name is a word or two in a row of them. A model asked for a tab
// "for everything about the house move in March" could call it that, and
// the tab bar cut it short; nothing told the model, which went on calling
// the tab by a name the person never saw. Making or renaming a tab with a
// long name says how the bar shows it.

// tabWords is how many words a tab's name reads well in.
const tabWords = 3

// longTabSaid is " That is long for a tab: …", or "" for a name of a word
// or two.
func longTabSaid(name string) string {
	shown := trim.Title(name)
	if len(strings.Fields(name)) <= tabWords && shown == name {
		return ""
	}
	if shown != name {
		return fmt.Sprintf(" That is long for a tab: the tab bar shows it as %q. A word or two, in the person's words, reads better; rename_canvas changes it.", shown)
	}
	return " That is long for a tab, beside the others in the bar. A word or two, in the person's words, reads better; rename_canvas changes it."
}
