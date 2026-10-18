package segcheck

import (
	"fmt"
	"slices"
	"testing"

	"github.com/tobbee/dashcheck/addressing"
)

func TestWAVERules(t *testing.T) {
	_, base := dvbReps(t)
	base.Profiles = []string{"urn:mpeg:dash:profile:cmaf:2019", "urn:cta:wave:test-content-media-profile:2022"}
	track := func(handler, entry, scheme string, brands ...string) *Init {
		return &Init{Handler: handler, SampleEntry: entry, Brands: brands, Encrypted: scheme != "", Scheme: scheme}
	}
	type set struct {
		period int
		id     string
		rate   string
		width  uint32
		inits  []*Init
	}
	cases := []struct {
		desc string
		sets []set
		want []string
	}{
		{"one profile per set", []set{
			{0, "1", "25", 1920, []*Init{track("vide", "avc1", "", "cmfc", "cfhd"), track("vide", "avc1", "", "cfhd")}},
			{0, "2", "", 0, []*Init{track("soun", "mp4a", "", "caac")}},
		}, nil},
		{"no WAVE brand", []set{{0, "1", "25", 1920, []*Init{track("vide", "avc1", "", "cmfc")}}}, []string{"SEG-WAVE-PROFILE"}},
		{"mixed brands in the set", []set{{0, "1", "25", 1920, []*Init{track("vide", "avc1", "", "cfhd"), track("vide", "hvc1", "", "chh1")}}}, []string{"SEG-WAVE-PROFILE"}},
		{"another set covers it", []set{
			{0, "1", "25", 1920, []*Init{track("vide", "avc1", "", "cmfc")}},
			{0, "3", "25", 1920, []*Init{track("vide", "avc1", "", "cfhd")}},
		}, nil},
		{"unchecked track", []set{{0, "1", "25", 1920, []*Init{track("vide", "avc1", "", "cmfc"), nil}}}, nil},
		{"two schemes", []set{
			{0, "1", "25", 1920, []*Init{track("vide", "avc1", "cenc", "cfhd")}},
			{0, "2", "", 0, []*Init{track("soun", "mp4a", "cbcs", "caac")}},
		}, []string{"SEG-WAVE-SCHEME"}},
		{"clean splice", []set{
			{0, "1", "25", 1920, []*Init{track("vide", "avc1", "", "cfhd")}},
			{1, "1", "50", 1920, []*Init{track("vide", "avc1", "", "cfhd")}},
		}, nil},
		{"splice changes", []set{
			{0, "1", "25", 1920, []*Init{track("vide", "avc1", "", "cfhd")}},
			{1, "1", "30", 1440, []*Init{track("vide", "avc3", "", "cfhd")}},
		}, []string{"SEG-WAVE-SPLICE"}},
	}
	for _, c := range cases {
		var reps []*addressing.Rep
		var inits []*Init
		for si, s := range c.sets {
			for ti, init := range s.inits {
				r := base
				r.PeriodIndex, r.ASID, r.FrameRate, r.Width, r.Height = s.period, s.id, s.rate, s.width, 1080
				r.ASPath = fmt.Sprintf("Period[%d]/AdaptationSet[%d]", s.period, si)
				r.ID = fmt.Sprintf("r%d_%d", si, ti)
				reps, inits = append(reps, &r), append(inits, init)
			}
		}
		if got := ruleIDs(waveRules(reps, inits)); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.desc, got, c.want)
		}
	}
	// Without a WAVE profile nothing is checked.
	plain := base
	plain.Profiles = []string{"urn:mpeg:dash:profile:isoff-live:2011"}
	if f := waveRules([]*addressing.Rep{&plain}, []*Init{track("vide", "avc1", "")}); len(f) > 0 {
		t.Errorf("not WAVE: %v", f)
	}
}
