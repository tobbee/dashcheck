package segcheck

import (
	"fmt"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// A convention for mfhd sequence numbers, checked only when asked for
// (Init.SequenceNumbers, -sequence-numbers): one number per segment, shared
// by all its chunks, increasing from segment to segment. Numbering segments
// rather than chunks keeps the 32-bit number from wrapping around within the
// lifetime of a live stream. Neither ISO/IEC 14496-12 8.8.5 nor CMAF requires
// it: CMAF 7.5.15 says the number need not be unique or increase.

var RuleSegSequenceNumber = report.Register(report.Rule{
	ID: "SEG-SEQUENCE-NUMBER", Severity: report.Warning,
	Desc: "The chunks of a segment should share one mfhd sequence_number, which should increase from segment to segment " +
		"(a convention checked with -sequence-numbers, not a CMAF requirement)",
})

// firstSequenceNumber returns the sequence number of the first fragment and,
// if another fragment of the segment has a different one, a description of it.
func firstSequenceNumber(mf *mp4.File) (first uint32, other string) {
	found := false
	for si, ms := range mf.Segments {
		for fi, frag := range ms.Fragments {
			if frag.Moof == nil || frag.Moof.Mfhd == nil {
				continue
			}
			n := frag.Moof.Mfhd.SequenceNumber
			switch {
			case !found:
				first, found = n, true
			case n != first && other == "":
				other = fmt.Sprintf("segment[%d]/fragment[%d] has %d", si, fi, n)
			}
		}
	}
	return first, other
}

func sequenceNumberMediaRules(init *Init, other string, first uint32, loc report.Location) []report.Finding {
	if !init.SequenceNumbers || other == "" {
		return nil
	}
	return []report.Finding{RuleSegSequenceNumber.New(loc, "the chunks have different sequence numbers: the first has %d, %s", first, other)}
}

// sequenceNumberContinuity checks that cur has a higher sequence number than
// prev, the segment before it, counting a wrap-around past 2^32-1 as higher.
func sequenceNumberContinuity(prev, cur *Media, init *Init, rep *addressing.Rep, seg addressing.Segment) []report.Finding {
	if !init.SequenceNumbers {
		return nil
	}
	if diff := cur.SequenceNumber - prev.SequenceNumber; diff != 0 && diff < 1<<31 {
		return nil
	}
	return []report.Finding{RuleSegSequenceNumber.New(rep.SegLoc(seg),
		"sequence number %d does not increase from %d in the previous segment", cur.SequenceNumber, prev.SequenceNumber)}
}
