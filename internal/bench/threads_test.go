package bench

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// say is one email in a conversation: whether the person wrote it, and
// what it says.
type say struct {
	mine bool
	text string
}

// threadCases are conversations whose last email is to do, or not, only
// when read with what came before it.
var threadCases = []struct {
	name string
	says []say
	todo bool
}{
	{"invoice sent", []say{{true, "Could you send me the invoice for September?"}, {false, "Sure, it's attached."}}, false},
	{"thanks", []say{{true, "Here's the report you asked for."}, {false, "Thanks, got it!"}}, false},
	{"review done", []say{{false, "Could you review the draft?"}, {true, "Will do by Friday."}, {false, "Great, thanks!"}}, false},
	{"delivered", []say{{false, "Your order has shipped."}, {false, "Your order was delivered."}}, false},
	{"lunch booked", []say{{false, "Team lunch Friday?"}, {true, "I'm in."}, {false, "Booked, see you there."}}, false},
	{"price", []say{{true, "What would it cost?"}, {false, "200 pounds. Shall I go ahead?"}}, true},
	{"tuesday", []say{{true, "Are you free Tuesday?"}, {false, "Tuesday works. 3pm?"}}, true},
	{"chasing", []say{{false, "Quote attached."}, {false, "Did you get a chance to look? I need to know by Wednesday."}}, true},
	{"missing page", []say{{true, "I've sent you the forms."}, {false, "One page is missing a signature. Please resend it."}}, true},
	{"move call", []say{{false, "Can we move our call?"}, {true, "Sure, when?"}, {false, "Friday at 2pm?"}}, true},
	{"dessert", []say{{false, "Dinner on Saturday?"}, {true, "Yes!"}, {false, "Great, bring dessert."}}, true},
	{"which one", []say{{true, "Could you book the room?"}, {false, "The big one or the small one?"}}, true},
}

// The classify action measured on the last email of each conversation,
// read alone and read with what came before it.
func TestThreads(t *testing.T) {
	model, base := os.Getenv("SAMEWAY_BENCH_MODEL"), os.Getenv("SAMEWAY_BENCH_BASE_URL")
	if model == "" || base == "" {
		t.Skip("set SAMEWAY_BENCH_MODEL and SAMEWAY_BENCH_BASE_URL")
	}
	runs, _ := strconv.Atoi(os.Getenv("SAMEWAY_BENCH_RUNS"))
	if runs < 1 {
		runs = 1
	}
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.Chat.Provider, err = llm.New(llm.Config{Provider: "openai", BaseURL: base, Model: model, MaxTokens: 2048}); err != nil {
		t.Fatal(err)
	}
	a.Chat.ProviderErr = nil
	a.Store.Create("tag", map[string]any{"name": "to do", "means": "Something I have to do: pay, reply, book, bring, send, sign, renew, attend, buy or call. Not newsletters, adverts, receipts for what is paid, or notices that need nothing from me."})
	a.Store.Create("tag", map[string]any{"name": "nothing to do", "means": "Needs nothing from me.", "alone": true})
	act, _ := a.Store.Create("action", map[string]any{"title": "Sort", "kind": "classify"})
	for _, withThread := range []bool{false, true} {
		right, n := 0, 0
		for run := 0; run < runs; run++ {
			for _, c := range threadCases {
				root := ""
				var last string
				at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
				for i, m := range c.says {
					fields := map[string]any{"subject": "Re: " + c.name, "body": m.text, "from_me": m.mine, "received": at.Add(time.Duration(i) * time.Hour).Format(time.RFC3339)}
					if !m.mine {
						fields["from"] = "Sam <sam@example.com>"
					}
					if withThread && root != "" {
						fields["thread"] = root
					}
					rec, _, err := records.WriteKept(a.Store, records.Who{Actor: "system"}, "created", "email", "", fields)
					if err != nil {
						t.Fatal(err)
					}
					if root == "" {
						root = rec.ID
					}
					last = rec.ID
				}
				_ = a.Chat.Classify(context.Background(), act, "email", last)
				got, _ := a.Store.Get("email", last)
				has := strings.Contains(","+fmtTags(got.Fields["tags"])+",", ",to do,")
				n++
				if has == c.todo {
					right++
				} else {
					t.Logf("thread=%v %-14s to do: gave %v, want %v", withThread, c.name, has, c.todo)
				}
			}
		}
		t.Logf("THREADS %s read with the conversation=%v: to do right %d/%d", model, withThread, right, n)
	}
}
