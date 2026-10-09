package ui

import (
	"fmt"
	"html/template"
	"testing"
)

func fakeRender(p Part) template.HTML {
	return template.HTML(fmt.Sprintf("[%s %v]", p.Component(), p.Props()))
}

func TestFormWritesTheConventionsOnce(t *testing.T) {
	f := Form{Action: "/t/task/a1/delete", From: "/t/task?x=1&y=2", Back: "row-a1",
		Hidden: Hidden("id", `a"1`), Button: &Button{Label: "Delete", Variant: Quiet}}
	got := string(f.HTML(fakeRender))
	want := `<form method="post" action="/t/task/a1/delete">` +
		`<input type="hidden" name="from" value="/t/task?x=1&amp;y=2">` +
		`<input type="hidden" name="back" value="row-a1">` +
		`<input type="hidden" name="id" value="a&#34;1">` +
		`[button map[label:Delete type:submit variant:quiet]]</form>`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestFormGetWithTarget(t *testing.T) {
	f := Form{Action: "https://example.com/new", Get: true, Target: "_blank", Class: "sw-stack", Body: "<p>x</p>"}
	got := string(f.HTML(fakeRender))
	want := `<form method="get" action="https://example.com/new" target="_blank" rel="noopener" class="sw-stack"><p>x</p></form>`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}
