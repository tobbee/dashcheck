package segcheck

import (
	"fmt"
	"slices"
	"testing"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
)

func dvbReps(t *testing.T) (audio, video addressing.Rep) {
	rs := reps(t)
	audio, video = *rs[0], *rs[1]
	for _, r := range []*addressing.Rep{&audio, &video} {
		r.Profiles = []string{profileDVB2014, "urn:dvb:dash:profile:dvb-dash:isoff-ext-live:2014"}
	}
	return audio, video
}

func TestDVBInit(t *testing.T) {
	audio, video := dvbReps(t)
	// The test video is avc1, High profile (100) at level 3.0 (30).
	_, f := CheckInit(read(t, "V300/init.mp4"), &video, video.Loc())
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-DVB-AVC3"}) {
		t.Errorf("video: got %v", f)
	}
	audio.AudioChannelConfigs = []*m.DescriptorType{{SchemeIdUri: "urn:mpeg:dash:23003:3:audio_channel_configuration:2011", Value: "2"}}
	if _, f = CheckInit(read(t, "A48/init.mp4"), &audio, audio.Loc()); len(f) > 0 {
		t.Errorf("audio: %v", f)
	}
	audio.AudioChannelConfigs[0].Value = "6"
	_, f = CheckInit(read(t, "A48/init.mp4"), &audio, audio.Loc())
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-DVB-CHANNELS"}) {
		t.Errorf("audio channels: got %v", f)
	}
	schemes := []struct {
		scheme  string
		profile string
		want    int
	}{
		{"cenc", profileDVB2014, 0}, {"cbcs", profileDVB2017, 0}, {"cbcs", profileDVB2014, 1}, {"cens", profileDVB2017, 1},
	}
	for _, c := range schemes {
		r := video
		r.Profiles = []string{c.profile}
		if f := dvbInitRules(&r, &Init{Encrypted: true, Scheme: c.scheme}, sampleEntryInfo{fourCC: "avc3"}, r.Loc()); len(f) != c.want {
			t.Errorf("scheme %s with %s: got %v", c.scheme, c.profile, f)
		}
	}
	// Main 10, 10 bits, progressive frames.
	const flags = 1<<47 | 1<<44
	good := videoConfig{codec: "hevc", hasConfig: true, profile: 2, compatibility: 1 << 29, level: 153,
		bitDepth: 10, chromaBitDepth: 10, constraints: flags, width: 3840, height: 2160}
	cases := []struct {
		desc   string
		mutate func(v *videoConfig)
		want   int
	}{
		{"valid", func(v *videoConfig) {}, 0},
		{"interlaced", func(v *videoConfig) { v.constraints = 1 << 46 }, 1},
		{"main 10 with 12 bits", func(v *videoConfig) { v.bitDepth, v.chromaBitDepth = 12, 12 }, 1},
		{"main 10 with different chroma", func(v *videoConfig) { v.chromaBitDepth = 8 }, 1},
		{"main with 10 bits", func(v *videoConfig) { v.profile, v.compatibility = 1, 3<<29 }, 1},
		{"main with 8 bits", func(v *videoConfig) { v.profile, v.compatibility, v.bitDepth, v.chromaBitDepth = 1, 3<<29, 8, 8 }, 0},
	}
	for _, c := range cases {
		v := good
		c.mutate(&v)
		info := sampleEntryInfo{fourCC: "hvc1", width: uint16(v.width), height: uint16(v.height), video: v}
		if f := dvbInitRules(&video, &Init{}, info, video.Loc()); len(f) != c.want {
			t.Errorf("hevc %s: got %v", c.desc, f)
		}
	}
}

func TestDVBConformancePoints(t *testing.T) {
	_, video := dvbReps(t)
	avc := func(profile, level int, constraints uint64, w, h int) *Init {
		return &Init{Video: videoConfig{codec: "avc", hasConfig: true, profile: profile, level: level, constraints: constraints, width: w, height: h}}
	}
	hevc := func(profile, level int, highTier bool, w, h int) *Init {
		return &Init{Video: videoConfig{codec: "hevc", hasConfig: true, profile: profile, compatibility: 1 << (31 - profile),
			level: level, highTier: highTier, width: w, height: h}}
	}
	cases := []struct {
		desc  string
		inits []*Init
		want  []string
	}{
		{"high 4.0", []*Init{avc(100, 40, 0, 1920, 1080)}, nil},
		{"main 3.1 and high 4.2", []*Init{avc(77, 31, 0, 1280, 720), avc(100, 42, 0, 1920, 1080)}, nil},
		{"constrained baseline", []*Init{avc(66, 21, 1<<6, 640, 360)}, nil},
		{"baseline", []*Init{avc(66, 21, 0, 640, 360)}, []string{"SEG-DVB-AVC"}},
		{"only 4.2", []*Init{avc(100, 42, 0, 1920, 1080)}, []string{"SEG-DVB-AVC"}},
		{"high 10", []*Init{avc(110, 40, 0, 1920, 1080)}, []string{"SEG-DVB-AVC"}},
		{"unknown init", []*Init{avc(100, 42, 0, 1920, 1080), nil}, nil},
		{"hevc main 4.1", []*Init{hevc(1, 123, false, 1920, 1080)}, nil},
		{"hevc main 5.1 uhd", []*Init{hevc(1, 153, false, 3840, 2160)}, nil},
		{"hevc main 10 6.1 8k", []*Init{hevc(2, 183, false, 7680, 4320)}, nil},
		{"hevc 5.1 hd", []*Init{hevc(2, 153, false, 1920, 1080)}, []string{"SEG-DVB-HEVC"}},
		{"hevc high tier", []*Init{hevc(2, 123, true, 1920, 1080)}, []string{"SEG-DVB-HEVC"}},
		{"hevc rext", []*Init{hevc(4, 123, false, 1920, 1080)}, []string{"SEG-DVB-HEVC"}},
	}
	for _, c := range cases {
		reps := make([]*addressing.Rep, len(c.inits))
		for i := range reps {
			r := video
			r.Codecs = "avc1.64001f"
			if c.inits[0].Video.codec == "hevc" {
				r.Codecs = "hvc1.2.4.L123.B0"
			}
			reps[i] = &r
		}
		if got := ruleIDs(CheckPresentation(reps, c.inits)); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.desc, got, c.want)
		}
	}
	// Players without cbcs support need AVC video and audio that are clear or cenc.
	enc := func(handler, codec, scheme string) *Init {
		return &Init{Handler: handler, Encrypted: scheme != "", Scheme: scheme,
			Video: videoConfig{codec: codec, hasConfig: codec != "", profile: 100, level: 40, width: 1920, height: 1080}}
	}
	fallback := []struct {
		desc  string
		inits []*Init
		want  []string
	}{
		{"clear", []*Init{enc("vide", "avc", ""), enc("soun", "", "")}, nil},
		{"cenc", []*Init{enc("vide", "avc", "cenc"), enc("soun", "", "cenc")}, nil},
		{"cbcs with cenc copies", []*Init{enc("vide", "avc", "cbcs"), enc("vide", "avc", "cenc"), enc("soun", "", "cbcs"), enc("soun", "", "")}, nil},
		{"cbcs audio only", []*Init{enc("vide", "avc", "cenc"), enc("soun", "", "cbcs")}, []string{"SEG-DVB-CENC-FALLBACK"}},
		{"cbcs everywhere", []*Init{enc("vide", "avc", "cbcs"), enc("soun", "", "cbcs")}, []string{"SEG-DVB-CENC-FALLBACK", "SEG-DVB-CENC-FALLBACK"}},
		{"unchecked set", []*Init{enc("vide", "avc", "cenc"), enc("soun", "", "cbcs"), nil}, nil},
	}
	for _, c := range fallback {
		reps := make([]*addressing.Rep, len(c.inits))
		for i := range reps {
			r := video
			r.ASPath = fmt.Sprintf("Period[0]/AdaptationSet[%d]", i)
			reps[i] = &r
		}
		if got := ruleIDs(CheckPresentation(reps, c.inits)); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.desc, got, c.want)
		}
	}
	// Each Period is checked on its own.
	p0, p1 := video, video
	p1.PeriodIndex = 1
	f := CheckPresentation([]*addressing.Rep{&p0, &p1}, []*Init{avc(100, 40, 0, 1920, 1080), avc(100, 42, 0, 1920, 1080)})
	if len(f) != 1 || f[0].Location.Path != "Period[1]" {
		t.Errorf("periods: got %v", f)
	}
}

func TestDVBMediaAndSets(t *testing.T) {
	_, video := dvbReps(t)
	init, _ := CheckInit(read(t, "V300/init.mp4"), &video, video.Loc())
	segs, _ := video.Segments(0, video.PeriodDur)
	if _, f := CheckMedia(read(t, "V300/1.m4s"), init, &video, segs[0], false); len(f) > 0 {
		t.Errorf("media: %v", f)
	}
	other := *init
	other.TrackID, other.SampleEntry = 7, "avc3"
	f := CheckAdaptationSet([]*addressing.Rep{&video, &video}, []*Init{init, &other})
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-DVB-SWITCHABLE", "SEG-DVB-SWITCHABLE"}) {
		t.Errorf("switchable: got %v", f)
	}
	ondemand := video
	ondemand.Kind = addressing.KindSegmentBase
	ondemand.Profiles = []string{profileDVB2014, profileDVBOnDemand}
	sidx := &mp4.SidxBox{Timescale: 1000, SidxRefs: []mp4.SidxRef{{SubSegmentDuration: 20000, StartsWithSAP: 1, SAPType: 3}}}
	f = CheckDVBSidx(&ondemand, sidx, 2)
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-DVB-SIDX", "SEG-DVB-DURATION", "SEG-DVB-SAP"}) {
		t.Errorf("sidx: got %v", f)
	}
}
