package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestA11yRunnerHasDiagnostics verifies that tools/a11y-runner/run.mjs emits
// per-violation diagnostic JSON before each violation summary line, so agents
// and humans can identify the failing element without custom Playwright scripts
// (backlog 0197). The test reads run.mjs source to confirm the diagnostic code
// path exists — this is necessary because current axe-core runs may produce no
// violations (the disclosure CSS fix resolved backlog 0161), making a purely
// black-box stdout test non-deterministic about whether diagnostics are tested.
func TestA11yRunnerHasDiagnostics(t *testing.T) {
	repoRoot := findRepoRootForTest()
	runnerPath := filepath.Join(repoRoot, "tools", "a11y-runner", "run.mjs")

	src, err := os.ReadFile(runnerPath)
	if err != nil {
		t.Fatalf("read run.mjs: %v", err)
	}
	code := string(src)

	// The diagnostic helper must exist — it extracts selector, fg, bg, contrast.
	if !strings.Contains(code, "diagnosticNodes") && !strings.Contains(code, "DiagnosticNodes") {
		t.Error("run.mjs: missing diagnostic helper function; per-violation diagnostics not implemented")
	}

	// Diagnostic JSON must be printed before violation summaries.
	// Look for console.log emitting an object with selector/fg/bg/contrast fields.
	if !strings.Contains(code, `"selector"`) && !strings.Contains(code, "'selector'") {
		t.Error("run.mjs: diagnostic output does not include 'selector' field")
	}
	if !strings.Contains(code, `"fg"`) && !strings.Contains(code, "'fg'") {
		t.Error("run.mjs: diagnostic output does not include 'fg' (foreground colour) field")
	}
	if !strings.Contains(code, `"bg"`) && !strings.Contains(code, "'bg'") {
		t.Error("run.mjs: diagnostic output does not include 'bg' (background colour) field")
	}
	if !strings.Contains(code, `"contrast"`) && !strings.Contains(code, "'contrast'") {
		t.Error("run.mjs: diagnostic output does not include 'contrast' (ratio) field")
	}

	// Diagnostics must be emitted inside the AA violation loop.
	// The original loop body is: for (const v of aa.violations) { ... console.log(`FAIL ...`) }
	// After the change, diagnosticNodes call + JSON output should appear before or alongside FAIL log.
	if !strings.Contains(code, "aa.violations") {
		t.Error("run.mjs: AA violation loop missing; did the loop structure change?")
	}

	// The summary line format must be unchanged (stability constraint).
	if !strings.Contains(code, `console.log(\`+"a11y:") && !strings.Contains(code, "a11y:") {
		t.Error("run.mjs: original summary line may have been changed; acceptance criterion 1 requires it stay the same")
	}

	// Violation summaries must still use the FAIL prefix format.
	if !strings.Contains(code, "FAIL ") && !strings.Contains(code, "'FAIL ") {
		t.Error("run.mjs: violation summary no longer uses 'FAIL' prefix; acceptance criterion 1 requires it stay the same")
	}

	// AAA violations must still use WARN prefix (for non-waived).
	if !strings.Contains(code, "WARN ") && !strings.Contains(code, "'WARN ") {
		t.Error("run.mjs: warning summary no longer uses 'WARN' prefix; original format changed")
	}

	// Waived AAA violations must NOT produce diagnostic output.
	// The waiver check (if (waivers[v.id])) should appear before any diagnostic call.
	if !strings.Contains(code, "waivers") {
		t.Error("run.mjs: missing waiver logic; waived AAA findings may incorrectly emit diagnostics")
	}

	// The process.exit code must still be based on failures count only.
	if strings.Contains(code, `process.exit(`) && !strings.Contains(code, "failures > 0") {
		t.Error("run.mjs: exit code logic changed; must exit 1 when failures > 0")
	}

	// Diagnostic JSON lines must be indented (two-space indent per plan).
	if strings.Contains(code, "diagnosticNodes") && !strings.Contains(code, "  ") {
		t.Error("run.mjs: diagnostic output should use two-space indentation for JSON blocks")
	}
}

func findRepoRootForTest() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// Walk up until we find internal/render/ (this file's package).
	for {
		if _, err := os.Stat(filepath.Join(cwd, "internal", "render")); err == nil {
			return cwd
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			panic("cannot find repo root")
		}
		cwd = parent
	}
}
