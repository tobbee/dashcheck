package spec

import (
	"slices"
	"testing"

	m "github.com/Eyevinn/dash-mpd/mpd"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in, want string // want is String() of the result, or "error"
		only     bool
	}{
		{"", "auto", false},
		{"auto", "auto", false},
		{"dvb", "dvb", true},
		{" dvb , dvb ", "dvb", true},
		{"auto,dvb2017,hbbtv", "auto,dvb2017,hbbtv", false},
		{"dvb,auto", "auto,dvb", false},
		{"none", "none", true},
		{"dvb3000", "error", false},
		{"none,dvb", "error", false},
		{"auto,none", "error", false},
	}
	for _, c := range cases {
		sel, err := Parse(c.in)
		if c.want == "error" {
			if err == nil {
				t.Errorf("Parse(%q) = %v, want an error", c.in, sel)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q): %v", c.in, err)
			continue
		}
		if got := sel.String(); got != c.want || sel.Only != c.only {
			t.Errorf("Parse(%q) = %q, only %v; want %q, only %v", c.in, got, sel.Only, c.want, c.only)
		}
	}
}

const testMPD = `<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static" mediaPresentationDuration="PT8S" minBufferTime="PT2S"
     profiles="urn:mpeg:dash:profile:isoff-live:2011,http://dashif.org/guidelines/dash-if-simple">
  <Period id="p0">
    <AdaptationSet id="1" profiles="urn:mpeg:dash:profile:isoff-live:2011,http://dashif.org/guidelines/dash-if-simple">
      <Representation id="v1" bandwidth="1000000"/>
    </AdaptationSet>
    <AdaptationSet id="2">
      <Representation id="a1" bandwidth="128000" profiles="http://dashif.org/guidelines/dash-if-simple"/>
    </AdaptationSet>
  </Period>
</MPD>`

func TestApply(t *testing.T) {
	const (
		live = "urn:mpeg:dash:profile:isoff-live:2011"
		iop  = "http://dashif.org/guidelines/dash-if-simple"
		dvb  = "urn:dvb:dash:profile:dvb-dash:2014"
	)
	cases := []struct {
		choice        string
		mpd, as, rep  string // @profiles of the MPD, the first AdaptationSet and the audio Representation
		wantUnlisted  []string
		wantSignalled []string
	}{
		{"auto", live + "," + iop, live + "," + iop, iop, nil, []string{"iop"}},
		{"auto,dvb", live + "," + iop + "," + dvb, live + "," + iop + "," + dvb, iop + "," + dvb,
			[]string{"dvb"}, []string{"iop", "dvb"}},
		{"dvb", live + "," + dvb, live + "," + dvb, dvb, []string{"dvb"}, []string{"dvb"}},
		{"iop", live + "," + iop, live + "," + iop, iop, nil, []string{"iop"}},
		{"none", live, live, "", nil, nil},
	}
	for _, c := range cases {
		t.Run(c.choice, func(t *testing.T) {
			md, err := m.ReadFromString(testMPD)
			if err != nil {
				t.Fatal(err)
			}
			sel, err := Parse(c.choice)
			if err != nil {
				t.Fatal(err)
			}
			unlisted := names(sel.Apply(md))
			p := md.Periods[0]
			if got := string(md.Profiles); got != c.mpd {
				t.Errorf("MPD@profiles %q, want %q", got, c.mpd)
			}
			if got := string(p.AdaptationSets[0].Profiles); got != c.as {
				t.Errorf("AdaptationSet@profiles %q, want %q", got, c.as)
			}
			if got := string(p.AdaptationSets[1].Profiles); got != "" {
				t.Errorf("unset AdaptationSet@profiles became %q", got)
			}
			if got := string(p.AdaptationSets[1].Representations[0].Profiles); got != c.rep {
				t.Errorf("Representation@profiles %q, want %q", got, c.rep)
			}
			if !slices.Equal(unlisted, c.wantUnlisted) {
				t.Errorf("unlisted %v, want %v", unlisted, c.wantUnlisted)
			}
			if got := names(Signalled(md)); !slices.Equal(got, c.wantSignalled) {
				t.Errorf("signalled %v, want %v", got, c.wantSignalled)
			}
		})
	}
}

func names(ps []Profile) []string {
	var out []string
	for _, p := range ps {
		out = append(out, p.Name)
	}
	return out
}
