package server_test

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Machine words never reach a person. The crew found them one page at a
// time, some thirty times: a field's name for its label, a date as a
// machine writes it, an id, the program an agent called with, a failed
// program's exit status, values run together. This reads every page a
// workspace with real-looking records serves, as a person and a screen
// reader meet it (the text, and the names, labels and descriptions read
// out), and fails on any of them, so the next one is found here.
func TestNoMachineWordsReachAPerson(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	seedLikeAPerson(t, a, h) // machine_seed_test.go

	m := machineSeen{ids: map[string]bool{}, fields: map[string]bool{}, choices: map[string]bool{}}
	var paths []string
	for _, name := range a.Types.Names() {
		ty, _ := a.Types.Get(name)
		for _, f := range ty.Fields {
			for _, v := range f.Values {
				if f.Labels[v] != "" && f.Labels[v] != v || strings.Contains(v, "_") {
					m.choices[v] = true
				}
			}
			if strings.Contains(f.Name, "_") {
				m.fields[f.Name] = true
				m.fields[titled(f.Name)] = true
			}
		}
		if !ty.Content() {
			continue
		}
		paths = append(paths, "/t/"+name)
		recs, _ := a.Store.List(name, store.ListOptions{})
		for _, r := range recs {
			m.ids[r.ID] = true
			paths = append(paths, "/t/"+name+"/"+r.ID)
		}
	}
	m.fields["Created At"], m.fields["Updated At"] = true, true
	paths = append(paths, "/", "/activity", "/search?q=paint", "/search?q=draft", "/calendar", "/help", "/workspaces")
	sort.Strings(paths)

	found := map[string][]string{}
	for _, p := range paths {
		res := get(t, h, p)
		if res.Code != http.StatusOK || !strings.Contains(res.Header().Get("Content-Type"), "html") {
			continue
		}
		for _, why := range m.words(res.Body.String()) {
			found[why] = append(found[why], p)
		}
	}
	var lines []string
	for why, where := range found {
		if len(where) > 4 {
			where = append(where[:4], fmt.Sprintf("and %d more", len(where)-4))
		}
		lines = append(lines, why+"\n      on "+strings.Join(where, ", "))
	}
	sort.Strings(lines)
	if len(lines) > 0 {
		t.Errorf("%d machine words reach a person:\n  %s", len(lines), strings.Join(lines, "\n  "))
	}
}

var (
	machineStamp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}|T00:00:00Z|\b\d{4}-\d{2}-\d{2}\b`)
	machineMisc  = regexp.MustCompile(`Go-http-client|exit status \d|<nil>|%!|map\[|\bnull\b|\bundefined\b|\bNaN\b`)
	machineID    = regexp.MustCompile(`\b[a-z0-9]{16}\b`)
	machineWord  = regexp.MustCompile(`\b[a-z]+(?:_[a-z]+)+\b|\b[A-Z][a-z]+ (?:At|Of|For|By)\b`)
)

// machineSeen is what counts as a machine's word in this workspace: its
// records' ids, and its fields' names as a machine writes them.
type machineSeen struct{ ids, fields, choices map[string]bool }

// words is each machine word in what a person meets on a page, with the
// passage it is in.
func (m machineSeen) words(page string) []string {
	var out []string
	for _, text := range peopleText(page) {
		say := func(kind, what string) {
			out = append(out, fmt.Sprintf("%s %q in %q", kind, what, clipText(text, what)))
		}
		if strings.HasPrefix(text, "\x01") { // a whole element's words
			if v := text[1:]; m.choices[v] {
				say("a choice as stored", v)
			}
			continue
		}
		if strings.HasPrefix(text, "\x00") { // two values with nothing between them
			say("values run together", strings.TrimPrefix(text, "\x00"))
			continue
		}
		for _, w := range machineStamp.FindAllString(text, -1) {
			say("a machine date", w)
		}
		for _, w := range machineMisc.FindAllString(text, -1) {
			say("a program's words", w)
		}
		if strings.Contains(text, "(required)") && text != "(required)" && !strings.HasSuffix(text, " (required)") {
			say("a schema's words", "(required)")
		}
		for _, w := range machineID.FindAllString(text, -1) {
			if m.ids[w] {
				say("a record's id", w)
			}
		}
		for _, w := range machineWord.FindAllString(text, -1) {
			if m.fields[w] {
				say("a field's name", w)
			}
		}
	}
	return out
}

// titled is a field's name as a careless label writes it: created_at,
// Created At.
func titled(name string) string {
	parts := strings.Split(name, "_")
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

var inline = map[string]bool{"span": true, "a": true, "time": true, "strong": true, "em": true, "b": true, "i": true, "small": true, "abbr": true, "data": true}

// peopleText is a page's text as a person and a screen reader meet it: each
// run of text, and each name, label, description, title, alt and
// placeholder. Scripts, styles and what is hidden from everyone are left.
// Two inline elements side by side with nothing between them come as one
// entry starting "\x00": a reader says them as one word.
func peopleText(page string) []string {
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return nil
	}
	var out []string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "template", "head":
				return
			}
			for _, at := range n.Attr {
				switch at.Key {
				case "hidden":
					return
				case "aria-label", "title", "alt", "placeholder", "aria-description", "aria-valuetext":
					if v := strings.TrimSpace(at.Val); v != "" {
						out = append(out, v)
					}
				}
			}
			var prev *html.Node
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				// A link is a unit of its own to every reader; two words in one
				// run of text are not.
				if c.Type == html.ElementNode && inline[c.Data] && prev != nil && c.Data != "a" && prev.Data != "a" {
					a, b := textOf(prev), textOf(c)
					if a != "" && b != "" && !endsSpaced(a) && !startsSpaced(b) {
						out = append(out, "\x00"+a+"|"+b)
					}
				}
				switch {
				case c.Type == html.ElementNode && inline[c.Data]:
					prev = c
				case c.Type == html.TextNode && strings.TrimSpace(c.Data) == "" && c.Data != "":
					prev = nil
				default:
					prev = nil
				}
			}
		}
		if n.Type == html.TextNode {
			if v := strings.TrimSpace(n.Data); v != "" {
				out = append(out, v)
				box := n.Parent
				for box != nil && box.Type == html.ElementNode && inline[box.Data] && box.Data != "a" {
					box = box.Parent
				}
				if box != nil && strings.TrimSpace(textOf(box)) == v {
					out = append(out, ""+v) // the whole of what its element says
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for _, at := range n.Attr {
			if at.Key == "aria-hidden" && at.Val == "true" { // not read out
				return
			}
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func endsSpaced(s string) bool {
	return strings.TrimRight(s, " \n\t") != s || strings.HasSuffix(s, ":") || strings.HasSuffix(s, ",") || strings.HasSuffix(s, ".")
}

func startsSpaced(s string) bool {
	return strings.TrimLeft(s, " \n\t") != s || strings.HasPrefix(s, ",") || strings.HasPrefix(s, ".") || strings.HasPrefix(s, ":") || strings.HasPrefix(s, ")")
}

func clipText(text, around string) string {
	i := max(strings.Index(text, around), 0)
	from, to := max(i-40, 0), min(i+len(around)+40, len(text))
	return text[from:to]
}
