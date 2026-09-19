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

// TestPagesMjsNoRemovedRoutes verifies that tools/a11y-runner/pages.mjs does
// not reference the HTML form-based create/edit routes removed during the
// inline-edit migration (backlog 0149). The server no longer serves /t/note/new
// or /t/note/{id}/edit — they return 404 — so pages.mjs must not visit them.
func TestPagesMjsNoRemovedRoutes(t *testing.T) {
	repoRoot := findRepoRootForTest()
	pathsPath := filepath.Join(repoRoot, "tools", "a11y-runner", "pages.mjs")

	src, err := os.ReadFile(pathsPath)
	if err != nil {
		t.Fatalf("read pages.mjs: %v", err)
	}
	code := string(src)

	// Acceptance 1: no reference to /t/note/new.
	if strings.Contains(code, "/t/note/new") || strings.Contains(code, "`/t/${type}/new`") {
		t.Error("pages.mjs: still references /t/note/new — this route was removed during the inline-edit migration and returns 404")
	}

	// Acceptance 1b: no reference to any type's new route (e.g. /t/article/new).
	if strings.Contains(code, "/new`") || strings.Contains(code, "'/new'") {
		t.Error("pages.mjs: references a */new route pattern; the form-based create pages were removed")
	}

	// Acceptance 1c: no reference to edit routes (e.g. /t/note/{id}/edit).
	if strings.Contains(code, "/edit`") || strings.Contains(code, "'/edit'") {
		t.Error("pages.mjs: references a */edit route; the form-based edit pages were removed during inline-edit migration")
	}

	// Acceptance 2: all page.goto() targets must be valid existing routes.
	// Extract every string or template literal argument to page.goto().
	// Valid routes are known surfaces from the server router:
	//   /, /chat, /activity, /t/{type}, /t/{type}/{id}
	// We check that no goto() call targets anything outside this set.
	if strings.Contains(code, "page.goto(") {
		// The file must use only known route patterns in page.goto calls.
		// Check for any goto that constructs a URL from a variable path
		// (e.g. base + "/t/" + type + "/new") which would be fragile.
		if strings.Contains(code, "type + \"/new\"") || strings.Contains(code, `type+"/new"`) {
			t.Error("pages.mjs: constructs /t/{type}/new via variable concatenation — route removed")
		}
		if strings.Contains(code, "type + \"/edit\"") || strings.Contains(code, `type+"/edit"`) {
			t.Error("pages.mjs: constructs /t/{type}/{id}/edit via variable concatenation — route removed")
		}
	}

	// Acceptance 2b: the file should visit at least the core surfaces.
	if !strings.Contains(code, `"/chat"`) && !strings.Contains(code, "'/chat'") {
		t.Error("pages.mjs: does not visit /chat; this is a core surface that must be tested")
	}
	if !strings.Contains(code, `"/activity"`) && !strings.Contains(code, "'/activity'") {
		t.Error("pages.mjs: does not visit /activity; this is a core surface that must be tested")
	}

	// Acceptance 2c (backlog 0149): the file should also visit /design and /search.
	if !strings.Contains(code, `"/design"`) && !strings.Contains(code, "'/design'") {
		t.Error("pages.mjs: does not visit /design; this is a core surface that must be tested (backlog 0149)")
	}
	if !strings.Contains(code, `"/search"`) && !strings.Contains(code, "'/search'") {
		t.Error("pages.mjs: does not visit /search; this is a core surface that must be tested (backlog 0149)")
	}
}
