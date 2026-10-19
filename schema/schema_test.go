package schema

import (
	"os"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	valid, err := os.ReadFile("../testdata/testpic_2s/Manifest.mpd")
	if err != nil {
		t.Fatal(err)
	}
	malformed, err := os.ReadFile("../testdata/malformed.mpd")
	if err != nil {
		t.Fatal(err)
	}
	badAttrs := strings.Replace(string(valid), `minBufferTime="PT2S"`, `minBufferTime="2s"`, 1)
	badAttrs = strings.Replace(badAttrs, `<Period id="one"`, `<Period foo="bar" id="one"`, 1)

	cases := []struct {
		name      string
		data      []byte
		wantRules []string
		wantLine  int
	}{
		{"valid", valid, nil, 0},
		{"malformed", malformed, []string{"MPD-XML"}, 2},
		{"bad attributes", []byte(badAttrs), []string{"MPD-XSD", "MPD-XSD"}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			findings, err := Validate(c.data, "test.mpd")
			if err != nil {
				t.Fatal(err)
			}
			if len(findings) != len(c.wantRules) {
				t.Fatalf("got %d findings, want %d: %v", len(findings), len(c.wantRules), findings)
			}
			for i, f := range findings {
				if f.Rule != c.wantRules[i] {
					t.Errorf("finding %d: rule %s, want %s", i, f.Rule, c.wantRules[i])
				}
			}
			if len(findings) > 0 && findings[0].Location.Line != c.wantLine {
				t.Errorf("first finding on line %d, want %d", findings[0].Location.Line, c.wantLine)
			}
		})
	}
}

// TestMutations checks schema errors in variants of a valid MPD: the message
// and the element path of the first error, or that a variant is valid.
func TestMutations(t *testing.T) {
	data, err := os.ReadFile("../testdata/testpic_2s/Manifest.mpd")
	if err != nil {
		t.Fatal(err)
	}
	valid := string(data)
	const (
		period = `<Period id="one" start="PT0S">`
		xlink  = `xmlns:xlink="http://www.w3.org/1999/xlink"`
		remote = `<Period id="remote" xlink:href="https://example.com/period.xml" ` + xlink
	)
	cases := []struct {
		name     string
		old, new string
		wantMsg  string // "" for a valid MPD
		wantPath string
		// knownGap marks an error that jacoelho/xsd misses (jacoelho/xsd#137, see CLAUDE.md).
		knownGap bool
	}{
		{name: "unknown element", old: period, new: period + `<Foo/>`,
			wantMsg: "unexpected child element {urn:mpeg:dash:schema:mpd:2011}Foo", wantPath: "/MPD/Period"},
		{name: "element order", old: `</Period>`, new: `<BaseURL>a/</BaseURL></Period>`,
			wantMsg: "unexpected child element BaseURL", wantPath: "/MPD/Period"},
		{name: "MPD child order", old: `<ProgramInformation>`,
			new:     `<ServiceDescription id="0"><Latency target="3000"/></ServiceDescription><ProgramInformation>`,
			wantMsg: "unexpected child element ProgramInformation", wantPath: "/MPD"},
		{name: "other namespace before DASH elements", old: period, new: period + `<x:Foo xmlns:x="urn:example"/>`,
			wantMsg: "unexpected child element AdaptationSet", wantPath: "/MPD/Period"},
		{name: "not a decimal", old: `bandwidth="48000"`, new: `bandwidth="abc"`,
			wantMsg: "invalid attribute bandwidth: invalid decimal", wantPath: "/MPD/Period/AdaptationSet/Representation"},
		{name: "not a duration", old: `mediaPresentationDuration="PT8S"`, new: `mediaPresentationDuration="8S"`,
			wantMsg: "invalid attribute mediaPresentationDuration: invalid duration", wantPath: "/MPD"},
		{name: "not a boolean", old: `segmentAlignment="true"`, new: `segmentAlignment="maybe"`,
			wantMsg: "invalid attribute segmentAlignment: invalid boolean", wantPath: "/MPD/Period/AdaptationSet"},
		{name: "not in the enumeration", old: `type="static"`, new: `type="live"`,
			wantMsg: "invalid attribute type: enumeration facet failed", wantPath: "/MPD"},
		{name: "above the maximum", old: `startWithSAP="1"`, new: `startWithSAP="9"`,
			wantMsg: "invalid attribute startWithSAP: upper bound facet failed", wantPath: "/MPD/Period/AdaptationSet"},
		{name: "no Representation@id", old: `<Representation id="A48" `, new: `<Representation `,
			wantMsg: "missing required attribute id", wantPath: "/MPD/Period/AdaptationSet/Representation"},
		{name: "no Representation@bandwidth", old: ` bandwidth="48000"`, new: ``,
			wantMsg: "missing required attribute bandwidth", wantPath: "/MPD/Period/AdaptationSet/Representation"},
		{name: "undeclared attribute", old: ` id="base"`, new: ` id="base" foo="bar"`,
			wantMsg: "attribute is not declared: foo", wantPath: "/MPD"},
		{name: "no DASH namespace", old: ` xmlns="urn:mpeg:dash:schema:mpd:2011"`, new: ``,
			wantMsg: "root element is not declared: MPD", wantPath: ""},
		{name: "xlink:type not simple", old: period, new: remote + ` xlink:type="extended"/>` + period,
			wantMsg: "fixed attribute mismatch type", wantPath: "/MPD/Period"},
		{name: "xlink:show not embed", old: period, new: remote + ` xlink:show="new"/>` + period,
			wantMsg: "fixed attribute mismatch show", wantPath: "/MPD/Period"},
		{name: "xlink:actuate not in the enumeration", old: period, new: remote + ` xlink:actuate="bogus"/>` + period,
			wantMsg: "invalid attribute actuate", wantPath: "/MPD/Period", knownGap: true},

		{name: "valid", old: ``, new: ``},
		{name: "other namespace after DASH elements", old: `</Period>`, new: `<x:Foo xmlns:x="urn:example"/></Period>`},
		{name: "attribute in other namespace", old: ` id="base"`, new: ` id="base" xmlns:x="urn:example" x:foo="1"`},
		{name: "DVB BaseURL attributes", old: period,
			new: period + `<BaseURL xmlns:dvb="urn:dvb:dash-extensions:2014-1" dvb:priority="1" dvb:weight="1">a/</BaseURL>`},
		{name: "remote Period", old: period, new: remote + ` xlink:actuate="onLoad"/>` + period},
		{name: "ServiceDescription", old: period,
			new: `<ServiceDescription id="0"><Latency target="3000"/></ServiceDescription>` + period},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(valid, c.old) {
				t.Fatalf("test MPD lacks %q", c.old)
			}
			findings, err := Validate([]byte(strings.Replace(valid, c.old, c.new, 1)), "test.mpd")
			if err != nil {
				t.Fatal(err)
			}
			if c.knownGap {
				if len(findings) > 0 {
					t.Fatalf("jacoelho/xsd reports this now (%v): remove knownGap and the note in CLAUDE.md", findings)
				}
				t.Skip("known gap of jacoelho/xsd, see CLAUDE.md")
			}
			if c.wantMsg == "" {
				if len(findings) > 0 {
					t.Errorf("valid MPD: %v", findings)
				}
				return
			}
			if len(findings) == 0 {
				t.Fatalf("no error, want %q", c.wantMsg)
			}
			f := findings[0]
			if f.Rule != "MPD-XSD" || !strings.Contains(f.Message, c.wantMsg) || f.Location.Path != c.wantPath {
				t.Errorf("got %s %q at %q, want MPD-XSD %q at %q", f.Rule, f.Message, f.Location.Path, c.wantMsg, c.wantPath)
			}
		})
	}
}
