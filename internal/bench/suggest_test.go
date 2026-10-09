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

// The suggest action measured on the messages: a task suggested where one
// is asked and none where not, its day counted right.
func TestSuggest(t *testing.T) {
	model, base := os.Getenv("SAMEWAY_BENCH_MODEL"), os.Getenv("SAMEWAY_BENCH_BASE_URL")
	if model == "" || base == "" {
		t.Skip("set SAMEWAY_BENCH_MODEL and SAMEWAY_BENCH_BASE_URL to measure the suggest action with a model server")
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
	act, _ := a.Store.Create("action", map[string]any{"title": "Suggest a task", "kind": "suggest", "make": "task"})
	var made, day, n int
	began := time.Now()
	for run := 0; run < runs; run++ {
		for _, c := range triageCases {
			rec, _ := a.Store.Create("email", map[string]any{"subject": strings.SplitN(c.text, "\n", 2)[0], "body": c.text, "received": triageSent.UTC().Format(time.RFC3339)})
			_ = a.Chat.Suggest(context.Background(), act, "email", rec.ID)
			var fields map[string]any
			for _, p := range a.Chat.Suggestions() {
				if p.Fields["from"] == "email/"+rec.ID {
					act, _ := p.Fields["action"].(map[string]any)
					fields, _ = act["fields"].(map[string]any)
					a.Store.Update(records.ProposalType, p.ID, map[string]any{"state": "dismissed"})
				}
			}
			n++
			if (fields != nil) == c.task {
				made++
			} else {
				t.Logf("%-20s made %v, want %v", c.name, fields != nil, c.task)
			}
			due, _ := fields["due"].(string)
			if len(due) >= 10 {
				due = due[:10]
			}
			if !c.task || fields == nil || due == c.due {
				day++
			} else {
				t.Logf("%-20s due %q, want %q", c.name, due, c.due)
			}
		}
	}
	t.Logf("SUGGEST %s: task or not %d/%d; day %d/%d; %.1fs each", model, made, n, day, n, time.Since(began).Seconds()/float64(n))
}
