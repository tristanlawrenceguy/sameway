package chat

import (
	"context"
	"log"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// pagePath is a record's page as the model would name it: /t/<type>/<id>.
var pagePath = regexp.MustCompile(`/t/([a-z][a-z0-9_-]*)/([a-z0-9]{8,})`)

// ClaimResult reports whether one URL mentioned in an assistant reply
// points to a real resource. Exists is false when the store has no such
// record, or when the type named in the path is not known to this workspace.
type ClaimResult struct {
	URL    string
	Exists bool
}

// VerifyClaims scans reply for /t/<type>/<id> URLs and checks each one
// against the store. Only types that exist in the schema are checked; a
// made-up type yields Exists: false because the model is wrong about it too.
func VerifyClaims(ctx context.Context, store *store.Store, reply string) ([]ClaimResult, error) {
	var results []ClaimResult
	for _, m := range pagePath.FindAllStringSubmatch(reply, -1) {
		typ, id := m[1], m[2]
		results = append(results, ClaimResult{URL: m[0]})
		if _, ok := store.Types().Get(typ); !ok {
			continue // will be Exists:false by zero value
		}
		if _, err := store.Get(typ, id); err != nil {
			continue // also Exists:false
		}
		results[len(results)-1].Exists = true
	}
	return results, nil
}

// saidDone is a reply that opens by saying a change was made, in the words
// a model reports one with: "Created your task", "I've added", "Moved it".
// Only the opening: a model that made something says so first, and an
// answer that lists records says "made by you" or "logged 3 times" further
// on. Read anywhere, a list of actions was taken for a claim, the model was
// asked again, and it told the person their message had come through empty.
var saidDone = regexp.MustCompile(`(?i)\A\W*(?:I(?:'ve| have)?\s+|Done[,:—-]?\s+)?(?:created|added|made|scheduled|moved|deleted|removed|renamed|updated|changed|marked|logged|saved|booked|archived|ticked|set up)\b`)

// unmade is what to tell a model whose reply says it made something it
// did not: pages it named that do not exist, or a change said in a turn
// that changed nothing, which is the model answering in words where a tool
// was needed (copying how earlier replies read). Empty when the reply
// stands. A change made outside (Claude Code's tools) is in the log.
func (s *Service) unmade(ctx context.Context, reply, said string, changes []records.Change) string {
	claims, _ := VerifyClaims(ctx, s.Store, reply)
	var missing []string
	for _, c := range claims {
		if !c.Exists {
			missing = append(missing, c.URL)
		}
	}
	if len(missing) > 0 {
		log.Printf("chat: the reply named %s, which does not exist; asking the model to make it", strings.Join(missing, ", "))
		return fromSameway("there is no page at " + strings.Join(missing, ", ") + ": nothing was made. Make it with the tools, then say where it is." + answerAgain)
	}
	if s.runsToolsOutside() {
		changes = s.changesAfter(said)
	}
	if len(changes) == 0 && saidDone.MatchString(reply) {
		log.Printf("chat: the reply says it changed something, and nothing changed; asking the model again")
		return fromSameway("Nothing has changed in this turn: no tool made or changed anything. If the person asked for a change, make it now with the tools, then say what you did. If not, answer them without saying anything was done." + answerAgain)
	}
	return ""
}

// fromSameway is a word to the model that comes from Sameway, not the
// person. Every such word goes in the person's turn (a provider takes no
// other), and said bare, the model took it for what they wrote and told
// them their message had come through empty. OWASP's LLM01 asks for the
// same: say where text comes from, so instructions and data are told apart.
func fromSameway(note string) string {
	return "(From Sameway, not the person: " + note + ")"
}

// answerAgain ends a note that interrupts the model's answer.
const answerAgain = " Then answer the person's last message."

// nothingChanged is said under a reply that still says it changed
// something when nothing changed, so the person is not left believing it.
// It says it is Sameway's: under the assistant's words it read as the
// assistant taking its own reply back, and the model, reading it in later
// turns, as something it had said.
const nothingChanged = "\n\n(Sameway checked: nothing was changed. Ask again if something should be.)"
