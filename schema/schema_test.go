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
