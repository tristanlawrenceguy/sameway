package server

import (
	"regexp"
	"strings"
)

// What differs between the page's version of a text and the other one,
// word by word, so a person need not read two long texts side by side to
// find the one sentence that changed. Each part is text both have, text
// only the page's has, or text only the other has, in reading order.

var clashTokens = regexp.MustCompile(`\s+|\S+`)

// clashMost is how many word pairs are compared at most; past that, what
// differs after the shared start and end is shown as one change.
const clashMost = 250000

// clashParts is what differs, for the clash component's diff prop, or
// nothing when the two are the same.
func clashParts(page, other string) []any {
	if page == other {
		return nil
	}
	a, b := clashTokens.FindAllString(page, -1), clashTokens.FindAllString(other, -1)
	start := 0
	for start < len(a) && start < len(b) && a[start] == b[start] {
		start++
	}
	end := 0
	for end < len(a)-start && end < len(b)-start && a[len(a)-1-end] == b[len(b)-1-end] {
		end++
	}
	var ops [][2]string
	for _, t := range a[:start] {
		ops = append(ops, [2]string{"both", t})
	}
	ops = append(ops, clashMiddle(a[start:len(a)-end], b[start:len(b)-end])...)
	for _, t := range a[len(a)-end:] {
		ops = append(ops, [2]string{"both", t})
	}
	return clashRuns(ops)
}

// clashMiddle compares the words that differ by their longest run in
// common, the way a diff does.
func clashMiddle(a, b []string) [][2]string {
	var ops [][2]string
	if len(a)*len(b) > clashMost {
		for _, t := range a {
			ops = append(ops, [2]string{"page", t})
		}
		for _, t := range b {
			ops = append(ops, [2]string{"this", t})
		}
		return ops
	}
	n, m := len(a), len(b)
	common := make([][]int32, n+1)
	for i := range common {
		common[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				common[i][j] = common[i+1][j+1] + 1
			} else {
				common[i][j] = max(common[i+1][j], common[i][j+1])
			}
		}
	}
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && a[i] == b[j]:
			ops = append(ops, [2]string{"both", a[i]})
			i, j = i+1, j+1
		case j == m || (i < n && common[i+1][j] >= common[i][j+1]):
			ops = append(ops, [2]string{"page", a[i]})
			i++
		default:
			ops = append(ops, [2]string{"this", b[j]})
			j++
		}
	}
	return ops
}

// clashRuns joins the words into parts: a change is one run, the page's
// words then the other's, and a space between two changed words belongs
// to the change rather than splitting it in two.
func clashRuns(ops [][2]string) []any {
	var parts []any
	var same, page, this strings.Builder
	emit := func(in string, b *strings.Builder) {
		if strings.TrimSpace(b.String()) != "" || (in == "both" && b.Len() > 0) {
			parts = append(parts, map[string]any{"in": in, "text": b.String()})
		}
		b.Reset()
	}
	for k, op := range ops {
		changing := page.Len() > 0 || this.Len() > 0
		next := k+1 < len(ops) && ops[k+1][0] != "both"
		switch {
		case op[0] == "page":
			emit("both", &same)
			page.WriteString(op[1])
		case op[0] == "this":
			emit("both", &same)
			this.WriteString(op[1])
		case changing && next && strings.TrimSpace(op[1]) == "":
			page.WriteString(op[1])
			this.WriteString(op[1])
		default:
			emit("page", &page)
			emit("this", &this)
			same.WriteString(op[1])
		}
	}
	emit("both", &same)
	emit("page", &page)
	emit("this", &this)
	return parts
}
