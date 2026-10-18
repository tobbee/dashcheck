package mpdcheck

import (
	"slices"
	"strings"
	"testing"
)

const dvbMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="dynamic" availabilityStartTime="1970-01-01T00:00:00Z"
     publishTime="2026-10-18T10:00:00Z" minimumUpdatePeriod="PT2S" timeShiftBufferDepth="PT30S" minBufferTime="PT2S"
     profiles="urn:dvb:dash:profile:dvb-dash:2014,urn:dvb:dash:profile:dvb-dash:isoff-ext-live:2014">
  <Period id="p0" start="PT0S">
    <AdaptationSet id="1" contentType="video" mimeType="video/mp4" codecs="avc1.64001f" par="16:9"
                   maxWidth="1280" maxHeight="720" frameRate="25" segmentAlignment="true" startWithSAP="1">
      <SegmentTemplate timescale="1000" duration="2000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="v1" bandwidth="1000000" width="640" height="360"/>
      <Representation id="v2" bandwidth="3000000" width="1280" height="720"/>
    </AdaptationSet>
    <AdaptationSet id="2" contentType="audio" mimeType="audio/mp4" codecs="mp4a.40.2" lang="en" audioSamplingRate="48000" segmentAlignment="true" startWithSAP="1">
      <AudioChannelConfiguration schemeIdUri="urn:mpeg:dash:23003:3:audio_channel_configuration:2011" value="2"/>
      <Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>
      <SegmentTemplate timescale="1000" duration="2000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="a1" bandwidth="128000"/>
    </AdaptationSet>
  </Period>
  <UTCTiming schemeIdUri="urn:mpeg:dash:utc:http-iso:2014" value="https://time.akamai.com/?iso&amp;ms"/>
</MPD>`

func TestDVBRules(t *testing.T) {
	cases := []struct {
		name     string
		old, new string
		want     []string
	}{
		{"clean", "", "", nil},
		{"short mup", `minimumUpdatePeriod="PT2S"`, `minimumUpdatePeriod="PT0.5S"`, []string{"MPD-DVB-MUP"}},
		{"no utc timing", `<UTCTiming schemeIdUri="urn:mpeg:dash:utc:http-iso:2014" value="https://time.akamai.com/?iso&amp;ms"/>`, "",
			[]string{"MPD-DVB-UTCTIMING", "MPD-DYN-UTCTIMING"}},
		{"no sub-profile", `,urn:dvb:dash:profile:dvb-dash:isoff-ext-live:2014"`, `"`,
			[]string{"MPD-DVB-PROFILE", "MPD-DVB-PROFILE", "MPD-DVB-PROFILE"}},
		{"video without par", ` par="16:9"`, "", []string{"MPD-DVB-VIDEO"}},
		{"odd resolution", `width="640" height="360"`, `width="600" height="360"`, []string{"MPD-DVB-RESOLUTION"}},
		{"audio without role", `<Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>`, "", []string{"MPD-DVB-AUDIO"}},
		{"audio without channel config", `<AudioChannelConfiguration schemeIdUri="urn:mpeg:dash:23003:3:audio_channel_configuration:2011" value="2"/>`, "",
			[]string{"MPD-DVB-AUDIO"}},
		{"e-ac-3 mpeg scheme", `codecs="mp4a.40.2"`, `codecs="ec-3"`, []string{"MPD-DVB-AUDIO"}},
		{"high audio share", `bandwidth="128000"`, `bandwidth="512000"`, []string{"MPD-DVB-AUDIO-SHARE"}},
		{"unsupported codec", `codecs="mp4a.40.2"`, `codecs="opus"`, []string{"MPD-DVB-CODEC"}},
		{"drm without generic cp", `<Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>`,
			`<ContentProtection schemeIdUri="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed"/><Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>`,
			[]string{"MPD-DVB-CP"}},
		{"font on video", `<SegmentTemplate timescale="1000" duration="2000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="v1"`, `<SupplementalProperty schemeIdUri="urn:dvb:dash:fontdownload:2014" value="1"/>
      <SegmentTemplate timescale="1000" duration="2000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="v1"`, []string{"MPD-DVB-FONT"}},
		{"fallback to missing", `<Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>`,
			`<Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/><SupplementalProperty schemeIdUri="urn:dvb:dash:fallback_adaptation_set:2014" value="7"/>`,
			[]string{"MPD-DVB-FALLBACK"}},
		{"segment list on period", `<Period id="p0" start="PT0S">`, `<Period id="p0" start="PT0S"><SegmentList duration="2"/>`,
			[]string{"MPD-DVB-PERIOD"}},
		{"frame rates not multiples", `<Representation id="v2" bandwidth="3000000" width="1280" height="720"/>`,
			`<Representation id="v2" bandwidth="3000000" width="1280" height="720" frameRate="30"/>`, []string{"MPD-COMMON-BOTH", "MPD-DVB-FRAMERATES"}},
		{"scte35 stream", `<AdaptationSet id="1"`, `<EventStream schemeIdUri="urn:scte:scte35:2014:xml+bin" value="1" timescale="90000"><Event presentationTime="0" duration="4294967295" messageData="x"/></EventStream>
    <AdaptationSet id="1"`, []string{"MPD-DVB-SCTE", "MPD-DVB-SCTE", "MPD-DVB-SCTE-VALUE"}},
		{"tv-anytime event", `<AdaptationSet id="1"`, `<EventStream schemeIdUri="urn:dvb:iptv:cpm:2014" value="1"><Event>&lt;BroadcastEvent&gt;</Event></EventStream>
    <AdaptationSet id="1"`, []string{"MPD-DVB-EVENT", "MPD-DVB-EVENT"}},
		{"metric reporting", `<UTCTiming`, `<Metrics metrics="DVBErrors"><Reporting schemeIdUri="urn:dvb:dash:reporting:2014" value="1" reportingUrl="/report" probability="0"/></Metrics>
  <UTCTiming`, []string{"MPD-DVB-REPORTING", "MPD-DVB-REPORTING"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := dvbMPD
			if c.old != "" {
				if !strings.Contains(data, c.old) {
					t.Fatalf("test MPD lacks %q", c.old)
				}
				data = strings.Replace(data, c.old, c.new, 1)
			}
			md, findings := Check([]byte(data), "https://example.com/dvb.mpd")
			if md == nil {
				t.Fatalf("MPD not parsed: %v", findings)
			}
			var got []string
			for _, f := range findings {
				if f.Rule != "MPD-XSD" && f.Rule != "MPD-PROFILE-UNKNOWN" {
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

func TestDVBContinuity(t *testing.T) {
	// Two Periods with the same AssetIdentifier; both AdaptationSets of p1 continue p0.
	period := dvbMPD[strings.Index(dvbMPD, "  <Period"):strings.Index(dvbMPD, "  <UTCTiming")]
	asset := `<AssetIdentifier schemeIdUri="urn:example:asset" value="show"/>
`
	p0 := strings.Replace(period, `start="PT0S">
`, `start="PT0S">
    `+asset, 1)
	p1 := strings.ReplaceAll(p0, `start="PT0S"`, `start="PT60S"`)
	p1 = strings.Replace(p1, `<Period id="p0"`, `<Period id="p1"`, 1)
	p1 = strings.ReplaceAll(p1, `startWithSAP="1">
`, `startWithSAP="1">
      <SupplementalProperty schemeIdUri="urn:dvb:dash:period_continuity:2014" value="p0"/>
`)
	base := strings.Replace(dvbMPD, period, p0+p1, 1)
	cases := []struct {
		name     string
		old, new string
		want     int
	}{
		{"associated", "", "", 0},
		{"unknown Period", `value="p0"/>`, `value="p9"/>`, 1},
		{"no AdaptationSet with the id", `<AdaptationSet id="2" contentType="audio" mimeType="audio/mp4" codecs="mp4a.40.2" lang="en" audioSamplingRate="48000" segmentAlignment="true" startWithSAP="1">
      <Supplemental`, `<AdaptationSet id="3" contentType="audio" mimeType="audio/mp4" codecs="mp4a.40.2" lang="en" audioSamplingRate="48000" segmentAlignment="true" startWithSAP="1">
      <Supplemental`, 1},
		{"other language", `lang="en" audioSamplingRate="48000" segmentAlignment="true" startWithSAP="1">
      <Supplemental`, `lang="sv" audioSamplingRate="48000" segmentAlignment="true" startWithSAP="1">
      <Supplemental`, 1},
		{"other sampling rate", `audioSamplingRate="48000" segmentAlignment="true" startWithSAP="1">
      <Supplemental`, `audioSamplingRate="44100" segmentAlignment="true" startWithSAP="1">
      <Supplemental`, 1},
		{"no AssetIdentifier", `start="PT60S">
    ` + asset, `start="PT60S">
`, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := base
			if c.old != "" {
				if !strings.Contains(data, c.old) {
					t.Fatalf("test MPD lacks %q", c.old)
				}
				data = strings.Replace(data, c.old, c.new, 1)
			}
			md, findings := Check([]byte(data), "https://example.com/dvb.mpd")
			if md == nil {
				t.Fatalf("MPD not parsed: %v", findings)
			}
			if f := dvbContinuity(md); len(f) != c.want {
				t.Errorf("got %d findings, want %d: %v", len(f), c.want, f)
			}
		})
	}
}
