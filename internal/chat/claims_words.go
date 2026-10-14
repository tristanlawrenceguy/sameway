package chat

import (
	"context"
	"log"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A reply in the person's words. Replies named the tools they used or
// could not ("I couldn't use add_component for this"), words the person
// never sees anywhere else and cannot act on; the prompt says to say
// "page", not block or component, and a model writing about its own steps
// reached for their names anyway. A reply naming a tool is asked once to
// say it again in the person's words, as a reply claiming what was not
// made is (claims.go).

// toolWord is a tool's name as a word: lower case, at least one
// underscore, as every tool is named.
var toolWord = regexp.MustCompile(`\b[a-z]+(?:_[a-z]+)+\b`)

// inTheirWords is what to tell a model whose reply names one of its tools,
// "" when it names none.
func (s *Service) inTheirWords(reply string) string {
	names := map[string]bool{}
	for _, t := range s.Tools() {
		names[t.Name] = true
	}
	var named []string
	for _, w := range toolWord.FindAllString(reply, -1) {
		if names[w] {
			named = append(named, w)
		}
	}
	if len(named) == 0 {
		return ""
	}
	log.Printf("chat: the reply names %s; asking for it in the person's words", strings.Join(named, ", "))
	return fromSameway("your reply names " + strings.Join(named, ", ") + ", a tool's name the person does not see anywhere and cannot act on. Say the same again in their words: what you did, or what you could not do and what they can do instead." + answerAgain)
}

// stillWrong is what a reply still wrong after it was asked again carries:
// that nothing changed, when it claimed a change; nothing more otherwise.
func (s *Service) stillWrong(ctx context.Context, reply, said string, changes []records.Change) string {
	if s.claimWrong(ctx, reply, said, changes) {
		return nothingChanged
	}
	return ""
}
