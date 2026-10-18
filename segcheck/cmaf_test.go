package segcheck

import (
	"slices"
	"testing"

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
