package segcheck

import (
	"bytes"
	"slices"
	"testing"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
)

func TestCMAFRules(t *testing.T) {
	rs := reps(t)
	audio, video := *rs[0], *rs[1]
	for _, r := range []*addressing.Rep{&audio, &video} {
		r.Profiles = []string{"urn:mpeg:dash:profile:cmaf:2019"}
	}
	vInit, f := CheckInit(read(t, "V300/init.mp4"), &video, video.Loc())
	if len(f) > 0 || !vInit.CMAF {
		t.Fatalf("video init: CMAF=%v %v", vInit.CMAF, f)
	}
	segs, _ := video.Segments(0, video.PeriodDur)
	// The test content has B-frames (composition offsets) in trun version 0 but no edit list.
	_, f = CheckMedia(read(t, "V300/1.m4s"), vInit, &video, segs[0], false)
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-CMAF-COMPTIME"}) {
		t.Errorf("video media: got %v", f)
	}
	withEdit := *vInit
	withEdit.HasEditList = true
	_, f = CheckMedia(read(t, "V300/1.m4s"), &withEdit, &video, segs[0], false)
	if len(f) > 0 {
		t.Errorf("video with edit list: %v", f)
	}

	aInit, f := CheckInit(read(t, "A48/init.mp4"), &audio, audio.Loc())
	if len(f) > 0 {
		t.Fatalf("audio init: %v", f)
	}
	// Audio and video in one AdaptationSet are not a valid switching set.
	f = CheckAdaptationSet([]*addressing.Rep{&audio, &video}, []*Init{aInit, vInit})
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-CMAF-SWITCHINGSET"}) {
		t.Errorf("switching set: got %v", f)
	}

	// Encryption profiles: a clear track with the cmfhdc profile is fine, an encrypted cmfhd is not.
	video.Profiles = append(video.Profiles, "urn:mpeg:cmaf:representation_profile:cmfhd:2017")
	enc := *vInit
	enc.Encrypted, enc.Scheme = true, "cbcs"
	if f := cmafInitRules(&video, &enc, sampleEntryInfo{fourCC: "avc1", width: 1, height: 1, video: videoConfig{codec: "avc", hasConfig: true}}, nil, video.Loc()); len(f) != 1 || f[0].Rule != "SEG-CMAF-ENCRYPTION" {
		t.Errorf("encryption profile: got %v", f)
	}
}

// TestCMAFFragment mutates a decoded video segment: a CMAF chunk has one
// traf with one trun, and a video CMAF fragment starts with a sync sample.
func TestCMAFFragment(t *testing.T) {
	video := *reps(t)[1]
	init := &Init{CMAF: true, Handler: "vide", HasEditList: true}
	cases := []struct {
		name         string
		startWithSAP uint32
		modify       func(frag *mp4.Fragment, media *Media)
		want         []string
	}{
		{"clean", 0, func(*mp4.Fragment, *Media) {}, nil},
		{"two trun boxes", 0, func(frag *mp4.Fragment, _ *Media) {
			_ = frag.Moof.Traf.AddChild(frag.Moof.Traf.Trun)
		}, []string{"SEG-CMAF-CHUNK"}},
		{"two traf boxes", 0, func(frag *mp4.Fragment, _ *Media) {
			_ = frag.Moof.AddChild(frag.Moof.Traf)
		}, []string{"SEG-CMAF-CHUNK"}},
		{"starts with a non-sync sample", 0, func(_ *mp4.Fragment, media *Media) { media.FirstSync = false },
			[]string{"SEG-CMAF-SAP"}},
		{"non-sync start reported by SEG-SAP", 1, func(_ *mp4.Fragment, media *Media) { media.FirstSync = false }, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mf, err := mp4.DecodeFile(bytes.NewReader(read(t, "V300/1.m4s")))
			if err != nil {
				t.Fatal(err)
			}
			rep := video
			rep.StartWithSAP = c.startWithSAP
			media := &Media{FirstSync: true}
			c.modify(mf.Segments[0].Fragments[0], media)
			if got := ruleIDs(cmafMediaRules(&rep, init, media, mf, rep.Loc())); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsCMAF(t *testing.T) {
	cases := []struct {
		name     string
		profiles []string
		brands   []string
		only     bool
		want     bool
	}{
		{"profile", []string{profileCMAF}, nil, false, true},
		{"brand", nil, []string{"iso6", "cmfc"}, false, true},
		{"no signal", nil, []string{"iso6"}, false, false},
		{"brand, profiles chosen", nil, []string{"iso6", "cmfc"}, true, false},
		{"profile, profiles chosen", []string{profileCMAF}, []string{"cmfc"}, true, true},
	}
	for _, c := range cases {
		rep := &addressing.Rep{Profiles: c.profiles, OnlyProfiles: c.only}
		if got := isCMAF(rep, c.brands); got != c.want {
			t.Errorf("%s: isCMAF = %v, want %v", c.name, got, c.want)
		}
	}
	// Low-latency chunked segments must be CMAF fragments; segment mode need not.
	if !isCMAF(&addressing.Rep{LowLatency: true, ATO: 1.5}, nil) {
		t.Error("low-latency chunked: not CMAF")
	}
	if isCMAF(&addressing.Rep{LowLatency: true}, nil) {
		t.Error("low-latency segment mode: CMAF")
	}
}

func TestCMAFSize(t *testing.T) {
	cmaf := []string{profileCMAF}
	cases := []struct {
		name     string
		profiles []string
		maxW     uint32
		maxH     uint32
		sar      string
		widths   [2]uint32 // @width of the two Representations
		tkhd     [2][2]uint32
		want     []string
	}{
		{"max size", cmaf, 1280, 720, "", [2]uint32{}, [2][2]uint32{{640, 360}, {1280, 720}}, nil},
		{"max width too large", cmaf, 1920, 720, "", [2]uint32{}, [2][2]uint32{{640, 360}, {1280, 720}}, []string{"XCHK-CMAF-SIZE"}},
		{"max height too small", cmaf, 1280, 360, "", [2]uint32{}, [2][2]uint32{{640, 360}, {1280, 720}}, []string{"XCHK-CMAF-SIZE"}},
		{"max width with sar", cmaf, 1440, 1080, "4:3", [2]uint32{}, [2][2]uint32{{960, 720}, {1920, 1080}}, nil},
		{"widths", cmaf, 0, 0, "", [2]uint32{640, 1280}, [2][2]uint32{{640, 360}, {1280, 720}}, nil},
		{"wrong width", cmaf, 0, 0, "", [2]uint32{640, 1024}, [2][2]uint32{{640, 360}, {1280, 720}}, []string{"XCHK-CMAF-SIZE"}},
		{"not the CMAF profile", nil, 1920, 1080, "", [2]uint32{}, [2][2]uint32{{640, 360}, {1280, 720}}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var rs []*addressing.Rep
			var inits []*Init
			for i := range 2 {
				rs = append(rs, &addressing.Rep{ID: string(rune('a' + i)), ASPath: "Period[0]/AdaptationSet[0]", Profiles: c.profiles,
					MaxWidth: c.maxW, MaxHeight: c.maxH, Sar: c.sar, Width: c.widths[i]})
				inits = append(inits, &Init{Handler: "vide", tkhdWidth: c.tkhd[i][0], tkhdHeight: c.tkhd[i][1]})
			}
			if got := ruleIDs(cmafSizeRules(rs, inits)); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
