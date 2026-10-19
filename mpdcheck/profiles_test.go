package mpdcheck

import (
	"slices"
	"strings"
	"testing"

	"github.com/tobbee/dashcheck/spec"
)

const iopMPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" xmlns:cenc="urn:mpeg:cenc:2013" type="static" mediaPresentationDuration="PT8S"
     profiles="urn:mpeg:dash:profile:isoff-live:2011,http://dashif.org/guidelines/dash-if-simple" minBufferTime="PT2S">
  <Period id="p0">
    <AdaptationSet id="1" contentType="video" mimeType="video/mp4" codecs="avc1.64001e" par="16:9" segmentAlignment="true"
                   maxWidth="1280" maxHeight="720" frameRate="30" sar="1:1" startWithSAP="1">
      <ContentProtection schemeIdUri="urn:mpeg:dash:mp4protection:2011" value="cenc" cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"/>
      <Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>
      <SegmentTemplate timescale="90000" duration="180000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="v1" bandwidth="1000000" width="640" height="360"/>
      <Representation id="v2" bandwidth="2000000" width="1280" height="720"/>
    </AdaptationSet>
    <AdaptationSet id="2" contentType="audio" mimeType="audio/mp4" codecs="mp4a.40.2" lang="en" segmentAlignment="true" audioSamplingRate="48000">
      <AudioChannelConfiguration schemeIdUri="urn:mpeg:mpegB:cicp:ChannelConfiguration" value="2"/>
      <SegmentTemplate timescale="48000" duration="96000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="a1" bandwidth="128000"/>
    </AdaptationSet>
  </Period>
</MPD>`

func TestProfileRules(t *testing.T) {
	cases := []struct {
		name     string
		old, new string
		want     []string
	}{
		{"clean", "", "", nil},
		{"on-demand in dynamic", `type="static" mediaPresentationDuration="PT8S"
     profiles="urn:mpeg:dash:profile:isoff-live:2011,`, `type="dynamic" availabilityStartTime="1970-01-01T00:00:00Z" publishTime="2026-01-01T00:00:00Z" minimumUpdatePeriod="PT2S"
     profiles="urn:mpeg:dash:profile:isoff-on-demand:2011,`, []string{"MPD-DYN-TSBD", "MPD-DYN-UTCTIMING", "MPD-IOP-LIVE", "MPD-IOP-ONDEMAND", "MPD-IOP-ONDEMAND",
			"MPD-IOP-ONDEMAND", "MPD-IOP-ONDEMAND", "MPD-IOP-ONDEMAND", "MPD-PROFILE-TYPE"}},
		{"static with TSBD", `minBufferTime="PT2S">`, `minBufferTime="PT2S" timeShiftBufferDepth="PT30S">`, []string{"MPD-IOP-STATIC"}},
		{"no segmentAlignment", `lang="en" segmentAlignment="true"`, `lang="en"`, []string{"MPD-IOP-LIVE"}},
		{"video without par", ` par="16:9"`, "", []string{"MPD-IOP-VIDEO"}},
		{"video without sar", ` sar="1:1"`, "", []string{"MPD-IOP-VIDEO", "MPD-IOP-VIDEO"}},
		{"sizes differ without max", ` maxWidth="1280"`, ` width="1280"`, []string{"MPD-COMMON-BOTH", "MPD-IOP-VIDEO"}},
		{"interlaced", `par="16:9"`, `par="16:9" scanType="interlaced"`, []string{"MPD-IOP-SCANTYPE"}},
		{"audio without lang", ` lang="en"`, "", []string{"MPD-IOP-AUDIO"}},
		{"audio without channel config", `<AudioChannelConfiguration schemeIdUri="urn:mpeg:mpegB:cicp:ChannelConfiguration" value="2"/>`, "",
			[]string{"MPD-IOP-AUDIO"}},
		{"bad mimeType", `mimeType="audio/mp4"`, `mimeType="audio/aac"`, []string{"MPD-IOP-MIMETYPE"}},
		{"two codecs", `codecs="mp4a.40.2"`, `codecs="mp4a.40.2,ec-3"`, []string{"MPD-IOP-CODECS"}},
		{"cp without default_KID", ` cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"`, "", []string{"MPD-IOP-CP"}},
		{"cp on representation", `<Representation id="v1" bandwidth="1000000" width="640" height="360"/>`,
			`<Representation id="v1" bandwidth="1000000" width="640" height="360"><ContentProtection schemeIdUri="urn:x" value="y"/></Representation>`,
			[]string{"MPD-IOP-CP"}},
		{"two videos without main", `value="main"/>`, `value="alternate"/>`, nil},
		{"two videos, none main", `<Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>
      <SegmentTemplate timescale="90000" duration="180000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="v1" bandwidth="1000000" width="640" height="360"/>
      <Representation id="v2" bandwidth="2000000" width="1280" height="720"/>
    </AdaptationSet>`, `<Role schemeIdUri="urn:mpeg:dash:role:2011" value="alternate"/>
      <SegmentTemplate timescale="90000" duration="180000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="v1" bandwidth="1000000" width="640" height="360"/>
      <Representation id="v2" bandwidth="2000000" width="1280" height="720"/>
    </AdaptationSet>
    <AdaptationSet id="4" contentType="video" mimeType="video/mp4" codecs="avc1.64001e" par="16:9" segmentAlignment="true" width="640" height="360" frameRate="30" sar="1:1">
      <SegmentTemplate timescale="90000" duration="180000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="v3" bandwidth="1000000"/>
    </AdaptationSet>`, []string{"MPD-PROFILE-MAINROLE"}},
		{"profile not in MPD", `<Representation id="a1" bandwidth="128000"/>`,
			`<Representation id="a1" bandwidth="128000" profiles="urn:mpeg:dash:profile:isoff-on-demand:2011"/>`,
			[]string{"MPD-PROFILE-SUBSET"}},
		{"live without template", `<SegmentTemplate timescale="48000" duration="96000" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>`,
			`<BaseURL>a.mp4</BaseURL>`, []string{"MPD-PROFILE-SEGINFO"}},
		{"thumbnails", `</Period>`, `<AdaptationSet id="3" contentType="image" mimeType="image/jpeg" codecs="jpeg" segmentAlignment="true"><SegmentTemplate duration="2" media="t/$Number$.jpg"/><Representation id="t" bandwidth="1000" width="320"><EssentialProperty schemeIdUri="http://dashif.org/guidelines/thumbnail_tile" value="10"/></Representation></AdaptationSet></Period>`,
			[]string{"MPD-THUMBNAIL", "MPD-THUMBNAIL"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := iopMPD
			if c.old != "" {
				if !strings.Contains(data, c.old) {
					t.Fatalf("test MPD lacks %q", c.old)
				}
				data = strings.Replace(data, c.old, c.new, 1)
			}
			md, findings := Check([]byte(data), "https://example.com/iop.mpd")
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

func TestProfileChoice(t *testing.T) {
	noDVB := strings.Replace(dvbMPD, "urn:dvb:dash:profile:dvb-dash:2014,", "", 1)
	badDVB := strings.Replace(dvbMPD, ` par="16:9"`, "", 1)
	noLL := strings.Replace(llMPD, ",http://www.dashif.org/guidelines/low-latency-live-v5", "", 1)
	cases := []struct {
		name, mpd, choice string
		want              []string // the MPD-DVB-, MPD-LL- and MPD-PROFILE-CHOSEN findings
	}{
		{"dvb signalled", badDVB, "auto", []string{"MPD-DVB-VIDEO"}},
		{"dvb not signalled", noDVB, "auto", nil},
		{"dvb chosen", noDVB, "dvb", []string{"MPD-PROFILE-CHOSEN"}},
		{"dvb chosen and signalled", badDVB, "dvb", []string{"MPD-DVB-VIDEO"}},
		{"dvb not chosen", badDVB, "none", nil},
		{"dvb added", strings.Replace(badDVB, "urn:dvb:dash:profile:dvb-dash:2014,", "", 1), "auto,dvb",
			[]string{"MPD-DVB-VIDEO", "MPD-PROFILE-CHOSEN"}},
		{"chunked", noLL, "auto", []string{"MPD-LL-PROFILE"}},
		{"chunked, other profile chosen", noLL, "iop", []string{"MPD-PROFILE-CHOSEN"}},
		{"chunked, low latency chosen", noLL, "low-latency", []string{"MPD-PROFILE-CHOSEN"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sel, err := spec.Parse(c.choice)
			if err != nil {
				t.Fatal(err)
			}
			md, findings := CheckWith([]byte(c.mpd), "https://example.com/Manifest.mpd", sel)
			if md == nil {
				t.Fatalf("MPD not parsed: %v", findings)
			}
			var got []string
			for _, f := range findings {
				if strings.HasPrefix(f.Rule, "MPD-DVB-") || strings.HasPrefix(f.Rule, "MPD-LL-") || f.Rule == "MPD-PROFILE-CHOSEN" {
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
