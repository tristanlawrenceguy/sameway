package server

import (
	"crypto/rand"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"
)

// A phone pairs by scanning the code on Workspaces, but not everyone can
// scan: a camera that will not focus, a phone held by someone who cannot
// see the screen well, a phone without a camera app that reads codes. So
// the code is also eight letters a person can type on the phone, the way
// WhatsApp links a computer by a phone number and a code of eight
// characters. The letters are from RFC 8628's set for codes typed by hand
// (no vowels, so no words; no digits, so no 0 and O), shown in two groups
// of four, and read however they are typed: small letters, spaces and the
// dash are all the same. Eight of twenty letters is enough with a limit on
// guesses: after five wrong tries every code waiting is dropped, and the
// owner reloads Workspaces for a new one.

// pairLetters are the letters a pairing code is made of.
const pairLetters = "BCDFGHJKLMNPQRSTVWXZ"

// pairTries is how many wrong codes are let through before every code
// waiting is dropped.
const pairTries = 5

// pairing is the codes waiting to pair a phone, each until it is used or
// ten minutes old, and the wrong tries since the last were dropped.
type pairing struct {
	mu    sync.Mutex
	codes map[string]time.Time
	wrong int
}

// fresh is a new code, working once for ten minutes, as it is shown:
// two groups of four letters.
func (p *pairing) fresh(now time.Time) string {
	b := make([]byte, 0, 8)
	for len(b) < 8 {
		var c [1]byte
		rand.Read(c[:])
		if int(c[0]) < 256/len(pairLetters)*len(pairLetters) { // no letter more likely than another
			b = append(b, pairLetters[int(c[0])%len(pairLetters)])
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.codes == nil {
		p.codes = map[string]time.Time{}
	}
	for k, until := range p.codes {
		if now.After(until) {
			delete(p.codes, k)
		}
	}
	p.codes[string(b)] = now.Add(lanPairFor)
	return string(b[:4]) + "-" + string(b[4:])
}

// use says whether a code, as typed or scanned, pairs a phone, and spends
// it when it does.
func (p *pairing) use(typed string, now time.Time) bool {
	code := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		if r >= 'A' && r <= 'Z' {
			return r
		}
		return -1
	}, typed)
	if code == "" {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	until, ok := p.codes[code]
	if ok && !now.After(until) {
		delete(p.codes, code)
		return true
	}
	if p.wrong++; p.wrong >= pairTries {
		p.codes, p.wrong = nil, 0
	}
	return false
}

// lanPairPage is the page a device not let in on the Wi-Fi sees: what to
// do, and a box to type the code's letters in.
func lanPairPage(w http.ResponseWriter, words string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`<!doctype html><html lang="en-GB"><meta name="viewport" content="width=device-width"><title>Pair with Sameway</title>` +
		`<main style="font:1.1rem/1.5 system-ui;max-width:30rem;margin:2rem auto;padding:0 1rem"><p>` + template.HTMLEscapeString(words) + `</p>` +
		`<form action="/pair" method="get"><p><label for="code">The eight letters Sameway shows</label><br>` +
		`<input id="code" name="code" required autocomplete="one-time-code" autocapitalize="characters" spellcheck="false" style="font:inherit;min-height:44px;width:12ch;letter-spacing:.1em"></p>` +
		`<p><button style="font:inherit;min-height:44px;padding:0 1rem">Pair this phone</button></p></form></main>`))
}
