package mpdcheck

import (
	"slices"
	"strings"
	"testing"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
)

const updateMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" id="live" type="dynamic" availabilityStartTime="1970-01-01T00:00:00Z"
     publishTime="1970-01-01T00:01:00Z" minimumUpdatePeriod="PT2S" timeShiftBufferDepth="PT30S" maxSegmentDuration="PT2S"
     minBufferTime="PT2S" profiles="urn:dvb:dash:profile:dvb-dash:2014,urn:dvb:dash:profile:dvb-dash:isoff-ext-live:2014">
  <Period id="p0" start="PT0S">
    <AssetIdentifier schemeIdUri="urn:example:asset" value="a1"/>
    <AdaptationSet id="1" contentType="video" mimeType="video/mp4" codecs="avc3.64001f" segmentAlignment="true" startWithSAP="1">
      <Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>
      <SegmentTemplate timescale="1000" media="$RepresentationID$/$Time$.m4s" initialization="$RepresentationID$/init.mp4">
        <SegmentTimeline><S t="0" d="2000" r="29"/></SegmentTimeline>
      </SegmentTemplate>
      <Representation id="v1" bandwidth="1000000" width="1280" height="720"/>
      <Representation id="v2" bandwidth="2000000" width="1920" height="1080"/>
    </AdaptationSet>
    <AdaptationSet id="2" contentType="audio" mimeType="audio/mp4" codecs="mp4a.40.2" lang="en" segmentAlignment="true" startWithSAP="1">
      <AudioChannelConfiguration schemeIdUri="urn:mpeg:dash:23003:3:audio_channel_configuration:2011" value="2"/>
      <Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>
      <SegmentTemplate timescale="1000" media="$RepresentationID$/$Time$.m4s" initialization="$RepresentationID$/init.mp4">
        <SegmentTimeline><S t="0" d="2000" r="29"/></SegmentTimeline>
      </SegmentTemplate>
      <Representation id="a1" bandwidth="128000" audioSamplingRate="48000"/>
    </AdaptationSet>
  </Period>
</MPD>`

func TestCheckUpdate(t *testing.T) {
	// The next version, fetched 2s later, adds one segment.
	next := strings.ReplaceAll(updateMPD, `r="29"`, `r="30"`)
	next = strings.Replace(next, `publishTime="1970-01-01T00:01:00Z"`, `publishTime="1970-01-01T00:01:02Z"`, 1)
	video := `<Representation id="v2" bandwidth="2000000" width="1920" height="1080"/>`
	cases := []struct {
		name     string
		old, new string
		want     []string
	}{
		{"clean", "", "", nil},
		{"bandwidth may change", `bandwidth="2000000"`, `bandwidth="2500000"`, nil},
		{"same publishTime", `publishTime="1970-01-01T00:01:02Z"`, `publishTime="1970-01-01T00:01:00Z"`, []string{"MPD-UPDATE-PUBLISHTIME"}},
		{"earlier publishTime", `publishTime="1970-01-01T00:01:02Z"`, `publishTime="1970-01-01T00:00:58Z"`, []string{"MPD-UPDATE-PUBLISHTIME"}},
		{"reset", `<Period id="p0"`, `<EssentialProperty schemeIdUri="urn:mpeg:dash:reset:2016"/>
  <Period id="p0"`, []string{"MPD-UPDATE-RESET"}},
		{"MPD id", `id="live"`, `id="other"`, []string{"MPD-UPDATE-ID"}},
		{"Period id", `<Period id="p0"`, `<Period id="p1"`, []string{"MPD-UPDATE-ID"}},
		{"AdaptationSet id", `<AdaptationSet id="1"`, `<AdaptationSet id="3"`, []string{"MPD-UPDATE-ID"}},
		{"codecs", `codecs="avc3.64001f"`, `codecs="avc3.640028"`, []string{"MPD-UPDATE-REP", "MPD-UPDATE-REP"}},
		{"width", `width="1920"`, `width="1600"`, []string{"MPD-UPDATE-REP"}},
		{"old segments dropped", `<S t="0" d="2000" r="30"/></SegmentTimeline>
      </SegmentTemplate>
      <Representation id="v1"`, `<S t="20000" d="2000" r="20"/></SegmentTimeline>
      </SegmentTemplate>
      <Representation id="v1"`, nil},
		{"available segment dropped", `<S t="0" d="2000" r="30"/></SegmentTimeline>
      </SegmentTemplate>
      <Representation id="v1"`, `<S t="50000" d="2000" r="5"/></SegmentTimeline>
      </SegmentTemplate>
      <Representation id="v1"`, []string{"MPD-UPDATE-SEGMENTS", "MPD-UPDATE-SEGMENTS"}},
		{"segment timing changed", `<S t="0" d="2000" r="30"/></SegmentTimeline>
      </SegmentTemplate>
      <Representation id="v1"`, `<S t="0" d="2000" r="19"/><S d="1000"/><S d="2000" r="10"/></SegmentTimeline>
      </SegmentTemplate>
      <Representation id="v1"`, []string{"MPD-UPDATE-SEGMENTS", "MPD-UPDATE-SEGMENTS"}},
		{"timeShiftBufferDepth", `timeShiftBufferDepth="PT30S"`, `timeShiftBufferDepth="PT60S"`, []string{"MPD-DVB-UPDATE"}},
		{"availabilityStartTime", `availabilityStartTime="1970-01-01T00:00:00Z"`, `availabilityStartTime="1970-01-01T00:00:00.5Z"`, []string{"MPD-DVB-UPDATE"}},
		{"maxSegmentDuration up", `maxSegmentDuration="PT2S"`, `maxSegmentDuration="PT4S"`, []string{"MPD-DVB-UPDATE"}},
		{"maxSegmentDuration down", `maxSegmentDuration="PT2S"`, `maxSegmentDuration="PT1S"`, nil},
		{"Period start", `<Period id="p0" start="PT0S">`, `<Period id="p0" start="PT1S">`, []string{"MPD-DVB-UPDATE"}},
		{"AssetIdentifier", `value="a1"`, `value="a2"`, []string{"MPD-DVB-UPDATE"}},
		{"Role", `value="main"/>
      <SegmentTemplate`, `value="alternate"/>
      <SegmentTemplate`, []string{"MPD-DVB-UPDATE", "MPD-DVB-UPDATE"}},
		{"lang", `lang="en"`, `lang="sv"`, []string{"MPD-DVB-UPDATE"}},
		{"AudioChannelConfiguration", `value="2"/>`, `value="6"/>`, []string{"MPD-DVB-UPDATE"}},
		{"timescale", `<SegmentTemplate timescale="1000"`, `<SegmentTemplate timescale="90000"`, []string{"MPD-DVB-UPDATE", "MPD-DVB-UPDATE", "MPD-DVB-UPDATE"}},
		{"Representation removed", video, "", []string{"MPD-DVB-UPDATE"}},
		{"Representation added", video, video + `
      <Representation id="v3" bandwidth="3000000" width="1920" height="1080"/>`, nil},
		{"not DVB", `profiles="urn:dvb:dash:profile:dvb-dash:2014,urn:dvb:dash:profile:dvb-dash:isoff-ext-live:2014"`,
			`profiles="urn:mpeg:dash:profile:isoff-live:2011"`, nil},
	}
	t0 := time.Date(1970, 1, 1, 0, 1, 0, 0, time.UTC)
	prev := version(t, updateMPD, t0)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := next
			if c.old != "" {
				if !strings.Contains(data, c.old) {
					t.Fatalf("test MPD lacks %q", c.old)
				}
				data = strings.ReplaceAll(data, c.old, c.new)
			}
			p := prev
			if c.name == "not DVB" {
				p = version(t, strings.ReplaceAll(updateMPD, c.old, c.new), t0)
			}
			var got []string
			for _, f := range CheckUpdate(p, version(t, data, t0.Add(2*time.Second)), "https://example.com/live.mpd") {
				got = append(got, f.Rule)
			}
			slices.Sort(got)
			if !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
	// An unchanged MPD has nothing to compare.
	if f := CheckUpdate(prev, version(t, updateMPD, t0.Add(2*time.Second)), ""); len(f) > 0 {
		t.Errorf("unchanged: %v", f)
	}
	// A utc:direct UTCTiming holds the time of each response, so only its
	// value may change without a new @publishTime.
	utc := func(scheme, value string) string {
		return strings.Replace(updateMPD, "</MPD>", `  <UTCTiming schemeIdUri="urn:mpeg:dash:utc:`+scheme+`:2014" value="`+value+`"/>
</MPD>`, 1)
	}
	for _, c := range []struct {
		scheme, v0, v1 string
		want           int
	}{
		{"direct", "1970-01-01T00:01:00Z", "1970-01-01T00:01:02Z", 0},
		{"http-iso", "https://time.example.com/a", "https://time.example.com/b", 1},
	} {
		f := CheckUpdate(version(t, utc(c.scheme, c.v0), t0), version(t, utc(c.scheme, c.v1), t0.Add(2*time.Second)), "")
		if len(f) != c.want {
			t.Errorf("%s: got %v, want %d findings", c.scheme, f, c.want)
		}
	}
}

func version(t *testing.T, data string, at time.Time) Version {
	t.Helper()
	md, err := m.ReadFromString(data)
	if err != nil {
		t.Fatal(err)
	}
	return Version{MPD: md, Data: []byte(data), Time: at}
}
