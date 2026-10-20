package mpdcheck

import (
	"slices"
	"strings"
	"testing"
)

const ssrMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static" mediaPresentationDuration="PT8S" minBufferTime="PT2S"
     profiles="urn:mpeg:dash:profile:isoff-live:2011">
  <Period id="p0">
    <AdaptationSet id="1" contentType="video" mimeType="video/mp4" codecs="avc1.64001e" segmentAlignment="true" startWithSAP="1">
      <SegmentTemplate media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4" duration="2000" timescale="1000"/>
      <Representation id="v1" bandwidth="300000" width="640" height="360"/>
    </AdaptationSet>
    <AdaptationSet id="2" contentType="video" mimeType="video/mp4" codecs="avc1.64001e" segmentAlignment="true" startWithSAP="1">
      <EssentialProperty schemeIdUri="urn:mpeg:dash:ssr:2023" value="1"/>
      <SegmentSequenceProperties sapType="1" cadence="1" alignment="v1"/>
      <SegmentTemplate media="$RepresentationID$/$Number$_$SubNumber$.m4s" initialization="$RepresentationID$/init.mp4" duration="2000" k="4" timescale="1000"/>
      <Representation id="ld1" bandwidth="300000" width="640" height="360"/>
    </AdaptationSet>
  </Period>
</MPD>`

func TestSSRRules(t *testing.T) {
	const timeline = `<SegmentTemplate media="$RepresentationID$/$Time$_$SubNumber$.m4s" initialization="$RepresentationID$/init.mp4" timescale="1000">
        <SegmentTimeline><S t="0" d="2000" r="2" k="4"/><S d="2000" k="4"/></SegmentTimeline></SegmentTemplate>`
	ssrTemplate := `<SegmentTemplate media="$RepresentationID$/$Number$_$SubNumber$.m4s" initialization="$RepresentationID$/init.mp4" duration="2000" k="4" timescale="1000"/>`
	essential := `<EssentialProperty schemeIdUri="urn:mpeg:dash:ssr:2023" value="1"/>`
	cases := []struct {
		name     string
		old, new string
		want     []string
	}{
		{"clean", "", "", nil},
		{"no EssentialProperty", essential, "", []string{"MPD-SSR-SIGNAL"}},
		{"advanced linear profile", essential + "\n", "", nil},
		{"timeline", ssrTemplate, timeline, nil},
		{"timeline with and without k", ssrTemplate, strings.Replace(timeline, `<S d="2000" k="4"/>`, `<S d="2000"/>`, 1), []string{"MPD-SSR-TIMELINE"}},
		{"k without d", ssrTemplate, strings.Replace(timeline, `<S d="2000" k="4"/>`, `<S k="4"/>`, 1), []string{"MPD-SSR-ATTRIBUTES"}},
		{"k 0", `k="4"`, `k="0"`, []string{"MPD-SSR-ATTRIBUTES"}},
		{"endSubNumber", `k="4"`, `k="4" endNumber="3" endSubNumber="1"`, nil},
		{"endSubNumber without k", `k="4"`, `endNumber="3" endSubNumber="1"`, []string{"MPD-SSR-ATTRIBUTES"}},
		{"endSubNumber with timeline", ssrTemplate, strings.Replace(timeline, `timescale="1000"`, `timescale="1000" endSubNumber="1"`, 1), []string{"MPD-SSR-ATTRIBUTES"}},
		{"cadence 0", `cadence="1"`, `cadence="0"`, []string{"MPD-SSR-PROPERTIES"}},
		{"alignment to unknown Representation", `alignment="v1"`, `alignment="v1 v9"`, []string{"MPD-SSR-PROPERTIES"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := ssrMPD
			if c.old != "" {
				if !strings.Contains(data, c.old) {
					t.Fatalf("test MPD lacks %q", c.old)
				}
				data = strings.Replace(data, c.old, c.new, 1)
			}
			if c.name == "advanced linear profile" {
				data = strings.Replace(data, `isoff-live:2011"`, `isoff-live:2011,urn:mpeg:dash:profile:advanced-linear:2025"`, 1)
			}
			_, findings := Check([]byte(data), "https://example.com/ssr.mpd")
			var got []string
			for _, f := range findings {
				if strings.HasPrefix(f.Rule, "MPD-SSR") || f.Rule == "MPD-XSD" {
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
