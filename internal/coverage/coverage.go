// Package coverage tracks which checks of the DASH-IF Conformance tool have
// been ported to dashcheck. The inventory lives in docs/dashif-checks.json
// (built by tools/dashif-inventory.py) and is rendered to docs/dashif-coverage.md.
// docs/rules.md lists all dashcheck rules with the checks they cover.
package coverage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tobbee/dashcheck/report"
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

// RulesMarkdown renders the list of all dashcheck rules, by group, with the
// DASH-IF checks that each covers.
func (inv *Inventory) RulesMarkdown(rules []report.Rule) string {
	covers := map[string][]string{}
	for _, c := range inv.Checks {
		for _, r := range c.Rules {
			covers[r] = append(covers[r], c.ID)
		}
	}
	byGroup := map[string][]report.Rule{}
	own := map[string]int{}
	for _, r := range rules {
		byGroup[r.Group] = append(byGroup[r.Group], r)
		if len(covers[r.ID]) == 0 {
			own[r.Group]++
			own[""]++
		}
	}
	var b strings.Builder
	b.WriteString("# dashcheck rules\n\n")
	b.WriteString("<!-- Generated from the rule registry and dashif-checks.json by `go test ./internal/coverage -update`. Do not edit. -->\n\n")
	fmt.Fprintf(&b, "All %d rules that dashcheck reports, as `dashcheck -rules` lists them. ", len(rules))
	b.WriteString("Severities follow the spec wording: **error** for SHALL, **warning** for SHOULD, and **info** for information. " +
		"The last column names the checks of the [DASH-IF Conformance tool](dashif-coverage.md) that a rule covers. ")
	fmt.Fprintf(&b, "The %d rules marked **dashcheck** have no counterpart in that tool.\n\n", own[""])

	b.WriteString("| Group | Rules | dashcheck only |\n|---|---:|---:|\n")
	for _, g := range report.Groups {
		if len(byGroup[g]) > 0 {
			fmt.Fprintf(&b, "| [%s](#%s) | %d | %d |\n", g, anchor(g), len(byGroup[g]), own[g])
		}
	}
	fmt.Fprintf(&b, "| **Total** | **%d** | **%d** |\n", len(rules), own[""])

	for _, g := range report.Groups {
		if len(byGroup[g]) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n\n", g)
		b.WriteString("| Rule | Severity | Description | Spec | DASH-IF checks |\n|---|---|---|---|---|\n")
		for _, r := range byGroup[g] {
			checks := "**dashcheck**"
			if ids := covers[r.ID]; len(ids) > 0 {
				checks = strings.Join(ids, ", ")
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", r.ID, r.Severity, escape(r.Desc), escape(r.SpecRef), checks)
		}
	}
	return b.String()
}

// anchor returns the GitHub anchor of a Markdown heading.
func anchor(heading string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(heading) {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_':
			b.WriteRune(c)
		case c == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}
