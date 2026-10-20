package segcheck

import (
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Rules for the Partial Segments of Segment Sequence Representations
// (ISO/IEC 23009-1 5.3.5.7).

var RuleSSRSAP = report.Register(report.Rule{
	ID: "SEG-SSR-SAP", Severity: report.Error,
	Desc:    "The Partial Segments that SegmentSequenceProperties@cadence selects must start with a SAP type of at most @sapType",
	SpecRef: "ISO/IEC 23009-1 5.3.5.7",
})

// ssrMediaRules checks a video Partial Segment against the
// SegmentSequenceProperties of its Representation. @cadence c selects the
// Partial Segments with sub-numbers 0, c, 2c and so on. Only @sapType 1 and
// 2, which need a sync sample, can be told from the samples. With
// @startWithSAP 1 or 2, SEG-SAP reports the first Partial Segment.
func ssrMediaRules(rep *addressing.Rep, init *Init, media *Media, loc report.Location) []report.Finding {
	if !media.Partial || media.FirstSync || init.Handler != "vide" {
		return nil
	}
	if media.SubNumber == 0 && (rep.StartWithSAP == 1 || rep.StartWithSAP == 2) {
		return nil
	}
	for _, p := range rep.SequenceProperties {
		cadence := uint64(max(p.Cadence, 1)) // an absent @cadence reads as 0; its default is 1
		if (p.SapType == 1 || p.SapType == 2) && media.SubNumber%cadence == 0 {
			return []report.Finding{RuleSSRSAP.New(loc, "Partial Segment %d of its sequence starts with a non-sync sample, but @sapType is %d with @cadence %d",
				media.SubNumber, p.SapType, cadence)}
		}
	}
	return nil
}
