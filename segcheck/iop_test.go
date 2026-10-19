package segcheck

import (
	"slices"
	"testing"

	"github.com/tobbee/dashcheck/addressing"
)

func TestIOPSwitchingRules(t *testing.T) {
	video := *reps(t)[1]
	video.Profiles = []string{"urn:mpeg:dash:profile:isoff-live:2011", "http://dashif.org/guidelines/dash-if-simple"}

	// Without bitstream switching the rules do not apply.
	_, f := CheckInit(read(t, "V300/init.mp4"), &video, video.Loc())
	if len(f) > 0 {
		t.Errorf("no switching: %v", f)
	}

	video.BitstreamSwitching = true
	init, f := CheckInit(read(t, "V300/init.mp4"), &video, video.Loc())
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-IOP-BITSTREAM"}) {
		t.Errorf("init: got %v", f)
	}
	segs, _ := video.Segments(0, video.PeriodDur)
	_, f = CheckMedia(read(t, "V300/1.m4s"), init, &video, segs[0], false)
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-IOP-FIRSTCT"}) {
		t.Errorf("media: got %v", f)
	}
}

func TestIOPAdaptationSet(t *testing.T) {
	video := *reps(t)[1]
	video.Profiles = []string{"http://dashif.org/guidelines/dash-if-simple"}
	init, _ := CheckInit(read(t, "V300/init.mp4"), &video, video.Loc())
	if init.Video.profile != 100 || init.Video.level != 30 {
		t.Fatalf("video config %v", init.Video)
	}
	cases := []struct {
		asCodecs string
		want     []string
	}{
		{"avc1.64001e", nil},
		{"", nil},
		{"avc1.64001f", []string{"SEG-IOP-ASCODECS"}},
		{"avc1.4d001e", []string{"SEG-IOP-ASCODECS"}},
	}
	for _, c := range cases {
		video.ASCodecs = c.asCodecs
		f := CheckAdaptationSet([]*addressing.Rep{&video}, []*Init{init})
		if got := ruleIDs(f); !slices.Equal(got, c.want) {
			t.Errorf("%q: got %v", c.asCodecs, f)
		}
	}
	// An edit list in one of two Representations.
	video.ASCodecs = ""
	withEdit := *init
	withEdit.HasEditList = true
	f := CheckAdaptationSet([]*addressing.Rep{&video, &video}, []*Init{init, &withEdit})
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-IOP-EDITLIST"}) {
		t.Errorf("edit lists: got %v", f)
	}
}

func TestCodecsProfileLevel(t *testing.T) {
	cases := []struct {
		codecs         string
		profile, level int
		ok             bool
	}{
		{"avc1.64001f", 100, 31, true},
		{"avc3.4d401e", 77, 30, true},
		{"hvc1.1.6.L93.B0", 1, 93, true},
		{"hev1.2.4.H120.90", 2, 120, true},
		{"hvc1.A1.6.L93", 1, 93, true},
		{"avc1.64", 0, 0, false},
		{"mp4a.40.2", 0, 0, false},
	}
	for _, c := range cases {
		p, l, ok := codecsProfileLevel(c.codecs)
		if ok != c.ok || (ok && (p != c.profile || l != c.level)) {
			t.Errorf("%s: got %d %d %v", c.codecs, p, l, ok)
		}
	}
}
