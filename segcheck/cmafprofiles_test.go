package segcheck

import (
	"slices"
	"testing"

	"github.com/tobbee/dashcheck/addressing"
)

func TestVideoMediaProfile(t *testing.T) {
	rep := &addressing.Rep{ID: "v", FrameRate: "30"}
	ok := sampleEntryInfo{width: 1920, height: 1080,
		video: videoConfig{codec: "avc", profile: 100, level: 40, colourDescribed: true, primaries: 1, transfer: 1, matrix: 1}}
	cases := []struct {
		name   string
		brand  string
		modify func(i *sampleEntryInfo, r *addressing.Rep)
		n      int
	}{
		{"cfhd ok", "cfhd", func(i *sampleEntryInfo, r *addressing.Rep) {}, 0},
		{"cfhd level too high", "cfhd", func(i *sampleEntryInfo, r *addressing.Rep) { i.video.level = 42 }, 1},
		{"cfhd too wide", "cfhd", func(i *sampleEntryInfo, r *addressing.Rep) { i.width = 3840 }, 1},
		{"cfhd PQ", "cfhd", func(i *sampleEntryInfo, r *addressing.Rep) { i.video.transfer = 16 }, 1},
		{"cfhd no colour info", "cfhd", func(i *sampleEntryInfo, r *addressing.Rep) { i.video.colourDescribed, i.video.primaries = false, 9 }, 0},
		{"cfhd 120 fps", "cfhd", func(i *sampleEntryInfo, r *addressing.Rep) { r.FrameRate = "120" }, 1},
		{"cfsd too large", "cfsd", func(i *sampleEntryInfo, r *addressing.Rep) {}, 2},
		{"chdf level 4.2", "chdf", func(i *sampleEntryInfo, r *addressing.Rep) { i.video.level = 42 }, 0},
	}
	for _, c := range cases {
		info, r := ok, *rep
		c.modify(&info, &r)
		f := checkVideoProfile(c.brand, videoProfiles[c.brand], &r, info, r.Loc())
		if len(f) != c.n {
			t.Errorf("%s: got %d findings, want %d: %v", c.name, len(f), c.n, f)
		}
	}
}

func TestAACMediaProfile(t *testing.T) {
	rep := &addressing.Rep{ID: "a"}
	init := &Init{Brands: []string{"cmfc", "caac"}}
	for _, c := range []struct {
		aac audioConfig
		n   int
	}{
		{audioConfig{2, 48000, 2}, 0},
		{audioConfig{2, 96000, 2}, 1},
		{audioConfig{2, 48000, 6}, 1},
		{audioConfig{42, 48000, 2}, 1},
	} {
		a := c.aac
		f := cmafMediaProfileRules(rep, init, sampleEntryInfo{aac: &a}, rep.Loc())
		if len(f) != c.n {
			t.Errorf("%+v: got %v", c.aac, f)
		}
	}
}

func TestMediaProfileSets(t *testing.T) {
	rep := &addressing.Rep{ID: "a", ASPath: "AS", Profiles: []string{"urn:mpeg:cmaf:presentation_profile:cmfhd:2016"}}
	aac := &Init{CMAF: true, Handler: "soun", Brands: []string{"cmfc"}}
	f := cmafMediaProfileSetRules([]*addressing.Rep{rep}, []*Init{aac})
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-CMAF-PRESENTATIONPROFILE"}) {
		t.Errorf("presentation profile: got %v", f)
	}
	h1 := &Init{CMAF: true, Handler: "vide", Video: videoConfig{codec: "hevc", spsSummary: "a", seiNalus: "x"}}
	h2 := &Init{CMAF: true, Handler: "vide", Video: videoConfig{codec: "hevc", spsSummary: "b", seiNalus: "x"}}
	rep.Profiles = nil
	f = cmafMediaProfileSetRules([]*addressing.Rep{rep, rep}, []*Init{h1, h2})
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-CMAF-HEVC-SWITCHING"}) {
		t.Errorf("HEVC switching: got %v", f)
	}
	s1 := &Init{CMAF: true, Handler: "subt", cmafBrand: "im1t"}
	s2 := &Init{CMAF: true, Handler: "subt", cmafBrand: "im1i"}
	f = cmafMediaProfileSetRules([]*addressing.Rep{rep, rep}, []*Init{s1, s2})
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-CMAF-SUBTITLEPROFILE"}) {
		t.Errorf("subtitles: got %v", f)
	}
}
