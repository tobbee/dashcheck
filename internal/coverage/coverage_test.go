package coverage

import (
	"flag"
	"os"
	"slices"
	"testing"

	_ "github.com/tobbee/dashcheck" // registers all rules
	"github.com/tobbee/dashcheck/report"
)

var update = flag.Bool("update", false, "rewrite docs/dashif-coverage.md and docs/rules.md")

const (
	inventoryFile = "../../docs/dashif-checks.json"
	markdownFile  = "../../docs/dashif-coverage.md"
	rulesFile     = "../../docs/rules.md"
)

func TestInventory(t *testing.T) {
	inv, err := Load(inventoryFile)
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, r := range report.Rules() {
		known[r.ID] = true
		if !slices.Contains(report.Groups, r.Group) {
			t.Errorf("rule %s has no display group (see report.groupOf)", r.ID)
		}
	}
	ids := map[string]bool{}
	for _, c := range inv.Checks {
		if ids[c.ID] {
			t.Errorf("duplicate check ID %s", c.ID)
		}
		ids[c.ID] = true
		if groupOf(c.ID) < 0 {
			t.Errorf("%s: no group", c.ID)
		}
		switch c.Status {
		case StatusDone, StatusPartial:
			if len(c.Rules) == 0 {
				t.Errorf("%s is %s but names no dashcheck rule", c.ID, c.Status)
			}
		case StatusNA:
			if c.Note == "" {
				t.Errorf("%s is n/a without a note", c.ID)
			}
		case StatusTodo:
		default:
			t.Errorf("%s: unknown status %q", c.ID, c.Status)
		}
		for _, r := range c.Rules {
			if !known[r] {
				t.Errorf("%s refers to unknown rule %s", c.ID, r)
			}
		}
	}
}

func TestMarkdownUpToDate(t *testing.T) {
	inv, err := Load(inventoryFile)
	if err != nil {
		t.Fatal(err)
	}
	checkGenerated(t, markdownFile, inv.Markdown())
	checkGenerated(t, rulesFile, inv.RulesMarkdown(report.Rules()))
}

// checkGenerated compares a generated file with its content, or rewrites it with -update.
func checkGenerated(t *testing.T, path, content string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(current) != content {
		t.Errorf("%s is out of date; run: go test ./internal/coverage -update", path)
	}
}
