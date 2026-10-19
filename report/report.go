// Package report defines the findings produced by all checks.
//
// Every check is described by a Rule with a stable ID, a default severity and,
// where known, a reference to the specification clause it enforces.
// Checks produce Findings by calling Rule.New.
package report

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

// Severity of a finding.
type Severity int

const (
	Info Severity = iota
	Warning
	Error
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return fmt.Sprintf("severity(%d)", int(s))
	}
}

// MarshalJSON writes the severity as a string.
func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON parses a severity string.
func (s *Severity) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	switch str {
	case "info":
		*s = Info
	case "warning":
		*s = Warning
	case "error":
		*s = Error
	default:
		return fmt.Errorf("unknown severity %q", str)
	}
	return nil
}

// Rule describes one check.
type Rule struct {
	ID       string   `json:"id"`
	Severity Severity `json:"severity"`
	Desc     string   `json:"desc"`
	SpecRef  string   `json:"specRef,omitempty"`
	Group    string   `json:"group"` // set by Register from the ID, one of Groups
}

// Groups are the rule groups in display order.
var Groups = []string{
	"MPD syntax", "MPD", "MPD updates", "Segment addressing", "Segments", "Live timing and HTTP",
	"DASH-IF IOP", "CMAF", "Low latency", "DVB-DASH", "HbbTV", "Dolby AC-4", "CTA WAVE",
	"DASH-HLS interoperability (CTA-5005)",
}

// profileGroups maps the second part of an ID such as SEG-DVB-SAP to a group.
var profileGroups = map[string]string{
	"IOP": "DASH-IF IOP", "CMAF": "CMAF", "LL": "Low latency", "EMSG": "Low latency", "DVB": "DVB-DASH",
	"HBBTV": "HbbTV", "AC4": "Dolby AC-4", "WAVE": "CTA WAVE", "HLS": "DASH-HLS interoperability (CTA-5005)",
	"UPDATE": "MPD updates",
}

// groupOf derives the group of a rule from its ID.
func groupOf(id string) string {
	parts := strings.Split(id, "-")
	if len(parts) > 1 {
		if g, ok := profileGroups[parts[1]]; ok {
			return g
		}
	}
	switch parts[0] {
	case "MPD":
		if len(parts) > 1 && (parts[1] == "XML" || parts[1] == "XSD" || parts[1] == "PARSE") {
			return "MPD syntax"
		}
		return "MPD"
	case "ADDR":
		return "Segment addressing"
	case "INIT", "SEG", "XCHK":
		return "Segments"
	case "LIVE", "HTTP", "CLOCK":
		return "Live timing and HTTP"
	}
	return "Other"
}

// Location points at where a finding applies.
type Location struct {
	URL     string `json:"url,omitempty"`
	Line    int    `json:"line,omitempty"`
	Col     int    `json:"col,omitempty"`
	Path    string `json:"path,omitempty"`    // element path in the MPD, or box path in a segment
	RepID   string `json:"repID,omitempty"`   // Representation@id
	Segment string `json:"segment,omitempty"` // segment number or time
}

func (l Location) String() string {
	var parts []string
	if l.RepID != "" {
		parts = append(parts, "rep="+l.RepID)
	}
	if l.Segment != "" {
		parts = append(parts, "seg="+l.Segment)
	}
	if l.Path != "" {
		parts = append(parts, l.Path)
	}
	if l.Line > 0 {
		parts = append(parts, fmt.Sprintf("line %d:%d", l.Line, l.Col))
	}
	if len(parts) == 0 && l.URL != "" {
		parts = append(parts, l.URL)
	}
	return strings.Join(parts, " ")
}

// Finding is one reported problem or observation.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	SpecRef  string   `json:"specRef,omitempty"`
	Location Location `json:"location"`
	// Count is the number of segments with the same finding, when merged by Collapse.
	Count int `json:"count,omitempty"`
}

func (f Finding) String() string {
	loc := f.Location.String()
	if loc != "" {
		loc = " [" + loc + "]"
	}
	more := ""
	if f.Count > 1 {
		more = fmt.Sprintf(" (and %d more segments)", f.Count-1)
	}
	return fmt.Sprintf("%-7s %s: %s%s%s", f.Severity, f.Rule, f.Message, more, loc)
}

// Collapse merges segment findings with the same rule and Representation
// into the first one, counting them, so that a problem present in every
// segment is reported once.
func Collapse(findings []Finding) []Finding {
	type key struct{ rule, rep string }
	first := map[key]int{}
	var out []Finding
	for _, f := range findings {
		if f.Location.Segment == "" {
			out = append(out, f)
			continue
		}
		k := key{f.Rule, f.Location.RepID}
		if i, ok := first[k]; ok {
			out[i].Count++
			continue
		}
		f.Count = 1
		first[k] = len(out)
		out = append(out, f)
	}
	return out
}

// New creates a finding for the rule with the default severity.
func (r *Rule) New(loc Location, format string, args ...any) Finding {
	return Finding{
		Rule:     r.ID,
		Severity: r.Severity,
		Message:  strings.ToValidUTF8(fmt.Sprintf(format, args...), "\uFFFD"),
		SpecRef:  r.SpecRef,
		Location: loc,
	}
}

var (
	registryMu sync.Mutex
	registry   = map[string]*Rule{}
)

// Register adds a rule to the global registry and returns it.
// It panics on duplicate IDs, since that is a programming error.
func Register(r Rule) *Rule {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry[r.ID]; ok {
		panic("duplicate rule ID " + r.ID)
	}
	if r.Group == "" {
		r.Group = groupOf(r.ID)
	}
	registry[r.ID] = &r
	return &r
}

// Rules returns all registered rules sorted by group, in the order of
// Groups, and then by ID.
func Rules() []Rule {
	registryMu.Lock()
	defer registryMu.Unlock()
	rules := make([]Rule, 0, len(registry))
	for _, r := range registry {
		rules = append(rules, *r)
	}
	slices.SortFunc(rules, func(a, b Rule) int {
		if c := cmpGroup(a.Group, b.Group); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return rules
}

// cmpGroup orders groups as in Groups, with unknown groups last.
func cmpGroup(a, b string) int {
	ia, ib := slices.Index(Groups, a), slices.Index(Groups, b)
	if ia < 0 {
		ia = len(Groups)
	}
	if ib < 0 {
		ib = len(Groups)
	}
	return ia - ib
}

// Report is the result of checking one MPD.
type Report struct {
	MPDURL      string    `json:"mpdURL"`
	Type        string    `json:"type,omitempty"` // static or dynamic
	Started     time.Time `json:"started"`
	Duration    float64   `json:"durationS"`
	ClockSource string    `json:"clockSource,omitempty"`
	ClockOffset float64   `json:"clockOffsetS,omitempty"` // reference clock minus local clock
	Segments    int       `json:"segmentsChecked"`
	MPDUpdates  int       `json:"mpdUpdatesChecked,omitempty"` // later fetches of a dynamic MPD that were compared
	// ProfileChoice is the choice of profiles, such as "auto" or "dvb,hbbtv",
	// and Profiles the titles of the profiles in @profiles after that choice.
	// In auto mode, CMAF brands and chunked low-latency Representations also
	// turn on the CMAF and low-latency rules.
	ProfileChoice string    `json:"profileChoice,omitempty"`
	Profiles      []string  `json:"profiles,omitempty"`
	Findings      []Finding `json:"findings"`
}

// Counts returns the number of findings per severity.
func (r *Report) Counts() map[Severity]int {
	c := map[Severity]int{}
	for _, f := range r.Findings {
		c[f.Severity]++
	}
	return c
}

// HasErrors reports whether any finding has severity Error.
func (r *Report) HasErrors() bool {
	return slices.ContainsFunc(r.Findings, func(f Finding) bool { return f.Severity == Error })
}
