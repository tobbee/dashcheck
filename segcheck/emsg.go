package segcheck

import (
	"fmt"
	"slices"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

var RuleEmsgRepeat = report.Register(report.Rule{
	ID: "SEG-EMSG-REPEAT", Severity: report.Error,
	Desc: "An emsg box after the first moof of a segment must be repeated before the first moof of the next " +
		"segment (low-latency chunked and DASH-HLS interoperable content)",
	SpecRef: "DASH-IF IOP low latency 9.X.4.5 6b, CTA-5005-B 4.5.2",
})

// emsgPlacement lists the event messages of a media segment before its first
// moof and after it. mp4ff attaches an emsg to the fragment of the moof that
// follows it, and trailing ones to the last fragment.
func emsgPlacement(mf *mp4.File) (early, late []string) {
	first := true
	for _, ms := range mf.Segments {
		for _, frag := range ms.Fragments {
			seenMoof := false
			for _, c := range frag.Children {
				switch b := c.(type) {
				case *mp4.MoofBox:
					seenMoof = true
				case *mp4.EmsgBox:
					key := fmt.Sprintf("%s %s %d", b.SchemeIDURI, b.Value, b.ID)
					if first && !seenMoof {
						early = append(early, key)
					} else {
						late = append(late, key)
					}
				}
			}
			first = false
		}
	}
	return early, late
}

// emsgRepeatRules checks that event messages inserted after the first chunk of
// a segment are repeated at the start of the next one (LL-13, LL-28,
// WAVEHLS-17).
func emsgRepeatRules(prev, cur *Media, init *Init, rep *addressing.Rep, seg addressing.Segment) []report.Finding {
	if !rep.LowLatency && !init.HLSInterop {
		return nil
	}
	for _, key := range prev.LateEmsgs {
		if !slices.Contains(cur.EarlyEmsgs, key) {
			return []report.Finding{RuleEmsgRepeat.New(rep.SegLoc(seg),
				"emsg %q inside the previous segment is not repeated before the first moof", key)}
		}
	}
	return nil
}
