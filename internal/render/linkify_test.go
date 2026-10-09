package render

import (
	"strings"
	"testing"
)

// A reply names the things it links to: a Markdown link reads as its words,
// a bare address to a record reads as the record's title, and only an
// address nothing can name shows where it goes (backlog 0548).
func TestLinkifyNamesThings(t *testing.T) {
	t.Parallel()
	titles := map[string]string{"/t/note/seeds1": "Seeds to buy"}
	name := func(p string) string { return titles[p] }
	cases := []struct{ in, want string }{
		// A Markdown link keeps the words the reply gave it.
		{"Filed as [Seeds to buy](/t/note/seeds1).",
			`Filed as <a class="sw-link" href="/t/note/seeds1">Seeds to buy</a>.`},
		// Even for a record that is gone, whose title nobody can look up.
		{"It was [Old list](/t/note/gone9).",
			`It was <a class="sw-link" href="/t/note/gone9">Old list</a>.`},
		// A bare address to a record reads as its title.
		{"Created your note at /t/note/seeds1.",
			`Created your note at <a class="sw-link" href="/t/note/seeds1">Seeds to buy</a>.`},
		// Words that are only the address again are no name.
		{"See [/t/note/seeds1](/t/note/seeds1)",
			`See <a class="sw-link" href="/t/note/seeds1">Seeds to buy</a>`},
		// A bare address to a record that is gone keeps its address.
		{"See /t/note/gone9", `See <a class="sw-link" href="/t/note/gone9">/t/note/gone9</a>`},
		// A page with a query keeps it, escaped, and the link its words.
		{"[All of it](/t/note/seeds1?show=fields&x=1)",
			`<a class="sw-link" href="/t/note/seeds1?show=fields&amp;x=1">All of it</a>`},
		// External links too, whether the words are a name or the address.
		{"[The guide](https://example.com/guide) and [https://example.com](https://example.com)",
			`<a class="sw-link" href="https://example.com/guide">The guide</a> and <a class="sw-link" href="https://example.com">example.com</a>`},
		// Words are escaped; an unsafe scheme is not a link.
		{"[<b>x</b>](/t/note/seeds1) [bad](javascript:alert(1))",
			`<a class="sw-link" href="/t/note/seeds1">&lt;b&gt;x&lt;/b&gt;</a> [bad](javascript:alert(1))`},
		// Two links in one line.
		{"[A](/t/task/a1) then [B](/t/task/b2)",
			`<a class="sw-link" href="/t/task/a1">A</a> then <a class="sw-link" href="/t/task/b2">B</a>`},
	}
	for _, c := range cases {
		if got := string(linkifyNamed(c.in, name)); got != c.want {
			t.Errorf("linkify(%q)\n got %s\nwant %s", c.in, got, c.want)
		}
	}
	if got := string(linkify("[x](/t/note/a)")); !strings.Contains(got, `>x</a>`) {
		t.Errorf("linkify without names keeps a Markdown link's words: %s", got)
	}
}

// Where a reply is read out rather than drawn, its links are their words.
func TestLinkWords(t *testing.T) {
	t.Parallel()
	name := func(p string) string {
		if p == "/t/note/seeds1" {
			return "Seeds to buy"
		}
		return ""
	}
	in := "Filed [Seeds](/t/note/seeds1), made /t/note/seeds1, lost /t/note/gone9, read https://example.com/guide."
	want := "Filed Seeds, made Seeds to buy, lost /t/note/gone9, read example.com/guide."
	if got := LinkWords(in, name); got != want {
		t.Errorf("LinkWords\n got %q\nwant %q", got, want)
	}
}
