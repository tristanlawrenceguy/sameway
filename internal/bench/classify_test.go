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
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// The classify action measured on the triage messages, with the starter
// tags: "to do" where the message asks something, "important" where it
// matters. Each tag is scored on its own, all at once and each apart.
func TestClassify(t *testing.T) {
	model, base := os.Getenv("SAMEWAY_BENCH_MODEL"), os.Getenv("SAMEWAY_BENCH_BASE_URL")
	if model == "" || base == "" {
		t.Skip("set SAMEWAY_BENCH_MODEL and SAMEWAY_BENCH_BASE_URL to measure the classify action with a model server")
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
	a.Store.Create("tag", map[string]any{"name": "important", "means": "Money owed, health, an official or legal deadline (tax, passport, insurance, a lease), school, a work deadline, or someone waiting on my answer. Not plans with friends, errands or reminders of habit."})
	for _, apart := range []bool{false, true} {
		act, _ := a.Store.Create("action", map[string]any{"title": "Sort", "kind": "classify", "apart": apart})
		var todo, matters, n int
		began := time.Now()
		for run := 0; run < runs; run++ {
			for _, c := range triageCases {
				rec, _ := a.Store.Create("email", map[string]any{"subject": strings.SplitN(c.text, "\n", 2)[0], "body": c.text})
				_ = a.Chat.Classify(context.Background(), act, "email", rec.ID)
				got, _ := a.Store.Get("email", rec.ID)
				tags := map[string]bool{}
				if l, ok := got.Fields["tags"].([]any); ok {
					for _, x := range l {
						tags[x.(string)] = true
					}
				}
				n++
				if tags["to do"] == c.task {
					todo++
				} else {
					t.Logf("apart=%v %-20s to do: gave %v, want %v", apart, c.name, tags["to do"], c.task)
				}
				if tags["important"] == (c.task && c.important) {
					matters++
				}
			}
		}
		t.Logf("CLASSIFY %s apart=%v: to do %d/%d; important %d/%d; %.1fs each", model, apart, todo, n, matters, n, time.Since(began).Seconds()/float64(n))
	}
}

// The check with past choices, measured: the person's choices on half the
// messages (each right tag kept, each wrong one taken off) are shown when
// the other half is tagged, against tagging it without them.
func TestClassifyFollowsJudgement(t *testing.T) {
	model, base := os.Getenv("SAMEWAY_BENCH_MODEL"), os.Getenv("SAMEWAY_BENCH_BASE_URL")
	if model == "" || base == "" {
		t.Skip("set SAMEWAY_BENCH_MODEL and SAMEWAY_BENCH_BASE_URL")
	}
	runs, _ := strconv.Atoi(os.Getenv("SAMEWAY_BENCH_RUNS"))
	if runs < 1 {
		runs = 1
	}
	for run := 0; run < runs; run++ {
		dir := t.TempDir()
		if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
			t.Fatal(err)
		}
		a, err := app.Load(dir, false)
		if err != nil {
			t.Fatal(err)
		}
		if a.Chat.Provider, err = llm.New(llm.Config{Provider: "openai", BaseURL: base, Model: model, MaxTokens: 2048}); err != nil {
			t.Fatal(err)
		}
		a.Chat.ProviderErr = nil
		a.Store.Create("tag", map[string]any{"name": "to do", "means": "Something I have to do: pay, reply, book, bring, send, sign, renew, attend, buy or call. Not newsletters, adverts, receipts for what is paid, or notices that need nothing from me."})
		learn, _ := a.Store.Create("action", map[string]any{"title": "Sort", "kind": "classify", "tags": []any{"to do"}})
		checked, _ := a.Store.Create("action", map[string]any{"title": "Sort, checked", "kind": "classify", "tags": []any{"to do"}, "check": true, "examples": 10})
		taught, test := triageCases[:15], triageCases[15:]
		for _, c := range taught {
			rec, _ := a.Store.Create("email", map[string]any{"subject": strings.SplitN(c.text, "\n", 2)[0], "body": c.text})
			_ = a.Chat.Classify(context.Background(), learn, "email", rec.ID)
			cls, _ := a.Store.List("classification", store.ListOptions{})
			for _, cl := range cls {
				if cl.Fields["record"] == "email/"+rec.ID {
					state := "removed"
					if c.task {
						state = "confirmed"
					}
					a.Store.Update("classification", cl.ID, map[string]any{"state": state, "confirmed_by": map[bool]string{true: "person", false: ""}[c.task]})
				}
			}
		}
		for _, act := range []struct {
			name string
			id   string
		}{{"without", learn.ID}, {"with", checked.ID}} {
			action, _ := a.Store.Get("action", act.id)
			right := 0
			for _, c := range test {
				rec, _ := a.Store.Create("email", map[string]any{"subject": strings.SplitN(c.text, "\n", 2)[0], "body": c.text})
				_ = a.Chat.Classify(context.Background(), action, "email", rec.ID)
				got, _ := a.Store.Get("email", rec.ID)
				has := strings.Contains(fmtTags(got.Fields["tags"]), "to do")
				if has == c.task {
					right++
				}
			}
			t.Logf("JUDGEMENT run %d %s past choices: to do right %d/%d", run+1, act.name, right, len(test))
		}
		a.Close()
	}
}

func fmtTags(v any) string {
	var out []string
	if l, ok := v.([]any); ok {
		for _, x := range l {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
	}
	return strings.Join(out, ",")
}
