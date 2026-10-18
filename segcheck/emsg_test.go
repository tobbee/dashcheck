package segcheck

import (
	"bytes"
	"slices"
	"testing"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
)

// emsgSegment encodes a two-fragment segment with event messages with the
// given ids before the first and second moof, and decodes it again.
func emsgSegment(t *testing.T, first, second []uint32) *mp4.File {
	t.Helper()
	seg := mp4.NewMediaSegment()
	for i, ids := range [][]uint32{first, second} {
		frag, err := mp4.CreateFragment(uint32(i+1), 1)
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range ids {
			frag.AddEmsg(&mp4.EmsgBox{Version: 1, TimeScale: 1000, ID: id, SchemeIDURI: "urn:example:event", Value: "1"})
		}
		frag.AddFullSample(mp4.FullSample{Sample: mp4.Sample{Dur: 1000, Size: 1}, DecodeTime: uint64(i * 1000), Data: []byte{0}})
		seg.AddFragment(frag)
	}
	var buf bytes.Buffer
	if err := seg.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	mf, err := mp4.DecodeFile(&buf)
	if err != nil {
		t.Fatal(err)
	}
	return mf
}

func TestEmsgRepeat(t *testing.T) {
	early, late := emsgPlacement(emsgSegment(t, []uint32{1}, []uint32{2}))
	if !slices.Equal(early, []string{"urn:example:event 1 1"}) || !slices.Equal(late, []string{"urn:example:event 1 2"}) {
		t.Fatalf("placement: early %v, late %v", early, late)
	}
	_, video := dvbReps(t)
	video.LowLatency = true
	prev := &Media{LateEmsgs: late}
	repeated := &Media{EarlyEmsgs: []string{"urn:example:event 1 2"}}
	if f := emsgRepeatRules(prev, repeated, &Init{}, &video, addressing.Segment{}); len(f) > 0 {
		t.Errorf("repeated: %v", f)
	}
	if got := ruleIDs(emsgRepeatRules(prev, &Media{}, &Init{}, &video, addressing.Segment{})); !slices.Equal(got, []string{"SEG-EMSG-REPEAT"}) {
		t.Errorf("not repeated: got %v", got)
	}
	video.LowLatency = false
	if f := emsgRepeatRules(prev, &Media{}, &Init{}, &video, addressing.Segment{}); len(f) > 0 {
		t.Errorf("not low latency: %v", f)
	}
}
