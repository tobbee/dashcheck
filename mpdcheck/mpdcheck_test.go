package mpdcheck

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/tobbee/dashcheck/report"
)

const liveMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="dynamic" availabilityStartTime="1970-01-01T00:00:00Z"
     publishTime="2026-10-18T10:00:00Z" profiles="urn:mpeg:dash:profile:isoff-live:2011" minBufferTime="PT2S"
     timeShiftBufferDepth="PT1M" minimumUpdatePeriod="PT2S" maxSegmentDuration="PT2S">
  <Period id="p0" start="PT0S">
    <AdaptationSet id="1" contentType="video" mimeType="video/mp4" codecs="avc1.64001e">
      <SegmentTemplate timescale="90000" media="$RepresentationID$/$Time$.m4s" initialization="$RepresentationID$/init.mp4">
        <SegmentTimeline>
          <S t="0" d="180000" r="2"/>
          <S d="180000"/>
        </SegmentTimeline>
      </SegmentTemplate>
      <Representation id="v1" bandwidth="1000000" width="640" height="360"/>
      <Representation id="v2" bandwidth="2000000" width="1280" height="720"/>
    </AdaptationSet>
  </Period>
  <UTCTiming schemeIdUri="urn:mpeg:dash:utc:http-iso:2014" value="https://time.akamai.com/?iso&amp;ms"/>
</MPD>`

func rules(findings []report.Finding) []string {
	var ids []string
	for _, f := range findings {
		ids = append(ids, f.Rule)
	}
	slices.Sort(ids)
	return ids
}

func TestRules(t *testing.T) {
	cases := []struct {
		name     string
		old, new string
		want     []string
	}{
		{"clean", "", "", nil},
		{"no AST", `availabilityStartTime="1970-01-01T00:00:00Z"`, "", []string{"MPD-DYN-AST"}},
		{"no publishTime", `publishTime="2026-10-18T10:00:00Z"`, "", []string{"MPD-DYN-PUBLISHTIME"}},
		{"no TSBD", `timeShiftBufferDepth="PT1M"`, "", []string{"MPD-DYN-TSBD"}},
		{"no UTCTiming", `<UTCTiming schemeIdUri="urn:mpeg:dash:utc:http-iso:2014" value="https://time.akamai.com/?iso&amp;ms"/>`, "", []string{"MPD-DYN-UTCTIMING"}},
		{"ntp UTCTiming", "utc:http-iso:2014", "utc:ntp:2014", []string{"MPD-UTCTIMING-SCHEME"}},
		{"no Period id", `<Period id="p0"`, "<Period", []string{"MPD-PERIOD-ID"}},
		{"duplicate rep id", `id="v2"`, `id="v1"`, []string{"MPD-REP-ID"}},
		{"no mimeType", ` mimeType="video/mp4"`, "", []string{"MPD-MIMETYPE", "MPD-MIMETYPE"}},
		{"no codecs", ` codecs="avc1.64001e"`, "", []string{"MPD-CODECS", "MPD-CODECS"}},
		{"image without codecs", `</Period>`, `<AdaptationSet id="2" contentType="image" mimeType="image/jpeg">
      <SegmentTemplate duration="2" media="$RepresentationID$/$Number$.jpg"/>
      <Representation id="t" bandwidth="1000" width="320" height="180"/>
    </AdaptationSet>
  </Period>`, nil},
		// No MPD-CODECS; the live profile of the test MPD does not allow side-loaded files.
		{"side-loaded text without codecs", `</Period>`, `<AdaptationSet id="2" contentType="text" mimeType="application/ttml+xml" lang="en">
      <Representation id="ttml" bandwidth="1000"><BaseURL>sub_en.ttml</BaseURL></Representation>
    </AdaptationSet>
    <AdaptationSet id="3" contentType="text" mimeType="text/vtt" lang="en">
      <Representation id="vtt" bandwidth="1000"><BaseURL>sub_en.vtt</BaseURL></Representation>
    </AdaptationSet>
  </Period>`, []string{"MPD-PROFILE-SEGINFO", "MPD-PROFILE-SEGINFO"}},
		{"segmented text without codecs", `</Period>`, `<AdaptationSet id="2" contentType="text" mimeType="application/mp4" lang="en">
      <SegmentTemplate duration="2" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="stpp" bandwidth="1000"/>
    </AdaptationSet>
  </Period>`, []string{"MPD-CODECS"}},
		{"contentType mismatch", `contentType="video"`, `contentType="audio"`, []string{"MPD-CONTENTTYPE"}},
		{"shared media", `media="$RepresentationID$/$Time$.m4s"`, `media="$Time$.m4s"`, []string{"MPD-TEMPLATE-REPID"}},
		{"timeline overlap", `<S d="180000"/>`, `<S t="360000" d="180000"/>`, []string{"MPD-TIMELINE-ORDER"}},
		{"timeline gap", `<S d="180000"/>`, `<S t="720000" d="180000"/>`, []string{"MPD-TIMELINE-GAP"}},
		{"negative r", `r="2"`, `r="-1"`, []string{"MPD-TIMELINE-NEGR"}},
		{"long segment", `<S d="180000"/>`, `<S d="270000"/>`, []string{"MPD-MAXSEGDUR"}},
		// @maxSegmentDuration bounds the 4 s Segment Sequence, not its 2 s Partial Segments;
		// the MPD does not signal SSR.
		{"segment sequence", `r="2"/>
          <S d="180000"/>`, `r="2" k="1"/>
          <S d="360000" k="2"/>`, []string{"MPD-MAXSEGDUR", "MPD-SSR-SIGNAL", "MPD-SSR-SIGNAL"}},
		{"duration in months", `timeShiftBufferDepth="PT1M"`, `timeShiftBufferDepth="P1M"`, []string{"MPD-DURATION-YM", "MPD-PARSE"}}, // dash-mpd rejects Y and M
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := liveMPD
			if c.old != "" {
				if !strings.Contains(data, c.old) {
					t.Fatalf("test MPD lacks %q", c.old)
				}
				data = strings.Replace(data, c.old, c.new, 1)
			}
			md, findings := Check([]byte(data), "https://example.com/live.mpd")
			if md == nil && !slices.Contains(c.want, "MPD-PARSE") {
				t.Fatalf("MPD not parsed: %v", findings)
			}
			if got := rules(findings); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v\n%v", got, c.want, findings)
			}
		})
	}
}

func TestMalformed(t *testing.T) {
	data, err := os.ReadFile("../testdata/malformed.mpd")
	if err != nil {
		t.Fatal(err)
	}
	_, findings := Check(data, "malformed.mpd")
	if len(findings) == 0 || findings[0].Rule != "MPD-XML" {
		t.Errorf("got %v", findings)
	}
}
