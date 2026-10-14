package fold

import (
	"sync"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Compare orders two strings as a person reading a list expects: by the
// Unicode Collation Algorithm's default order (UTS #10, CLDR root), with
// numbers by their value. So Émile sorts with Emile and before Zoë, not
// after every plain letter; ébène is among the e's; case only breaks a
// tie; and Item 2 comes before Item 10. It is -1, 0 or +1.
func Compare(a, b string) int {
	c := collators.Get().(*collate.Collator)
	defer collators.Put(c)
	return c.CompareString(a, b)
}

// Less is Compare for a sort.
func Less(a, b string) bool { return Compare(a, b) < 0 }

// A collator keeps buffers, so each goroutine takes its own.
var collators = sync.Pool{New: func() any { return collate.New(language.Und, collate.Numeric) }}
