package segcheck

import (
	"slices"
	"strings"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Segment checks ported from the Low Latency module of the DASH-IF
// Conformance tool (DASH-IF low-latency modes, 9.X.4).

const refLL = "DASH-IF Low-Latency Modes 9.X.4"

var (
	RuleLLChunks = report.Register(report.Rule{
		ID: "SEG-LL-CHUNKS", Severity: report.Warning,
		Desc: "Segments of chunked low-latency Representations should contain more than one chunk", SpecRef: refLL + ".5",
	})
	RuleLLSingleMoof = report.Register(report.Rule{
		ID: "SEG-LL-SINGLEMOOF", Severity: report.Warning,
		Desc: "Segments of low-latency segment-mode Representations should have a single moof box", SpecRef: refLL + ".4",
	})
	RuleLLSmds = report.Register(report.Rule{
		ID: "SEG-LL-SMDS", Severity: report.Error,
		Desc: "Segments with the smds brand need smds in @segmentProfiles", SpecRef: refLL + ".4",
	})
	RuleLLDuration = report.Register(report.Rule{
		ID: "SEG-LL-DURATION", Severity: report.Error,
		Desc:    "Low-latency segments must not exceed 50% of the target latency in segment mode",
		SpecRef: refLL + ".4",
	})
	RuleLLDurationRecommended = report.Register(report.Rule{
		ID: "SEG-LL-DURATION-RECOMMENDED", Severity: report.Warning,
		Desc:    "Low-latency segments should not exceed 30% of the target latency in segment mode, or the target latency when chunked",
		SpecRef: refLL + ".4",
	})
	RuleLLProtection = report.Register(report.Rule{
		ID: "SEG-LL-PROTECTION", Severity: report.Error,
		Desc: "Encrypted low-latency content must have a ContentProtection element", SpecRef: refLL + ".5",
	})
	RuleLLTimescale = report.Register(report.Rule{
		ID: "SEG-LL-TIMESCALE", Severity: report.Error,
		Desc: "The SegmentTemplate @timescale of low-latency content must equal the mdhd timescale", SpecRef: refLL + ".5",
	})
	RuleLLSidx = report.Register(report.Rule{
		ID: "SEG-LL-SIDX", Severity: report.Error,
		Desc:    "A self-initializing low-latency segment needs one sidx before the first moof that matches the track",
		SpecRef: refLL + ".5",
	})
)

// llChunked reports whether a Representation is low-latency chunked: with
// @availabilityTimeOffset or @availabilityTimeComplete false (9.X.4.5).
func llChunked(rep *addressing.Rep) bool {
	return rep.LowLatency && (rep.ATO > 0 || (rep.ATC != nil && !*rep.ATC))
}

func llInitRules(rep *addressing.Rep, init *Init, loc report.Location) []report.Finding {
	if !rep.LowLatency {
		return nil
	}
	var f []report.Finding
	// LL-24
	if init.Encrypted && !rep.HasContentProtection {
		f = append(f, RuleLLProtection.New(loc, "encrypted track without ContentProtection in the MPD"))
	}
	// LL-25
	switch rep.Kind {
	case addressing.KindNumber, addressing.KindTimelineTime, addressing.KindTimelineNumber:
		if rep.Timescale != init.Timescale {
			f = append(f, RuleLLTimescale.New(loc, "@timescale %d but mdhd timescale %d", rep.Timescale, init.Timescale))
		}
	}
	return f
}

func llMediaRules(rep *addressing.Rep, media *Media, mf *mp4.File, mediaTS float64, last bool, loc report.Location) []report.Finding {
	if !rep.LowLatency || media == nil {
		return nil
	}
	var f []report.Finding
	moofs := 0
	var brands []string
	for _, ms := range mf.Segments {
		moofs += len(ms.Fragments)
		if ms.Styp != nil && brands == nil {
			brands = append([]string{ms.Styp.MajorBrand()}, ms.Styp.CompatibleBrands()...)
		}
	}
	dur := float64(media.Duration) / mediaTS
	target := rep.TargetLatency
	if llChunked(rep) {
		// LL-14
		if moofs < 2 && !last {
			f = append(f, RuleLLChunks.New(loc, "chunked segment with %d chunk", moofs))
		}
		// LL-18: segments within the target latency are one of two options; the
		// other is Resync points, so a longer segment is only a warning.
		if target > 0 && dur > target {
			f = append(f, RuleLLDurationRecommended.New(loc, "%.3fs chunked segment exceeds the target latency %.3fs", dur, target))
		}
		return f
	}
	// LL-34
	if moofs != 1 {
		f = append(f, RuleLLSingleMoof.New(loc, "segment with %d moof boxes", moofs))
	}
	// LL-35
	if slices.Contains(brands, "smds") && !strings.Contains(rep.SegmentProfiles, "smds") {
		f = append(f, RuleLLSmds.New(loc, "smds brand but @segmentProfiles %q", rep.SegmentProfiles))
	}
	// LL-36 (upstream inverts both limits)
	if target > 0 {
		if dur > target/2 {
			f = append(f, RuleLLDuration.New(loc, "%.3fs segment exceeds 50%% of the target latency %.3fs", dur, target))
		} else if dur > 0.3*target {
			f = append(f, RuleLLDurationRecommended.New(loc, "%.3fs segment exceeds 30%% of the target latency %.3fs", dur, target))
		}
	}
	return f
}

// CheckSelfInitSidx checks the sidx box of a low-latency SegmentBase
// Representation (LL-43 to LL-54). first is the first checked subsegment.
func CheckSelfInitSidx(rep *addressing.Rep, init *Init, sidx *mp4.SidxBox, nSidx int, sidxBeforeMoof bool, first *Media) []report.Finding {
	if !rep.LowLatency || rep.Kind != addressing.KindSegmentBase || sidx == nil || init == nil {
		return nil
	}
	var f []report.Finding
	loc := rep.Loc()
	add := func(format string, args ...any) { f = append(f, RuleLLSidx.New(loc, format, args...)) }
	if nSidx != 1 {
		add("%d sidx boxes", nSidx)
	}
	if !sidxBeforeMoof {
		add("sidx box not before the first moof")
	}
	if sidx.ReferenceID != init.TrackID {
		add("sidx reference_ID %d, track ID %d", sidx.ReferenceID, init.TrackID)
	}
	if sidx.Timescale != init.Timescale {
		add("sidx timescale %d, mdhd timescale %d", sidx.Timescale, init.Timescale)
	}
	for i, ref := range sidx.SidxRefs {
		switch {
		case ref.ReferenceType != 0:
			add("reference %d points to a sidx box", i)
		case ref.StartsWithSAP != 1:
			add("reference %d does not start with a SAP", i)
		case ref.SAPType != 1 && ref.SAPType != 2:
			add("reference %d has SAP type %d, not 1 or 2", i, ref.SAPType)
		case ref.SAPDeltaTime != 0:
			add("reference %d has SAP_delta_time %d, not 0", i, ref.SAPDeltaTime)
		default:
			continue
		}
		break
	}
	if first != nil {
		if first.DecodeTime != 0 {
			add("first baseMediaDecodeTime is %d, not 0", first.DecodeTime)
		}
		if ept := uint64(first.EPT*float64(init.Timescale) + 0.5); ept != sidx.EarliestPresentationTime {
			add("sidx earliest_presentation_time %d, segment starts at %d", sidx.EarliestPresentationTime, ept)
		}
	}
	if !slices.Contains(init.Brands, "dash") {
		add("no dash brand in ftyp")
	}
	return f
}
