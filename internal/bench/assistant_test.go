// Package bench measures the assistant doing what people ask of it, with
// a real model, each request in a fresh workspace: did it do what was
// asked, how many tools it called, how many were refused, how long it
// took. It costs a model's time, so it runs only when asked:
//
//	SAMEWAY_BENCH_MODEL=haiku SAMEWAY_BENCH_OUT=bench.jsonl go test ./internal/bench -timeout 120m -v
//
// SAMEWAY_BENCH_RUNS repeats each request (a model varies), and
// SAMEWAY_BENCH_ONLY runs the requests whose names contain it.
package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// result is one request run once.
type result struct {
	Request string   `json:"request"`
	Run     int      `json:"run"`
	Passed  bool     `json:"passed"`
	Why     string   `json:"why,omitempty"`
	Calls   int      `json:"calls"`
	Refused int      `json:"refused"`
	Seconds float64  `json:"seconds"`
	Tools   []string `json:"tools"`
	Refusal []string `json:"refusals,omitempty"`
	Lines   []string `json:"lines"`
	Reply   string   `json:"reply"`
}

func TestAssistant(t *testing.T) {
	model := os.Getenv("SAMEWAY_BENCH_MODEL")
	if model == "" {
		t.Skip("set SAMEWAY_BENCH_MODEL=haiku to measure the assistant with a real model")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude is not on PATH")
	}
	runs, _ := strconv.Atoi(os.Getenv("SAMEWAY_BENCH_RUNS"))
	if runs < 1 {
		runs = 1
	}
	only := os.Getenv("SAMEWAY_BENCH_ONLY")
	exe := build(t)
	var out *os.File
	if p := os.Getenv("SAMEWAY_BENCH_OUT"); p != "" {
		var err error
		if out, err = os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err != nil {
			t.Fatal(err)
		}
		defer out.Close()
	}
	var all []result
	for _, r := range requests() {
		if only != "" && !strings.Contains(r.name, only) {
			continue
		}
		for run := 1; run <= runs; run++ {
			res := runOne(t, exe, model, r, run)
			all = append(all, res)
			if out != nil {
				line, _ := json.Marshal(res)
				out.Write(append(line, '\n'))
			}
			t.Logf("%-14s run %d  %-4s  %2d calls  %2d refused  %5.1fs  %s", res.Request, run, map[bool]string{true: "pass", false: "FAIL"}[res.Passed], res.Calls, res.Refused, res.Seconds, res.Why)
		}
	}
	passed, calls, refused, secs := 0, 0, 0, 0.0
	for _, r := range all {
		if r.Passed {
			passed++
		}
		calls, refused, secs = calls+r.Calls, refused+r.Refused, secs+r.Seconds
	}
	if n := len(all); n > 0 {
		t.Logf("%d of %d passed; %.1f calls, %.1f refused and %.0fs a request", passed, n, float64(calls)/float64(n), float64(refused)/float64(n), secs/float64(n))
	}
}

func runOne(t *testing.T, exe, model string, r request, run int) result {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if r.seed != nil {
		r.seed(t, a)
	}
	a.Chat.Provider, err = llm.New(llm.Config{Provider: "claude-code", Model: model, Workspace: dir, Executable: exe})
	if err != nil {
		t.Fatal(err)
	}
	a.Chat.ProviderErr = nil
	calls := filepath.Join(dir, "calls.log")
	os.Setenv("SAMEWAY_MCP_LOG", calls)
	defer os.Unsetenv("SAMEWAY_MCP_LOG")

	res := result{Request: r.name, Run: run}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	began := time.Now()
	rec, err := a.Chat.SendLive(ctx, "", r.say, "", nil)
	res.Seconds = time.Since(began).Seconds()
	if err != nil {
		res.Why = "the turn failed: " + err.Error()
	} else {
		res.Reply, _ = rec.Fields["content"].(string)
		a.ReloadSchema() // a type or field made in Claude Code's tool process
		res.Why = r.check(a, res.Reply)
		res.Passed = res.Why == ""
	}
	b, _ := os.ReadFile(calls)
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		res.Calls++
		res.Tools = append(res.Tools, f[2])
		res.Lines = append(res.Lines, line)
		if f[1] == "refused" {
			res.Refused++
			why := ""
			if i := strings.Index(line, "\t"); i >= 0 {
				why = line[i+1:]
			}
			res.Refusal = append(res.Refusal, f[2]+": "+why)
		}
	}
	return res
}

// build makes the sameway Claude Code is handed for its tools. Windows
// Smart App Control sometimes refuses a fresh build; another build id is
// another file, judged afresh.
func build(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "sameway.exe")
	for try := 0; try < 10; try++ {
		cmd := exec.Command("go", "build", "-ldflags", fmt.Sprintf("-buildid=bench-%d-%d-%d", os.Getpid(), time.Now().UnixNano(), try), "-o", out, "../../cmd/sameway")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("building sameway: %v\n%s", err, b)
		}
		if exec.Command(out, "--version").Run() == nil {
			return out
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("could not build a sameway this machine lets run")
	return ""
}
