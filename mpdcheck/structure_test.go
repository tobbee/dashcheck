package mpdcheck

import (
	"slices"
	"strings"
	"testing"
)

const staticMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" xmlns:cenc="urn:mpeg:cenc:2013" type="static"
     mediaPresentationDuration="PT8S" profiles="urn:mpeg:dash:profile:isoff-live:2011" minBufferTime="PT2S">
  <ProgramInformation lang="en"><Title>Test</Title></ProgramInformation>
  <Period id="p0">
    <AdaptationSet id="1" contentType="video" mimeType="video/mp4" codecs="avc1.64001e"
                   minWidth="640" maxWidth="1280" minHeight="360" maxHeight="720" maxFrameRate="30" maxBandwidth="3000000">
      <ContentProtection schemeIdUri="urn:mpeg:dash:mp4protection:2011" value="cenc" cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"/>
      <Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>
      <SegmentTemplate timescale="90000" duration="180000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="v1" bandwidth="1000000" width="640" height="360" frameRate="30"/>
      <Representation id="v2" bandwidth="2000000" width="1280" height="720" frameRate="30"/>
    </AdaptationSet>
  </Period>
</MPD>`

// The mp4protection descriptor of the AdaptationSet gets @value and
// cenc:default_KID from a source in the Period (ISO/IEC 23009-1 5.8.4.1.3).
func TestContentProtectionRef(t *testing.T) {
	ref := strings.NewReplacer(`<Period id="p0">`, `<Period id="p0">
    <ContentProtection schemeIdUri="urn:mpeg:dash:mp4protection:2011" value="cenc" refId="cp" cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"/>`,
		` value="cenc" cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"/>`, ` ref="cp"/>`)
	for name, mpd := range map[string]string{"static": staticMPD, "IOP": iopMPD} {
		data := ref.Replace(mpd)
		if !strings.Contains(data, `ref="cp"/>`) {
			t.Fatalf("%s: no referencing ContentProtection", name)
		}
		if _, findings := Check([]byte(data), "https://example.com/vod.mpd"); len(findings) > 0 {
			t.Errorf("%s: %v", name, findings)
		}
	}
}

func TestStructureRules(t *testing.T) {
	cases := []struct {
		name     string
		old, new string
		want     []string
	}{
		{"clean", "", "", nil},
		{"no duration", ` mediaPresentationDuration="PT8S"`, "", []string{"MPD-DURATION-MISSING"}},
		{"unknown profile", `isoff-live:2011"`, `isoff-live:2011,urn:example:profile"`, []string{"MPD-PROFILE-UNKNOWN"}},
		{"two ProgramInformation", `<ProgramInformation lang="en"><Title>Test</Title></ProgramInformation>`,
			`<ProgramInformation lang="en"/><ProgramInformation/>`, []string{"MPD-PROGINFO-LANG"}},
		{"bitstreamSwitching", `<Period id="p0">`, `<Period id="p0" bitstreamSwitching="true">`, nil},
		{"bitstreamSwitching conflict", `<Period id="p0">
    <AdaptationSet id="1"`, `<Period id="p0" bitstreamSwitching="true">
    <AdaptationSet id="1" bitstreamSwitching="false"`, []string{"MPD-BITSTREAMSWITCHING"}},
		{"two segment infos", `<Representation id="v2" bandwidth="2000000" width="1280" height="720" frameRate="30"/>`,
			`<Representation id="v2" bandwidth="2000000" width="1280" height="720" frameRate="30"><SegmentBase indexRange="0-100"/><SegmentList duration="2"/></Representation>`,
			[]string{"MPD-SEGINFO-MULTIPLE"}},
		{"init with Number", `initialization="$RepresentationID$/init.mp4"`, `initialization="$RepresentationID$/init-$Number$.mp4"`,
			[]string{"MPD-TEMPLATE-INIT"}},
		{"min above max", `minWidth="640"`, `minWidth="1920"`, []string{"MPD-AS-MINMAX", "MPD-AS-RANGE", "MPD-AS-RANGE"}},
		{"width above max", `width="1280"`, `width="1920"`, []string{"MPD-AS-RANGE"}},
		{"bandwidth above max", `bandwidth="2000000"`, `bandwidth="4000000"`, []string{"MPD-AS-RANGE"}},
		{"frame rate above max", `height="720" frameRate="30"`, `height="720" frameRate="60"`, []string{"MPD-AS-RANGE"}},
		{"common attribute on both", `<Representation id="v1" bandwidth`, `<Representation id="v1" codecs="avc1.64001e" bandwidth`,
			[]string{"MPD-COMMON-BOTH"}},
		{"no Representation", `<Representation id="v1" bandwidth="1000000" width="640" height="360" frameRate="30"/>
      <Representation id="v2" bandwidth="2000000" width="1280" height="720" frameRate="30"/>`, "",
			[]string{"MPD-AS-NOREP"}},
		{"mp4protection value", `value="cenc"`, `value="cenc-ctr"`, []string{"MPD-CP-VALUE"}},
		{"no default_KID", ` cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"`, "", []string{"MPD-CP-DEFAULTKID"}},
		{"no default_KID for another scheme", ` value="cenc" cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"`, ` value="abcd"`, nil},
		{"role value", `value="main"`, `value="primary"`, []string{"MPD-ROLE-VALUE"}},
		{"newer role value", `value="main"`, `value="forced-subtitle"`, nil},
		{"event stream actuate", `</Period>`, `<EventStream schemeIdUri="urn:x" xmlns:xlink="http://www.w3.org/1999/xlink" xlink:actuate="onLoad"/></Period>`,
			[]string{"MPD-EVENTSTREAM"}},
		{"remote period", `<Period id="p0">`, `<Period id="p1" duration="PT4S" xmlns:xlink="http://www.w3.org/1999/xlink" xlink:href="https://example.com/p1.xml" xlink:actuate="onRequest"/>
  <Period id="p0">`, []string{"MPD-XLINK-UNRESOLVED"}},
		{"fallback not on MPD", `<Role schemeIdUri`, `<SupplementalProperty schemeIdUri="urn:mpeg:dash:fallback:2016" value="https://b/m.mpd"/><Role schemeIdUri`,
			[]string{"MPD-CHAINING"}},
		{"zero duration period", `<Period id="p0">`, `<Period id="p0" duration="PT0S"><AdaptationSet id="9" mimeType="audio/mp4" codecs="mp4a.40.2"><SegmentTemplate duration="2" media="a/$Number$.m4s"/><Representation id="a" bandwidth="1"/></AdaptationSet>`,
			[]string{"MPD-PERIOD-ZERO"}},
		{"subrep without bandwidth", `frameRate="30"/>
    </AdaptationSet>`, `frameRate="30"><SubRepresentation level="1"/></Representation>
    </AdaptationSet>`, []string{"MPD-SUBREP-BANDWIDTH"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := staticMPD
			if c.old != "" {
				if !strings.Contains(data, c.old) {
					t.Fatalf("test MPD lacks %q", c.old)
				}
				data = strings.Replace(data, c.old, c.new, 1)
			}
			md, findings := Check([]byte(data), "https://example.com/vod.mpd")
			if md == nil {
				t.Fatalf("MPD not parsed: %v", findings)
			}
			// Schema findings are tested elsewhere; some mutations are schema-invalid on purpose.
			var got []string
			for _, f := range findings {
				if f.Rule != "MPD-XSD" && !strings.HasPrefix(f.Rule, "MPD-DVB-XSD") {
					got = append(got, f.Rule)
				}
			}
			slices.Sort(got)
			if !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v\n%v", got, c.want, findings)
			}
		})
	}
}
