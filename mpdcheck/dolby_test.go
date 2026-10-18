package mpdcheck

import (
	"slices"
	"strings"
	"testing"
)

const dvbAC4MPD = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static" mediaPresentationDuration="PT8S"
     profiles="urn:dvb:dash:profile:dvb-dash:2017" minBufferTime="PT2S">
  <Period id="p0">
    <AdaptationSet id="1" contentType="video" mimeType="video/mp4" codecs="avc1.64001e" frameRate="50">
      <SegmentTemplate duration="2" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="v1" bandwidth="1000000" width="1280" height="720"/>
    </AdaptationSet>
    <AdaptationSet id="2" contentType="audio" mimeType="audio/mp4" codecs="ac-4.02.01.01" lang="en" audioSamplingRate="48000" startWithSAP="1">
      <AudioChannelConfiguration schemeIdUri="urn:mpeg:mpegB:cicp:ChannelConfiguration" value="2"/>
      <Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>
      <SegmentTemplate duration="2" media="$RepresentationID$/$Number$.m4s" initialization="$RepresentationID$/init.mp4"/>
      <Representation id="a1" bandwidth="96000"/>
      <Representation id="a2" bandwidth="128000"/>
    </AdaptationSet>
  </Period>
</MPD>`

func TestDolbyRules(t *testing.T) {
	cases := []struct {
		name     string
		old, new string
		want     []string
	}{
		{"clean", "", "", nil},
		{"bad ac-4 codecs", `codecs="ac-4.02.01.01"`, `codecs="ac-4.02.01"`, []string{"MPD-AC4-CODECS"}},
		{"bitstream version 0 is not 2", `codecs="ac-4.02.01.01"`, `codecs="ac-4.00.00.01"`,
			[]string{"MPD-DVB-AC4", "MPD-DVB-AC4-VERSION"}},
		{"presentation version", `codecs="ac-4.02.01.01"`, `codecs="ac-4.02.02.01"`, []string{"MPD-AC4-CODECS", "MPD-DVB-AC4"}},
		{"sampling rate", `audioSamplingRate="48000"`, `audioSamplingRate="44100"`, []string{"MPD-DVB-AC4"}},
		{"no startWithSAP", ` startWithSAP="1"`, "", []string{"MPD-AC4-ATTRS"}},
		{"mpeg channel config 8", `value="2"/>`, `value="8"/>`, []string{"MPD-AC4-CHANNELCONFIG", "MPD-AC4-CHANNELCONFIG"}},
		{"dolby config mappable", `schemeIdUri="urn:mpeg:mpegB:cicp:ChannelConfiguration" value="2"`,
			`schemeIdUri="tag:dolby.com,2015:dash:audio_channel_configuration:2015" value="000001"`,
			[]string{"MPD-AC4-CHANNELCONFIG", "MPD-AC4-CHANNELCONFIG"}},
		{"different codecs", `<Representation id="a2" bandwidth="128000"/>`, `<Representation id="a2" bandwidth="128000" codecs="ac-4.02.01.02"/>`,
			[]string{"MPD-COMMON-BOTH", "MPD-DVB-AUDIO-SIMILAR"}},
		{"auxiliary with lang", `<Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/>`,
			`<Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/><EssentialProperty schemeIdUri="urn:mpeg:dash:preselection:2016" value="1"/>`,
			[]string{"MPD-DVB-AUXILIARY"}},
		{"high audio frame rate", `<Role schemeIdUri`, `<SupplementalProperty schemeIdUri="tag:dolby.com,2017:dash:audio_frame_rate:2017" value="100"/><Role schemeIdUri`,
			[]string{"MPD-DVB-AC4"}},
		{"preselection without role", `</Period>`, `<Preselection id="10" preselectionComponents="2" codecs="ac-4.02.01.01"/></Period>`,
			[]string{"MPD-DVB-PRESELECTION"}},
		{"preselection audio frame rate", `</Period>`, `<Preselection id="10" preselectionComponents="2" codecs="ac-4.02.01.01"><Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/><SupplementalProperty schemeIdUri="tag:dolby.com,2017:dash:audio_frame_rate:2017" value="120"/></Preselection></Period>`,
			[]string{"MPD-DVB-AC4"}},
		{"preselection bad component", `</Period>`, `<Preselection id="10" preselectionComponents="9" codecs="ac-4.02.01.01"><Role schemeIdUri="urn:mpeg:dash:role:2011" value="main"/></Preselection></Period>`,
			[]string{"MPD-DVB-PRESELECTION"}},
		{"two preselections, none main", `</Period>`, `<Preselection id="10" preselectionComponents="2" codecs="ac-4.02.01.01"><Role schemeIdUri="urn:mpeg:dash:role:2011" value="alternate"/></Preselection><Preselection id="11" preselectionComponents="2" codecs="ac-4.02.01.01"><Role schemeIdUri="urn:mpeg:dash:role:2011" value="commentary"/></Preselection></Period>`,
			[]string{"MPD-DVB-PRESELECTION"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := dvbAC4MPD
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
				if f.Rule != "MPD-XSD" {
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
