package mpdcheck

import (
	"slices"
	"strings"
	"testing"
)

const llMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="dynamic" availabilityStartTime="1970-01-01T00:00:00Z"
     publishTime="2026-10-18T10:00:00Z" minimumUpdatePeriod="PT2S" timeShiftBufferDepth="PT30S" minBufferTime="PT1S"
     profiles="urn:mpeg:dash:profile:isoff-live:2011,http://www.dashif.org/guidelines/low-latency-live-v5">
  <ServiceDescription id="0">
    <Latency referenceId="0" target="3000" min="2000" max="6000"/>
    <PlaybackRate min="0.96" max="1.04"/>
  </ServiceDescription>
  <Period id="p0" start="PT0S">
    <AdaptationSet id="1" contentType="video" mimeType="video/mp4" codecs="avc1.64001e" segmentAlignment="true">
      <InbandEventStream schemeIdUri="urn:mpeg:dash:event:2012" value="1"/>
      <ProducerReferenceTime id="0" type="encoder" wallClockTime="1970-01-01T00:00:00Z" presentationTime="0">
        <UTCTiming schemeIdUri="urn:mpeg:dash:utc:http-iso:2014" value="https://time.akamai.com/?iso&amp;ms"/>
      </ProducerReferenceTime>
      <Resync type="2" dT="1000000" marker="true"/>
      <SegmentTemplate timescale="1000000" duration="2000000" availabilityTimeOffset="1.5" availabilityTimeComplete="false"
                       media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="v1" bandwidth="1000000" qualityRanking="2"/>
      <Representation id="v2" bandwidth="2000000" qualityRanking="1"/>
    </AdaptationSet>
  </Period>
  <UTCTiming schemeIdUri="urn:mpeg:dash:utc:http-iso:2014" value="https://time.akamai.com/?iso&amp;ms"/>
  <LeapSecondInformation availabilityStartLeapOffset="37"/>
</MPD>`

func TestLowLatencyRules(t *testing.T) {
	cases := []struct {
		name     string
		old, new string
		want     []string
	}{
		{"clean", "", "", nil},
		{"no profile", `,http://www.dashif.org/guidelines/low-latency-live-v5`, "", []string{"MPD-LL-PROFILE"}},
		{"no leap seconds", `<LeapSecondInformation availabilityStartLeapOffset="37"/>`, "", []string{"MPD-LL-LEAPSECOND"}},
		{"no service description", `<Latency referenceId="0" target="3000" min="2000" max="6000"/>`, "", []string{"MPD-LL-SERVICEDESC"}},
		{"no min max", ` min="2000" max="6000"`, "", []string{"MPD-LL-SERVICEDESC-OPTIONAL"}},
		{"head utc only", `utc:http-iso:2014" value="https://time.akamai.com/?iso&amp;ms"/>
  <Leap`, `utc:http-head:2014" value="https://time.akamai.com/?iso&amp;ms"/>
  <Leap`, []string{"MPD-LL-UTCTIMING"}},
		{"inband value", `value="1"/>`, `value="2"/>`, []string{"MPD-LL-INBAND-VALUE", "MPD-LL-INBAND-VALUE"}},
		{"bad prt type", `type="encoder"`, `type="application"`, []string{"MPD-LL-PRT", "MPD-LL-PRT"}},
		{"same prt id, same reference", `</ProducerReferenceTime>`, `</ProducerReferenceTime>
      <ProducerReferenceTime id="0" wallClockTime="1970-01-01T00:00:00Z" presentationTime="0">
        <UTCTiming schemeIdUri="urn:mpeg:dash:utc:http-iso:2014" value="https://time.akamai.com/?iso&amp;ms"/>
      </ProducerReferenceTime>`, nil},
		{"same prt id, other reference", `</ProducerReferenceTime>`, `</ProducerReferenceTime>
      <ProducerReferenceTime id="0" type="encoder" wallClockTime="1970-01-01T00:00:01Z" presentationTime="0">
        <UTCTiming schemeIdUri="urn:mpeg:dash:utc:http-iso:2014" value="https://time.akamai.com/?iso&amp;ms"/>
      </ProducerReferenceTime>`, []string{"MPD-LL-PRT"}},
		{"chunked without ato", ` availabilityTimeOffset="1.5"`, "", []string{"MPD-LL-CHUNKED", "MPD-LL-CHUNKED"}},
		{"ato without atc", ` availabilityTimeComplete="false"`, "", []string{"MPD-LL-CHUNKED", "MPD-LL-CHUNKED"}},
		{"no alignment", ` segmentAlignment="true"`, "", []string{"MPD-LL-ALIGNMENT"}},
		{"resync marker", `marker="true"`, `marker="false"`, []string{"MPD-LL-RESYNC-VALUE", "MPD-LL-RESYNC-VALUE"}},
		{"resync type 3 on lowest", `type="2" dT`, `type="3" dT`, []string{"MPD-LL-RESYNC-VALUE"}},
		{"resync dT above target", `dT="1000000"`, `dT="4000000"`, []string{"MPD-LL-RESYNC-VALUE", "MPD-LL-RESYNC-VALUE"}},
		{"resync dT above half target", `dT="1000000"`, `dT="2000000"`, []string{"MPD-LL-RESYNC", "MPD-LL-RESYNC"}},
		{"bad mimeType", `mimeType="video/mp4"`, `mimeType="video/mp2t"`, []string{"MPD-LL-MIMETYPE", "MPD-LL-MIMETYPE"}},
		{"segment mode", ` availabilityTimeOffset="1.5" availabilityTimeComplete="false"`, "", nil},
		{"addressable chunks without k", `media="$RepresentationID$/$Number$.m4s"`, `media="$RepresentationID$/$Number$_$SubNumber$.m4s"`, []string{"MPD-LL-K"}},
		{"addressable chunks with k", `media="$RepresentationID$/$Number$.m4s"`, `k="8" media="$RepresentationID$/$Number$_$SubNumber$.m4s"`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := llMPD
			if c.old != "" {
				if !strings.Contains(data, c.old) {
					t.Fatalf("test MPD lacks %q", c.old)
				}
				data = strings.Replace(data, c.old, c.new, 1)
			}
			md, findings := Check([]byte(data), "https://example.com/ll.mpd")
			if md == nil {
				t.Fatalf("MPD not parsed: %v", findings)
			}
			var got []string
			for _, f := range findings {
				if strings.HasPrefix(f.Rule, "MPD-LL-") {
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
