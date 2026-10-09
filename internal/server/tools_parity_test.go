package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// What a person does from a page, the assistant can do with a tool, or it
// is theirs alone by design. Each page action's route says which: the tool
// that does the same, or why it is a person's. There is no third answer:
// a page action with neither is a gap, and a new one fails here until the
// assistant has a tool for it or the reason it should not is said. (A tool
// it names that the assistant does not have stops the server starting.)
func TestEveryPageActionIsTheAssistantsOrSaysWhyNot(t *testing.T) {
	t.Parallel()
	for _, r := range pageActions() {
		switch {
		case r.Tool == "" && r.Persons == "":
			t.Errorf("%s: which tool does the same for the assistant, or why is it a person's? Say so in its route", r.Pattern)
		case r.Tool != "" && r.Persons != "":
			t.Errorf("%s says both a tool and why it is a person's; it is one or the other", r.Pattern)
		}
	}
}

// pageActions are the routes a page's form posts to: not the API's, which
// are what a tool or a record names, and not a hook's.
func pageActions() []server.Route {
	var out []server.Route
	for _, r := range server.Routes() {
		path, post := strings.CutPrefix(r.Pattern, "POST ")
		if post && !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/hook/") {
			out = append(out, r)
		}
	}
	return out
}

// The assistant does no more than the pages let the one it speaks for do:
// a page action that is the owner's alone has only owner's tools, so
// someone let in cannot ask their assistant for what their pages refuse
// them. The other way round, a person may do on a page what their
// assistant may not, for a reason said here.
var narrowerForTheAssistant = map[string]string{
	"POST /t/{type}/{id}/discard": "undo_change takes back anyone's change, so it is the owner's; a person discards only the record they just added",
}

func TestTheAssistantIsNoWiderThanThePages(t *testing.T) {
	t.Parallel()
	for _, r := range pageActions() {
		if r.Tool == "" {
			continue
		}
		owners := r.Access == "owner"
		alone := chat.OwnersAlone(r.Tool)
		switch {
		case owners && !alone:
			t.Errorf("%s is the owner's, but %s is anyone's: someone let in could ask for what the page refuses them", r.Pattern, r.Tool)
		case !owners && alone && narrowerForTheAssistant[r.Pattern] == "":
			t.Errorf("%s is anyone's, but %s is the owner's: say why in narrowerForTheAssistant, or open the tool", r.Pattern, r.Tool)
		}
	}
}
