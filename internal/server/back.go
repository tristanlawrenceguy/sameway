package server

import (
	"net/http"
	"regexp"
	"strings"
)

// Every action ends back where it was taken, not at the top of the page:
// ticking the thirtieth task, logging a habit or answering a question far
// down a long page leaves the person there, told what happened. A form
// says where it sits with a back field, the id of the thing it belongs
// to; the server adds it to the page it returns to, and the browser
// scrolls there. A block on the canvas gives its forms its own id, with
// no script; 26-back.js narrows it to the row or card the form is in.

// backField is the form field that says where an action was taken.
const backField = "back"

// cameFrom is the place on the page an action was taken, as the part of
// an address after #, for the page to come back to it. Only an id is
// taken, so nothing else can be put into the address.
func cameFrom(back string) string {
	if back == "" || len(back) > 200 {
		return ""
	}
	for _, c := range back {
		if !(c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return ""
		}
	}
	return "#" + back
}

// withBack is where an action returns to with the place it was taken, when
// the form said one and the address has no place of its own.
func withBack(to, back string) string {
	if strings.Contains(to, "#") {
		return to
	}
	return to + cameFrom(back)
}

// placeOf is where a posted form says it was taken: the last back it
// sent, the closest, as a script or a card adds its own after the block's.
func placeOf(r *http.Request) string {
	if r.Method != http.MethodPost {
		return ""
	}
	r.PostFormValue(backField) // parses the form, of either kind
	if all := r.PostForm[backField]; len(all) > 0 {
		return all[len(all)-1]
	}
	return ""
}

var postForm = regexp.MustCompile(`(?i)<form\b[^>]*\bmethod="post"[^>]*>`)

// withBlock gives every form in a block's HTML the block to come back to.
func withBlock(html, id string) string {
	return postForm.ReplaceAllStringFunc(html, func(tag string) string {
		return tag + `<input type="hidden" name="` + backField + `" value="` + id + `">`
	})
}
