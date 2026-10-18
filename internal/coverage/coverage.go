// Package coverage tracks which checks of the DASH-IF Conformance tool have
// been ported to dashcheck. The inventory lives in docs/dashif-checks.json
// (built by tools/dashif-inventory.py) and is rendered to docs/dashif-coverage.md.
package coverage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Statuses of a check.
const (
	StatusDone    = "done"    // ported: a dashcheck rule covers it
	StatusPartial = "partial" // partly covered, see note
	StatusTodo    = "todo"    // not ported yet
	StatusNA      = "n/a"     // will not be ported, see note
)

// Check is one check in the DASH-IF Conformance tool.
type Check struct {
	ID      string   `json:"id"`
	Source  string   `json:"source"`
	Context string   `json:"context"`
	Text    string   `json:"text"`
	Status  string   `json:"status"`
	Rules   []string `json:"rules"`
	Note    string   `json:"note"`
}

// Inventory is the list of DASH-IF checks.
type Inventory struct {
	Upstream struct {
		Repo   string `json:"repo"`
		Commit string `json:"commit"`
		Date   string `json:"date"`
	} `json:"upstream"`
	Checks []Check `json:"checks"`
}

// Load reads an inventory file.
func Load(path string) (*Inventory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var inv Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &inv, nil
}

// group is a set of checks shown together.
type group struct {
	title  string
	prefix string // ID prefix, e.g. "SCH-" or "DVB-"
}

var groups = []group{
	{"Schematron: schematron.sch", "SCH-"},
	{"Schematron: dvb-dash.sch", "SCH-DVB-"},
	{"Schematron: ac4-generic.sch", "SCH-AC4-"},
	{"Module: Common", "COMMON-"},
	{"Module: IOP", "IOP-"},
	{"Module: CMAF", "CMAF-"},
	{"Module: Low Latency", "LL-"},
	{"Module: DVB", "DVB-"},
	{"Module: HbbTV", "HBBTV-"},
	{"Module: CTA WAVE", "WAVE-"},
	{"Module: WAVE HLS interop", "WAVEHLS-"},
	{"Module: Dolby", "DOLBY-"},
}

// groupOf returns the index of the group a check belongs to, or -1.
func groupOf(id string) int {
	best := -1
	for i, g := range groups {
		// The longest matching prefix wins, so SCH-DVB- is not counted as SCH-.
		if strings.HasPrefix(id, g.prefix) && (best < 0 || len(g.prefix) > len(groups[best].prefix)) {
			best = i
		}
	}
	return best
}

func escape(s string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ", "<", "&lt;", ">", "&gt;").Replace(s)
}

// Markdown renders the coverage report.
func (inv *Inventory) Markdown() string {
	byGroup := make([][]Check, len(groups))
	for _, c := range inv.Checks {
		if g := groupOf(c.ID); g >= 0 {
			byGroup[g] = append(byGroup[g], c)
		}
	}
	var b strings.Builder
	b.WriteString("# DASH-IF Conformance tool coverage\n\n")
	b.WriteString("<!-- Generated from dashif-checks.json by `go test ./internal/coverage -update`. Do not edit. -->\n\n")
	fmt.Fprintf(&b, "Checks of the [DASH-IF Conformance tool](%s) at commit %s (%s), and the dashcheck rules that cover them. ",
		inv.Upstream.Repo, inv.Upstream.Commit, inv.Upstream.Date)
	b.WriteString("Statuses: **done** is covered by a dashcheck rule, **partial** is partly covered, **todo** is not ported yet, " +
		"and **n/a** will not be ported (see the note).\n\n")

	b.WriteString("| Group | Checks | Done | Partial | Todo | n/a |\n|---|---:|---:|---:|---:|---:|\n")
	total := map[string]int{}
	for i, g := range groups {
		n := map[string]int{}
		for _, c := range byGroup[i] {
			n[c.Status]++
			total[c.Status]++
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d |\n", g.title, len(byGroup[i]),
			n[StatusDone], n[StatusPartial], n[StatusTodo], n[StatusNA])
	}
	fmt.Fprintf(&b, "| **Total** | **%d** | **%d** | **%d** | **%d** | **%d** |\n",
		len(inv.Checks), total[StatusDone], total[StatusPartial], total[StatusTodo], total[StatusNA])

	for i, g := range groups {
		fmt.Fprintf(&b, "\n## %s\n\n", g.title)
		b.WriteString("| ID | Check | Status | dashcheck rules | Note |\n|---|---|---|---|---|\n")
		for _, c := range byGroup[i] {
			text := c.Text
			if c.Context != "" {
				text = "`" + c.Context + "`: " + text
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", c.ID, escape(text), c.Status,
				escape(strings.Join(c.Rules, ", ")), escape(c.Note))
		}
	}
	return b.String()
}
