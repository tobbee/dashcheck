package segcheck

import (
	"slices"
	"testing"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

func TestCMAFBrandInitRules(t *testing.T) {
	cases := []struct {
		name    string
		major   string
		minor   uint32
		brands  []string
		handler string
		elst    bool
		want    []string
	}{
		{"cmfc", "cmfc", 0, []string{"cmfc", "iso6"}, "vide", true, nil},
		{"cmfc minor version", "cmfc", 1, []string{"cmfc"}, "vide", false, []string{"SEG-CMAF-MINORVERSION"}},
		{"other major brand, any minor version", "iso6", 512, []string{"iso6", "cmfc"}, "vide", false, nil},
		{"cmf2 video with edit list", "cmf2", 0, []string{"cmf2", "cmfc"}, "vide", true, []string{"SEG-CMF2-ELST"}},
		{"cmf2 video", "cmf2", 0, []string{"cmf2", "cmfc"}, "vide", false, nil},
		{"cmf2 audio with edit list", "iso6", 0, []string{"iso6", "cmf2"}, "soun", true, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			init := &Init{CMAF: true, Brands: append([]string{c.major}, c.brands...), Handler: c.handler, HasEditList: c.elst}
			ftyp := mp4.NewFtyp(c.major, c.minor, c.brands)
			if got := ruleIDs(cmafBrandInitRules(init, ftyp, report.Location{})); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestMissingDefaults(t *testing.T) {
	const (
		tfhdAll     = mp4.TfhdDefaultSampleDurationPresentFlag | mp4.TfhdDefaultSampleSizePresentFlag | mp4.TfhdDefaultSampleFlagsPresentFlag
		tfhdDurSize = mp4.TfhdDefaultSampleDurationPresentFlag | mp4.TfhdDefaultSampleSizePresentFlag
		trunAll     = mp4.TrunSampleDurationPresentFlag | mp4.TrunSampleSizePresentFlag | mp4.TrunSampleFlagsPresentFlag
	)
	cases := []struct {
		name    string
		tfhd    uint32
		trun    uint32
		samples int
		want    []string
	}{
		{"all from trex", 0, 0, 2, []string{"duration", "size", "flags"}},
		{"tfhd defaults", tfhdAll, 0, 2, nil},
		{"per-sample values", 0, trunAll, 2, nil},
		{"first_sample_flags for a single sample", tfhdDurSize, mp4.TrunFirstSampleFlagsPresentFlag, 1, nil},
		{"first_sample_flags for several samples", tfhdDurSize, mp4.TrunFirstSampleFlagsPresentFlag, 2, []string{"flags"}},
		{"size from trex", mp4.TfhdDefaultSampleDurationPresentFlag, mp4.TrunSampleFlagsPresentFlag, 2, []string{"size"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			trun := &mp4.TrunBox{Flags: c.trun, Samples: make([]mp4.Sample, c.samples)}
			if got := missingDefaults(&mp4.TfhdBox{Flags: c.tfhd}, trun); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestCMF2Media checks the test content, which has trun version 0 and all
// sample values in trun, as if its CMAF header had the cmf2 brand.
func TestCMF2Media(t *testing.T) {
	rs := reps(t)
	cases := []struct {
		name string
		rep  addressing.Rep
		want []string
	}{
		{"video", *rs[1], []string{"SEG-CMF2-TRUN"}},
		{"audio", *rs[0], nil},
		{"video track file", *rs[1], nil},
	}
	cases[2].rep.Kind = addressing.KindSegmentBase
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rep := &c.rep
			init, f := CheckInit(read(t, rep.ID+"/init.mp4"), rep, rep.Loc())
			if init == nil {
				t.Fatalf("init: %v", f)
			}
			init.CMAF = true
			init.Brands = append(init.Brands, "cmf2")
			segs, _ := rs[0].Segments(0, rs[0].PeriodDur)
			_, f = CheckMedia(read(t, rep.ID+"/1.m4s"), init, rep, segs[0], false)
			var got []string
			for _, x := range f {
				if x.Rule == "SEG-CMF2-TRUN" || x.Rule == "SEG-CMF2-DEFAULTS" || x.Rule == "SEG-CMAF-MINORVERSION" {
					got = append(got, x.Rule)
				}
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}
