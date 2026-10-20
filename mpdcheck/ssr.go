package mpdcheck

import (
	"fmt"
	"slices"
	"strings"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// MPD rules for Segment Sequences and Segment Sequence Representations
// (ISO/IEC 23009-1 5.3.5.7, 5.3.9.7).

const schemeSSR = "urn:mpeg:dash:ssr:2023"

// profilesWithSequences are the profiles that use Segment Sequences: the
// broadcast TV profile (ISO/IEC 23009-1 8.11), the CMAF profile with
// addressable chunks (8.12.4), the advanced linear profile (8.13) and DASH-IF
// low latency with addressable chunks.
var profilesWithSequences = []string{
	"urn:mpeg:dash:profile:isoff-broadcast:2015",
	"urn:mpeg:dash:profile:cmaf:2019",
	"urn:mpeg:dash:profile:advanced-linear:2025",
	profileLowLatency,
}

var (
	RuleSSRSignal = report.Register(report.Rule{
		ID: "MPD-SSR-SIGNAL", Severity: report.Error,
		Desc:    "Segment Sequences need a profile that allows them or an EssentialProperty with urn:mpeg:dash:ssr:2023",
		SpecRef: "ISO/IEC 23009-1 5.3.5.7, 5.3.9.7.1",
	})
	RuleSSRTimeline = report.Register(report.Rule{
		ID: "MPD-SSR-TIMELINE", Severity: report.Error,
		Desc:    "If one S element of a SegmentTimeline has @k or @ssp, all of them must have it",
		SpecRef: "ISO/IEC 23009-1 5.3.9.7.1",
	})
	RuleSSRAttributes = report.Register(report.Rule{
		ID: "MPD-SSR-ATTRIBUTES", Severity: report.Error,
		Desc:    "@k must be positive, S@k needs S@d, and @endSubNumber needs @k and no SegmentTimeline",
		SpecRef: "ISO/IEC 23009-1 5.3.9.2.2, 5.3.9.6.3",
	})
	RuleSSRProperties = report.Register(report.Rule{
		ID: "MPD-SSR-PROPERTIES", Severity: report.Error,
		Desc:    "SegmentSequenceProperties@cadence must be greater than 0, and @alignment must list Representation@id values",
		SpecRef: "ISO/IEC 23009-1 5.3.5.7",
	})
)

// ssrRules checks the signalling and syntax of Segment Sequences.
func ssrRules(md *m.MPD, raw *rawElem) []report.Finding {
	var f []report.Finding
	reps, _ := addressing.Reps(md, "")
	for _, r := range reps {
		if r.SegmentSequences && !slices.ContainsFunc(profilesWithSequences, r.HasProfile) &&
			!slices.ContainsFunc(r.Essentials, func(d *m.DescriptorType) bool { return d.SchemeIdUri == schemeSSR }) {
			f = append(f, RuleSSRSignal.New(r.Loc(), "Segment Sequences without a profile that allows them or an EssentialProperty with %s", schemeSSR))
		}
	}
	ids := map[string]bool{}
	for pi, p := range md.Periods {
		pPath := fmt.Sprintf("Period[%d]", pi)
		f = append(f, ssrTemplateRules(p.SegmentTemplate, p.SegmentList, pPath)...)
		for ai, as := range p.AdaptationSets {
			asPath := fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)
			f = append(f, ssrTemplateRules(as.SegmentTemplate, as.SegmentList, asPath)...)
			for _, rep := range as.Representations {
				ids[rep.Id] = true
				f = append(f, ssrTemplateRules(rep.SegmentTemplate, rep.SegmentList, fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id))...)
			}
		}
	}
	return append(f, ssrPropertyRules(raw, ids)...)
}

// ssrTemplateRules checks the Segment Sequence attributes of the
// SegmentTemplate and SegmentList of one element.
func ssrTemplateRules(t *m.SegmentTemplateType, l *m.SegmentListType, path string) []report.Finding {
	var f []report.Finding
	var bases []*m.MultipleSegmentBaseType
	if t != nil {
		bases = append(bases, &t.MultipleSegmentBaseType)
	}
	if l != nil {
		bases = append(bases, &l.MultipleSegmentBaseType)
	}
	for _, b := range bases {
		loc := report.Location{Path: path}
		if b.K != nil && *b.K == 0 {
			f = append(f, RuleSSRAttributes.New(loc, "@k 0"))
		}
		if b.EndSubNumber != nil && (b.K == nil || b.SegmentTimeline != nil) {
			f = append(f, RuleSSRAttributes.New(loc, "@endSubNumber without @k, or with a SegmentTimeline"))
		}
		if b.SegmentTimeline == nil {
			continue
		}
		var withK, withSsp int
		for i, s := range b.SegmentTimeline.S {
			if s.K != nil {
				withK++
				switch {
				case *s.K == 0:
					f = append(f, RuleSSRAttributes.New(loc, "SegmentTimeline S[%d] has @k 0", i))
				case s.D == 0:
					f = append(f, RuleSSRAttributes.New(loc, "SegmentTimeline S[%d] has @k but no @d", i))
				}
			}
			if s.Ssp != nil {
				withSsp++
			}
		}
		n := len(b.SegmentTimeline.S)
		if (withK > 0 && withK < n) || (withSsp > 0 && withSsp < n) {
			f = append(f, RuleSSRTimeline.New(loc, "%d of %d S elements have @k and %d have @ssp", withK, n, withSsp))
		}
	}
	return f
}

// ssrPropertyRules checks SegmentSequenceProperties elements as written,
// since an absent @cadence (default 1) reads as 0 in the MPD model.
func ssrPropertyRules(raw *rawElem, ids map[string]bool) []report.Finding {
	if raw == nil {
		return nil
	}
	var f []report.Finding
	raw.walk(func(e *rawElem, _ []*rawElem) {
		if e.Name != "SegmentSequenceProperties" {
			return
		}
		loc := report.Location{Line: e.Line, Path: "SegmentSequenceProperties"}
		if e.has("cadence") && strings.TrimSpace(e.attr("cadence")) == "0" {
			f = append(f, RuleSSRProperties.New(loc, "@cadence 0"))
		}
		for id := range strings.FieldsSeq(e.attr("alignment")) {
			if !ids[id] {
				f = append(f, RuleSSRProperties.New(loc, "@alignment refers to %q, which is not a Representation@id", id))
			}
		}
	})
	return f
}
