package mpdcheck

import (
	"slices"
	"strings"
	"testing"
)

// tiledMPD has two video tiles of one source with spatial relationship descriptors.
const tiledMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static" mediaPresentationDuration="PT8S"
     profiles="urn:mpeg:dash:profile:isoff-live:2011" minBufferTime="PT2S">
  <Period id="p0">
    <AdaptationSet id="1" contentType="video" mimeType="video/mp4" codecs="avc1.64001e">
      <SupplementalProperty schemeIdUri="urn:mpeg:dash:srd:2014" value="0,0,0,1920,1080,3840,1080"/>
      <SegmentTemplate duration="2" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="left" bandwidth="1000000" width="1920" height="1080"/>
    </AdaptationSet>
    <AdaptationSet id="2" contentType="video" mimeType="video/mp4" codecs="avc1.64001e">
      <EssentialProperty schemeIdUri="urn:mpeg:dash:srd:2014" value="0,1920,0,1920,1080"/>
      <FramePacking schemeIdUri="urn:mpeg:dash:14496:10:frame_packing_arrangement_type:2011" value="3"/>
      <SegmentTemplate duration="2" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="right" bandwidth="1000000" width="1920" height="1080"/>
    </AdaptationSet>
  </Period>
</MPD>`

func TestDescriptorRules(t *testing.T) {
	cases := []struct {
		name     string
		old, new string
		want     []string
	}{
		{"clean", "", "", nil},
		{"srd without value", `value="0,1920,0,1920,1080"`, ``, []string{"MPD-SRD-VALUE"}},
		{"srd with six values", `value="0,1920,0,1920,1080"`, `value="0,1920,0,1920,1080,3840"`, []string{"MPD-SRD-VALUE"}},
		{"srd not integers", `value="0,1920,0,1920,1080"`, `value="0,a,0,1920,1080"`, []string{"MPD-SRD-VALUE"}},
		{"srd on representation", `<Representation id="right" bandwidth="1000000" width="1920" height="1080"/>`,
			`<Representation id="right" bandwidth="1000000" width="1920" height="1080"><SupplementalProperty schemeIdUri="urn:mpeg:dash:srd:2014" value="1,0,0,10,10,10,10"/></Representation>`,
			[]string{"MPD-SRD-PLACEMENT"}},
		{"all essential", `<SupplementalProperty schemeIdUri="urn:mpeg:dash:srd:2014"`, `<EssentialProperty schemeIdUri="urn:mpeg:dash:srd:2014"`,
			[]string{"MPD-SRD-ESSENTIAL"}},
		{"no total size", `value="0,0,0,1920,1080,3840,1080"`, `value="0,0,0,1920,1080"`, []string{"MPD-SRD-TOTAL"}},
		{"two total sizes", `value="0,1920,0,1920,1080"`, `value="0,1920,0,1920,1080"/><SupplementalProperty schemeIdUri="urn:mpeg:dash:srd:2014" value="0,0,0,1920,1080,1920,1080"`,
			[]string{"MPD-SRD-BOUNDS", "MPD-SRD-TOTAL"}},
		{"outside total width", `value="0,1920,0,1920,1080"`, `value="0,2000,0,1920,1080"`, []string{"MPD-SRD-BOUNDS"}},
		{"frame packing for hevc", `codecs="avc1.64001e">
      <EssentialProperty`, `codecs="hvc1.1.6.L93.B0">
      <EssentialProperty`, []string{"MPD-FRAMEPACKING"}},
		{"frame packing value", `value="3"`, `value="7"`, []string{"MPD-FRAMEPACKING"}},
		{"frame packing scheme", `urn:mpeg:dash:14496:10:frame_packing_arrangement_type:2011`, `urn:example:fp`, []string{"MPD-FRAMEPACKING"}},
		{"channel positions", `<FramePacking`, `<AudioChannelConfiguration schemeIdUri="urn:mpeg:dash:outputChannelPositionList:2012" value="2"/><FramePacking`,
			[]string{"MPD-CHANNEL-POSITIONS"}},
		{"hbbtv", `isoff-live:2011"`, `isoff-live:2011,urn:hbbtv:dash:profile:isoff-live:2012"`, nil},
		{"hbbtv subsegment alignment", `isoff-live:2011" minBufferTime="PT2S">
  <Period id="p0">
    <AdaptationSet id="1"`, `isoff-live:2011,urn:hbbtv:dash:profile:isoff-live:2012" minBufferTime="PT2S">
  <Period id="p0">
    <AdaptationSet id="1" subsegmentAlignment="true"`, []string{"MPD-HBBTV"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := tiledMPD
			if c.old != "" {
				if !strings.Contains(data, c.old) {
					t.Fatalf("test MPD lacks %q", c.old)
				}
				data = strings.Replace(data, c.old, c.new, 1)
			}
			md, findings := Check([]byte(data), "https://example.com/tiles.mpd")
			if md == nil {
				t.Fatalf("MPD not parsed: %v", findings)
			}
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
