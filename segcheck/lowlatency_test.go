package segcheck

import (
	"slices"
	"testing"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
)

func TestLowLatencySegments(t *testing.T) {
	video := *reps(t)[1]
	video.LowLatency = true
	init, f := CheckInit(read(t, "V300/init.mp4"), &video, video.Loc())
	// The test MPD has @timescale 1 but the track has 90000.
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-LL-TIMESCALE"}) {
		t.Errorf("init: got %v", f)
	}
	segs, _ := video.Segments(0, video.PeriodDur)
	data := read(t, "V300/1.m4s")
	cases := []struct {
		name   string
		modify func(r *addressing.Rep)
		want   []string
	}{
		{"segment mode, long target", func(r *addressing.Rep) { r.TargetLatency = 10 }, nil},
		{"segment mode, 2 s of 5 s", func(r *addressing.Rep) { r.TargetLatency = 5 }, []string{"SEG-LL-DURATION-RECOMMENDED"}},
		{"segment mode, 2 s of 3 s", func(r *addressing.Rep) { r.TargetLatency = 3 }, []string{"SEG-LL-DURATION"}},
		{"chunked with one chunk", func(r *addressing.Rep) { r.ATO = 1.5 }, []string{"SEG-LL-CHUNKS"}},
		{"chunked, 2 s of 1.5 s", func(r *addressing.Rep) { r.ATO, r.TargetLatency = 1.5, 1.5 }, []string{"SEG-LL-CHUNKS", "SEG-LL-DURATION-RECOMMENDED"}},
	}
	for _, c := range cases {
		r := video
		c.modify(&r)
		_, f := CheckMedia(data, init, &r, segs[0], false)
		var got []string
		for _, x := range ruleIDs(f) {
			if x != "SEG-EPT" {
				got = append(got, x)
			}
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v", c.name, f)
		}
	}
}

func TestSelfInitSidx(t *testing.T) {
	rep := &addressing.Rep{ID: "v", LowLatency: true, Kind: addressing.KindSegmentBase}
	init := &Init{TrackID: 1, Timescale: 1000, Brands: []string{"iso6", "dash"}}
	good := &mp4.SidxBox{ReferenceID: 1, Timescale: 1000, SidxRefs: []mp4.SidxRef{{StartsWithSAP: 1, SAPType: 1}}}
	first := &Media{DecodeTime: 0, EPT: 0}
	if f := CheckSelfInitSidx(rep, init, good, 1, true, first); len(f) > 0 {
		t.Errorf("good: %v", f)
	}
	bad := &mp4.SidxBox{ReferenceID: 2, Timescale: 90000, EarliestPresentationTime: 5,
		SidxRefs: []mp4.SidxRef{{StartsWithSAP: 1, SAPType: 3}}}
	f := CheckSelfInitSidx(rep, &Init{TrackID: 1, Timescale: 1000}, bad, 2, true, &Media{DecodeTime: 10})
	// count, reference_ID, timescale, SAP type, decode time, EPT, dash brand
	if len(f) != 7 {
		t.Errorf("bad: got %d findings: %v", len(f), f)
	}
}
