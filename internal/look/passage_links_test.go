package look

import "testing"

// A passage says which of its words are links and where each goes; a
// passage with none says nothing of links; a hidden link is not read.
func TestAPassageSaysItsLinks(t *testing.T) {
	t.Parallel()
	o, err := Page(`<html><body><main><h1>Chat</h1><p>Filed <a href="/t/note/seeds1">Seeds to buy</a> and <a href="/t/task/dig2">Dig the <b>pond</b></a>.<a href="/x" hidden>gone</a></p><p>No links here.</p></main></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Text) != 2 {
		t.Fatalf("two passages: %+v", o.Text)
	}
	got := o.Text[0].Links
	if len(got) != 2 || got[0].Words != "Seeds to buy" || got[0].Href != "/t/note/seeds1" || got[1].Words != "Dig the pond" || got[1].Href != "/t/task/dig2" {
		t.Errorf("the passage's links by words and where they go: %+v", got)
	}
	if o.Text[1].Links != nil {
		t.Errorf("a passage with no link says none: %+v", o.Text[1].Links)
	}
}
