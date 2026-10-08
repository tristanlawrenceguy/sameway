package main

import (
	"os"
	"reflect"
	"testing"
)

const baseDebt = `package main

var longFuncs = map[string]int{
	"internal/chat.Service.sendTurn": 122,
	"internal/schema.Set.Complete":   83,
	"internal/server.Server.routes":  107,
	"internal/cli.ctx.openCmd":       111,
}

var splitByName = map[string]bool{
	"internal/server/views_helpers.go": true,
}
`

func TestDebtMayOnlyGoDown(t *testing.T) {
	was, err := parseDebt([]byte(baseDebt))
	if err != nil {
		t.Fatal(err)
	}
	now := map[string]map[string]int{
		"longFuncs": {
			"internal/chat.Service.sendTurn":   131, // raised
			"internal/schema.Set.Complete":     83,  // same
			"internal/server.Server.routes":    90,  // lowered
			"internal/server.Server.shareSave": 81,  // added
			// openCmd removed
		},
		"splitByName": {
			"internal/server/views_helpers.go": 1,
			"internal/server/new_extra.go":     1, // added
		},
	}
	r := compareDebt(was, now)
	for _, want := range []string{
		"internal/chat.Service.sendTurn: raised from 122 to 131",
		"internal/server.Server.shareSave: added to longFuncs",
		"internal/server/new_extra.go: added to splitByName",
	} {
		if !has(r.problems, want) {
			t.Errorf("problems = %q, want %q", r.problems, want)
		}
	}
	if len(r.problems) != 3 {
		t.Errorf("problems = %q, want three (lowering routes and removing openCmd are fine)", r.problems)
	}
}

func TestDebtUnchangedOrShrunkPasses(t *testing.T) {
	was, _ := parseDebt([]byte(baseDebt))
	now := map[string]map[string]int{"longFuncs": {"internal/chat.Service.sendTurn": 100}, "splitByName": {}}
	if r := compareDebt(was, now); len(r.problems) != 0 {
		t.Errorf("problems = %q, want none", r.problems)
	}
	if r := compareDebt(was, was); len(r.problems) != 0 {
		t.Errorf("problems = %q, want none for the same list", r.problems)
	}
}

// The compiled lists and the file parsed as the base would be agree, so a
// change that does not touch debt.go passes against itself.
func TestParseDebtReadsTheRealList(t *testing.T) {
	src, err := os.ReadFile("debt.go")
	if err != nil {
		t.Fatal(err)
	}
	got, err := parseDebt(src)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got["longFuncs"], longFuncs) {
		t.Errorf("parsed longFuncs = %v, want %v", got["longFuncs"], longFuncs)
	}
	if len(got["splitByName"]) != len(splitByName) {
		t.Errorf("parsed splitByName = %v, want %v", got["splitByName"], splitByName)
	}
}
