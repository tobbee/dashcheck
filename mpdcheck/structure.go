package mpdcheck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Rules ported from the generic ISO/IEC 23009-1 part of the DASH-IF
// Schematron rules (schematron.sch). The SCH numbers in the comments refer to
// docs/dashif-checks.json.

var (
	RuleDurationMissing = report.Register(report.Rule{
		ID: "MPD-DURATION-MISSING", Severity: report.Error,
		Desc:    "Without @minimumUpdatePeriod and a last Period@duration, MPD@mediaPresentationDuration must be present",
		SpecRef: refMPD,
	})
	RuleProfileUnknown = report.Register(report.Rule{
		ID: "MPD-PROFILE-UNKNOWN", Severity: report.Info,
		Desc: "Profiles unknown to dashcheck are not checked", SpecRef: "ISO/IEC 23009-1 8",
	})
	RuleBitstreamSwitching = report.Register(report.Rule{
		ID: "MPD-BITSTREAMSWITCHING", Severity: report.Error,
		Desc: "With Period@bitstreamSwitching true, no AdaptationSet may set it to false", SpecRef: refPeriod,
	})
	RuleSegInfoMultiple = report.Register(report.Rule{
		ID: "MPD-SEGINFO-MULTIPLE", Severity: report.Error,
		Desc: "At most one of SegmentBase, SegmentTemplate and SegmentList may be present on one element", SpecRef: "ISO/IEC 23009-1 5.3.9.1",
	})
	RuleSegInfoNone = report.Register(report.Rule{
		ID: "MPD-SEGINFO-NONE", Severity: report.Error,
		Desc: "A Period needs a BaseURL, SegmentTemplate or SegmentList at some level", SpecRef: "ISO/IEC 23009-1 5.3.9.1",
	})
	RuleXlinkUnresolved = report.Register(report.Rule{
		ID: "MPD-XLINK-UNRESOLVED", Severity: report.Info,
		Desc:    "Remote Periods (@xlink:href) are not resolved, so their content is not checked, nor the segments of a Period whose timing depends on them",
		SpecRef: "ISO/IEC 23009-1 5.5",
	})
	RuleXlinkStart = report.Register(report.Rule{
		ID: "MPD-XLINK-START", Severity: report.Warning,
		Desc:    "A Period after a remote Period should have @start, or the remote Period @duration, so that its start is known without resolving the remote Period",
		SpecRef: "ISO/IEC 23009-1 5.3.2.1",
	})
	RulePeriodZeroDuration = report.Register(report.Rule{
		ID: "MPD-PERIOD-ZERO", Severity: report.Warning,
		Desc: "A Period with @duration 0 should have at most one AdaptationSet", SpecRef: refPeriod,
	})
	RuleContentComponent = report.Register(report.Rule{
		ID: "MPD-CONTENTCOMPONENT", Severity: report.Error,
		Desc: "ContentComponent ids must be unique and must not repeat AdaptationSet attributes", SpecRef: "ISO/IEC 23009-1 5.3.4",
	})
	RuleCommonAttrBoth = report.Register(report.Rule{
		ID: "MPD-COMMON-BOTH", Severity: report.Warning,
		Desc: "Common attributes should be on the AdaptationSet or its Representations, not both", SpecRef: refCommon,
	})
	RuleASMinMax = report.Register(report.Rule{
		ID: "MPD-AS-MINMAX", Severity: report.Error,
		Desc: "AdaptationSet minimum values must not be larger than the maximum values", SpecRef: refAS,
	})
	RuleASRange = report.Register(report.Rule{
		ID: "MPD-AS-RANGE", Severity: report.Error,
		Desc:    "Representation bandwidth, width, height and frame rate must be within the AdaptationSet ranges",
		SpecRef: refAS,
	})
	RuleASNoRep = report.Register(report.Rule{
		ID: "MPD-AS-NOREP", Severity: report.Error,
		Desc: "An AdaptationSet must have at least one Representation", SpecRef: refAS,
	})
	RuleSubRepBandwidth = report.Register(report.Rule{
		ID: "MPD-SUBREP-BANDWIDTH", Severity: report.Error,
		Desc: "A SubRepresentation with @level must have @bandwidth", SpecRef: "ISO/IEC 23009-1 5.3.6",
	})
	RuleIndexRangeExact = report.Register(report.Rule{
		ID: "MPD-INDEXRANGEEXACT", Severity: report.Error,
		Desc: "@indexRangeExact must not be present without @indexRange", SpecRef: "ISO/IEC 23009-1 5.3.9.2",
	})
	RuleTemplateInit = report.Register(report.Rule{
		ID: "MPD-TEMPLATE-INIT", Severity: report.Error,
		Desc:    "SegmentTemplate @initialization and @bitstreamSwitching must not contain $Number$ or $Time$",
		SpecRef: refTemplate,
	})
	RuleSegBaseTSBD = report.Register(report.Rule{
		ID: "MPD-SEGBASE-TSBD", Severity: report.Error,
		Desc: "A segment-level @timeShiftBufferDepth must not be smaller than MPD@timeShiftBufferDepth", SpecRef: "ISO/IEC 23009-1 5.3.9.2",
	})
	RuleProgramInfoLang = report.Register(report.Rule{
		ID: "MPD-PROGINFO-LANG", Severity: report.Error,
		Desc: "With several ProgramInformation elements, each must have @lang", SpecRef: refMPD,
	})
	RuleRoleValue = report.Register(report.Rule{
		ID: "MPD-ROLE-VALUE", Severity: report.Error,
		Desc: "Role values must be valid for their scheme", SpecRef: "ISO/IEC 23009-1 5.8.5.5",
	})
	RuleEventStream = report.Register(report.Rule{
		ID: "MPD-EVENTSTREAM", Severity: report.Error,
		Desc: "EventStream must have @schemeIdUri, and @xlink:actuate only together with @xlink:href", SpecRef: "ISO/IEC 23009-1 5.10.2",
	})
	RuleSubsetID = report.Register(report.Rule{
		ID: "MPD-SUBSET-ID", Severity: report.Error,
		Desc: "Subset@id must be unique", SpecRef: "ISO/IEC 23009-1 5.3.8",
	})
	RuleChaining = report.Register(report.Rule{
		ID: "MPD-CHAINING", Severity: report.Error,
		Desc:    "Chaining and fallback descriptors need a @value, and fallback must be on the MPD level",
		SpecRef: "ISO/IEC 23009-1 5.11",
	})
)

// knownProfiles are the profiles listed in the DASH-IF Schematron rules,
// plus a few newer ones seen in practice.
var knownProfiles = map[string]bool{}

func init() {
	for _, p := range strings.Fields(`
urn:mpeg:dash:profile:isoff-on-demand:2011 urn:mpeg:dash:profile:isoff-ext-on-demand:2014
urn:mpeg:dash:profile:isoff-live:2011 urn:mpeg:dash:profile:isoff-ext-live:2014
urn:mpeg:dash:profile:isoff-main:2011 urn:mpeg:dash:profile:full:2011 urn:mpeg:dash:profile:isoff-common:2014
urn:mpeg:dash:profile:isoff-broadcast:2015 urn:mpeg:dash:profile:mp2t-main:2011 urn:mpeg:dash:profile:mp2t-simple:2011
urn:mpeg:dash:profile:cmaf:2019 urn:mpeg:dash:profile:advanced-linear:2025 urn:mpeg:dash:profile:list:2024
http://dashif.org/guidelines/dash264 http://dashif.org/guidelines/dash264#sd http://dashif.org/guidelines/dash264#hd
http://dashif.org/guidelines/dash264main http://dashif.org/guidelines/dash264high
http://dashif.org/guidelines/dash-if-simple http://dashif.org/guidelines/dash-if-main
http://dashif.org/guidelines/dash-if-ondemand http://dashif.org/guidelines/dash-if-mixed
http://dashif.org/guidelines/dashif#ec-3 http://dashif.org/guidelines/dashif#mlpa http://dashif.org/guidelines/dashif#ac-4
http://dashif.org/guidelines/dashif#dtsc http://dashif.org/guidelines/dashif#dtsh http://dashif.org/guidelines/dashif#dtse
http://dashif.org/guidelines/dashif#dtsl http://dashif.org/guidelines/dashif#mps http://dashif.org/guidelines/dashif#heaac-mc51
http://dashif.org/guidelines/dashif#heaac-mc71 http://dashif.org/guidelines/dashif#mpeg-h-3da http://dashif.org/guidelines/dashif#cxha
http://dashif.org/guidelines/dash-if-uhd#4k http://dashif.org/guidelines/dash-if-uhd#hdr-pq10 http://dashif.org/guidelines/dashif#vp9
http://dashif.org/guidelines/dash-if-uhd#vp9 http://dashif.org/guidelines/dashif#vp9-hdr http://dashif.org/guidelines/dash-if-uhd#vp9-hdr
http://www.dashif.org/guidelines/low-latency-live-v5
urn:dvb:dash:profile:dvb-dash:2014 urn:dvb:dash:profile:dvb-dash:2017
urn:dvb:dash:profile:dvb-dash:isoff-ext-live:2014 urn:dvb:dash:profile:dvb-dash:isoff-ext-on-demand:2014
urn:hbbtv:dash:profile:isoff-live:2012 urn:cta:wave:test-content-media-profile:2022`) {
		knownProfiles[p] = true
	}
}

// roleValues are the values of the urn:mpeg:dash:role:2011 scheme in the 6th edition.
// The DASH-IF Schematron rule only knows the first seven.
var roleValues = map[string]bool{
	"caption": true, "subtitle": true, "main": true, "alternate": true, "supplementary": true,
	"commentary": true, "dub": true, "description": true, "sign": true, "metadata": true,
	"enhanced-audio-intelligibility": true, "emergency": true, "forced-subtitle": true,
	"easyreader": true, "karaoke": true,
}

var numberOrTimeRe = regexp.MustCompile(`\$(Number|Time)(%[^$]*)?\$`)

// structureRules runs the generic Schematron rules.
func structureRules(md *m.MPD) []report.Finding {
	var f []report.Finding
	mpdLoc := report.Location{Path: "MPD"}
	cpSources := addressing.ContentProtectionSources(md)
	// MPD-IOP-CP requires cenc:default_KID on the AdaptationSets of IOP MPDs.
	iop := profiles(splitProfiles(string(md.Profiles))).iop()

	// SCH-3, SCH-6
	if md.MediaPresentationDuration == nil && md.MinimumUpdatePeriod == nil {
		n := len(md.Periods)
		if n == 0 || md.Periods[n-1].Duration == nil {
			f = append(f, RuleDurationMissing.New(mpdLoc,
				"no @mediaPresentationDuration, no @minimumUpdatePeriod and no @duration on the last Period"))
		}
	}
	// SCH-4
	for _, p := range splitProfiles(string(md.Profiles)) {
		if !knownProfiles[p] {
			f = append(f, RuleProfileUnknown.New(mpdLoc, "unknown profile %q", p))
		}
	}
	// SCH-54
	if len(md.ProgramInformation) > 1 {
		for i, pi := range md.ProgramInformation {
			if pi.Lang == "" {
				f = append(f, RuleProgramInfoLang.New(report.Location{Path: fmt.Sprintf("MPD/ProgramInformation[%d]", i)},
					"%d ProgramInformation elements but this one has no @lang", len(md.ProgramInformation)))
			}
		}
	}
	// SCH-68, SCH-70
	f = append(f, checkChaining(md.SupplementalProperties, "MPD", true)...)

	var mpdTSBD time.Duration
	if md.TimeShiftBufferDepth != nil {
		mpdTSBD = time.Duration(*md.TimeShiftBufferDepth)
	}
	subsetIDs := map[string]bool{}
	for pi, p := range md.Periods {
		pPath := fmt.Sprintf("Period[%d]", pi)
		pLoc := report.Location{Path: pPath}
		f = append(f, checkSegInfoCount(p.SegmentBase != nil, p.SegmentTemplate != nil, p.SegmentList != nil, pLoc)...)
		f = append(f, checkTemplateInit(p.SegmentTemplate, pLoc)...)
		f = append(f, checkSegInfoAttrs(p.SegmentBase, p.SegmentTemplate, p.SegmentList, mpdTSBD, pLoc)...)
		f = append(f, checkChaining(p.SupplementalProperties, pPath, false)...)
		// SCH-13. A remote Period is replaced by the element it references (5.5.3).
		switch {
		case p.XlinkHref == "urn:mpeg:dash:resolve-to-zero:2013":
		case p.XlinkHref != "":
			f = append(f, RuleXlinkUnresolved.New(pLoc, "remote Period %s is not resolved", p.XlinkHref))
		case !periodHasSegmentInfo(p):
			f = append(f, RuleSegInfoNone.New(pLoc, "no BaseURL, SegmentTemplate or SegmentList in the Period"))
		}
		if pi > 0 && p.Start == nil && addressing.IsRemote(md.Periods[pi-1]) && md.Periods[pi-1].Duration == nil {
			f = append(f, RuleXlinkStart.New(pLoc, "no @start, and the remote Period before it has no @duration"))
		}
		// SCH-14
		if p.Duration != nil && *p.Duration == 0 && len(p.AdaptationSets) > 1 {
			f = append(f, RulePeriodZeroDuration.New(pLoc, "@duration is 0 but there are %d AdaptationSets", len(p.AdaptationSets)))
		}
		// SCH-64, SCH-65
		for ei, es := range p.EventStreams {
			loc := report.Location{Path: fmt.Sprintf("%s/EventStream[%d]", pPath, ei)}
			if es.SchemeIdUri == "" {
				f = append(f, RuleEventStream.New(loc, "no @schemeIdUri"))
			}
			if es.XlinkActuate != "" && es.XlinkHref == "" {
				f = append(f, RuleEventStream.New(loc, "@xlink:actuate without @xlink:href"))
			}
		}
		// SCH-66
		for si, s := range p.Subsets {
			if s.Id == "" {
				continue
			}
			if subsetIDs[s.Id] {
				f = append(f, RuleSubsetID.New(report.Location{Path: fmt.Sprintf("%s/Subset[%d]", pPath, si)}, "Subset@id %q is not unique", s.Id))
			}
			subsetIDs[s.Id] = true
		}
		for ai, as := range p.AdaptationSets {
			asPath := fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)
			asLoc := report.Location{Path: asPath}
			// SCH-9
			if p.BitstreamSwitching != nil && *p.BitstreamSwitching && as.BitstreamSwitching != nil && !*as.BitstreamSwitching {
				f = append(f, RuleBitstreamSwitching.New(asLoc, "@bitstreamSwitching is false but Period@bitstreamSwitching is true"))
			}
			f = append(f, checkSegInfoCount(as.SegmentBase != nil, as.SegmentTemplate != nil, as.SegmentList != nil, asLoc)...)
			f = append(f, checkTemplateInit(as.SegmentTemplate, asLoc)...)
			f = append(f, checkSegInfoAttrs(as.SegmentBase, as.SegmentTemplate, as.SegmentList, mpdTSBD, asLoc)...)
			f = append(f, checkContentComponents(as, asLoc)...)
			f = append(f, checkCommonBoth(as, asLoc)...)
			f = append(f, checkRanges(as, asPath)...)
			f = append(f, checkDescriptors(addressing.ResolveContentProtections(as.ContentProtections, cpSources), as.Roles, asPath, iop)...)
			f = append(f, checkChaining(as.SupplementalProperties, asPath, false)...)
			// SCH-27
			if len(as.Representations) == 0 {
				f = append(f, RuleASNoRep.New(asLoc, "no Representation"))
			}
			for _, rep := range as.Representations {
				rPath := fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id)
				rLoc := report.Location{Path: rPath, RepID: rep.Id}
				f = append(f, checkSegInfoCount(rep.SegmentBase != nil, rep.SegmentTemplate != nil, rep.SegmentList != nil, rLoc)...)
				f = append(f, checkTemplateInit(rep.SegmentTemplate, rLoc)...)
				f = append(f, checkSegInfoAttrs(rep.SegmentBase, rep.SegmentTemplate, rep.SegmentList, mpdTSBD, rLoc)...)
				f = append(f, checkDescriptors(addressing.ResolveContentProtections(rep.ContentProtections, cpSources), nil, rPath, false)...)
				f = append(f, checkChaining(rep.SupplementalProperties, rPath, false)...)
				// SCH-40
				for si, sr := range rep.SubRepresentations {
					if sr.Level != nil && sr.Bandwidth == 0 {
						f = append(f, RuleSubRepBandwidth.New(report.Location{Path: fmt.Sprintf("%s/SubRepresentation[%d]", rPath, si), RepID: rep.Id},
							"@level without @bandwidth"))
					}
				}
			}
		}
	}
	return f
}

func splitProfiles(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// SCH-11, SCH-28, SCH-36
func checkSegInfoCount(base, tmpl, list bool, loc report.Location) []report.Finding {
	var present []string
	if base {
		present = append(present, "SegmentBase")
	}
	if tmpl {
		present = append(present, "SegmentTemplate")
	}
	if list {
		present = append(present, "SegmentList")
	}
	if len(present) > 1 {
		return []report.Finding{RuleSegInfoMultiple.New(loc, "has %s", strings.Join(present, " and "))}
	}
	return nil
}

// SCH-44, SCH-45
func checkTemplateInit(t *m.SegmentTemplateType, loc report.Location) []report.Finding {
	if t == nil {
		return nil
	}
	var f []report.Finding
	if numberOrTimeRe.MatchString(t.Initialization) {
		f = append(f, RuleTemplateInit.New(loc, "@initialization %q contains $Number$ or $Time$", t.Initialization))
	}
	if numberOrTimeRe.MatchString(t.BitstreamSwitching) {
		f = append(f, RuleTemplateInit.New(loc, "@bitstreamSwitching %q contains $Number$ or $Time$", t.BitstreamSwitching))
	}
	return f
}

// SCH-52 (timeShiftBufferDepth). The @indexRangeExact rules are in rawStructureRules.
func checkSegInfoAttrs(b *m.SegmentBaseType, t *m.SegmentTemplateType, l *m.SegmentListType, mpdTSBD time.Duration, loc report.Location) []report.Finding {
	var f []report.Finding
	check := func(name string, sb *m.SegmentBaseType) {
		if tsbd := time.Duration(sb.TimeShiftBufferDepth); tsbd > 0 && mpdTSBD > 0 && tsbd < mpdTSBD {
			f = append(f, RuleSegBaseTSBD.New(loc, "%s@timeShiftBufferDepth %v is smaller than MPD@timeShiftBufferDepth %v", name, tsbd, mpdTSBD))
		}
	}
	if b != nil {
		check("SegmentBase", b)
	}
	if t != nil {
		check("SegmentTemplate", &t.SegmentBaseType)
	}
	if l != nil {
		check("SegmentList", &l.SegmentBaseType)
	}
	return f
}

func periodHasSegmentInfo(p *m.Period) bool {
	if len(p.BaseURLs) > 0 || p.SegmentTemplate != nil || p.SegmentList != nil {
		return true
	}
	for _, as := range p.AdaptationSets {
		if len(as.BaseURLs) > 0 || as.SegmentTemplate != nil || as.SegmentList != nil {
			return true
		}
		for _, rep := range as.Representations {
			if len(rep.BaseURLs) > 0 || rep.SegmentTemplate != nil || rep.SegmentList != nil {
				return true
			}
		}
	}
	return false
}

// SCH-21, SCH-33
func checkContentComponents(as *m.AdaptationSetType, loc report.Location) []report.Finding {
	var f []report.Finding
	ids := map[uint32]bool{}
	for i, cc := range as.ContentComponents {
		ccLoc := loc
		ccLoc.Path = fmt.Sprintf("%s/ContentComponent[%d]", loc.Path, i)
		if cc.Id != nil {
			if ids[*cc.Id] {
				f = append(f, RuleContentComponent.New(ccLoc, "ContentComponent@id %d is not unique", *cc.Id))
			}
			ids[*cc.Id] = true
		}
		switch {
		case as.Lang != "" && cc.Lang == as.Lang:
			f = append(f, RuleContentComponent.New(ccLoc, "@lang repeats the AdaptationSet value"))
		case as.ContentType != "" && cc.ContentType == as.ContentType:
			f = append(f, RuleContentComponent.New(ccLoc, "@contentType repeats the AdaptationSet value"))
		case as.Par != "" && cc.Par == as.Par:
			f = append(f, RuleContentComponent.New(ccLoc, "@par repeats the AdaptationSet value"))
		}
	}
	return f
}

// SCH-22
func checkCommonBoth(as *m.AdaptationSetType, loc report.Location) []report.Finding {
	a := &as.RepresentationBaseType
	type attr struct {
		name  string
		onAS  bool
		onRep func(r *m.RepresentationBaseType) bool
	}
	attrs := []attr{
		{"profiles", a.Profiles != "", func(r *m.RepresentationBaseType) bool { return r.Profiles != "" }},
		{"width", a.Width != 0, func(r *m.RepresentationBaseType) bool { return r.Width != 0 }},
		{"height", a.Height != 0, func(r *m.RepresentationBaseType) bool { return r.Height != 0 }},
		{"sar", a.Sar != "", func(r *m.RepresentationBaseType) bool { return r.Sar != "" }},
		{"frameRate", a.FrameRate != "", func(r *m.RepresentationBaseType) bool { return r.FrameRate != "" }},
		{"audioSamplingRate", a.AudioSamplingRate != nil, func(r *m.RepresentationBaseType) bool { return r.AudioSamplingRate != nil }},
		{"mimeType", a.MimeType != "", func(r *m.RepresentationBaseType) bool { return r.MimeType != "" }},
		{"segmentProfiles", a.SegmentProfiles != nil, func(r *m.RepresentationBaseType) bool { return r.SegmentProfiles != nil }},
		{"codecs", a.Codecs != "", func(r *m.RepresentationBaseType) bool { return r.Codecs != "" }},
		{"maximumSAPPeriod", a.MaximumSAPPeriod != 0, func(r *m.RepresentationBaseType) bool { return r.MaximumSAPPeriod != 0 }},
		{"startWithSAP", a.StartWithSAP != 0, func(r *m.RepresentationBaseType) bool { return r.StartWithSAP != 0 }},
		{"maxPlayoutRate", a.MaxPlayoutRate != 0, func(r *m.RepresentationBaseType) bool { return r.MaxPlayoutRate != 0 }},
		{"codingDependency", a.CodingDependency != nil, func(r *m.RepresentationBaseType) bool { return r.CodingDependency != nil }},
		{"scanType", a.ScanType != "", func(r *m.RepresentationBaseType) bool { return r.ScanType != "" }},
	}
	var both []string
	for _, at := range attrs {
		if !at.onAS {
			continue
		}
		for _, rep := range as.Representations {
			if at.onRep(&rep.RepresentationBaseType) {
				both = append(both, "@"+at.name)
				break
			}
		}
	}
	if len(both) > 0 {
		return []report.Finding{RuleCommonAttrBoth.New(loc, "%s on both the AdaptationSet and a Representation", strings.Join(both, ", "))}
	}
	return nil
}

// SCH-23 to SCH-26, SCH-29
func checkRanges(as *m.AdaptationSetType, asPath string) []report.Finding {
	var f []report.Finding
	loc := report.Location{Path: asPath}
	minMax := []struct {
		name     string
		min, max uint32
	}{
		{"Width", as.MinWidth, as.MaxWidth},
		{"Height", as.MinHeight, as.MaxHeight},
		{"Bandwidth", as.MinBandwidth, as.MaxBandwidth},
	}
	for _, mm := range minMax {
		if mm.min != 0 && mm.max != 0 && mm.min > mm.max {
			f = append(f, RuleASMinMax.New(loc, "@min%s %d is larger than @max%s %d", mm.name, mm.min, mm.name, mm.max))
		}
	}
	minFR, okMin := parseFrameRate(as.MinFrameRate)
	maxFR, okMax := parseFrameRate(as.MaxFrameRate)
	for _, rep := range as.Representations {
		rLoc := report.Location{Path: fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id), RepID: rep.Id}
		outside := func(name string, v, lo, hi uint32) {
			if v == 0 {
				return
			}
			if (lo != 0 && v < lo) || (hi != 0 && v > hi) {
				f = append(f, RuleASRange.New(rLoc, "@%s %d is outside the AdaptationSet range [%d, %d]", name, v, lo, hi))
			}
		}
		outside("bandwidth", rep.Bandwidth, as.MinBandwidth, as.MaxBandwidth)
		outside("width", rep.Width, as.MinWidth, as.MaxWidth)
		outside("height", rep.Height, as.MinHeight, as.MaxHeight)
		if fr, ok := parseFrameRate(string(rep.FrameRate)); ok {
			if (okMin && fr < minFR) || (okMax && fr > maxFR) {
				f = append(f, RuleASRange.New(rLoc, "@frameRate %s is outside the AdaptationSet range [%s, %s]",
					rep.FrameRate, as.MinFrameRate, as.MaxFrameRate))
			}
		}
	}
	return f
}

// parseFrameRate parses "30" or "30000/1001".
func parseFrameRate(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	num, den, hasDen := strings.Cut(s, "/")
	n, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, false
	}
	if !hasDen {
		return n, true
	}
	d, err := strconv.ParseFloat(den, 64)
	if err != nil || d == 0 {
		return 0, false
	}
	return n / d, true
}

// SCH-55 to SCH-58. cps are the ContentProtection elements with @ref
// resolved; for kidRequired see checkContentProtectionValues.
func checkDescriptors(cps []*m.ContentProtectionType, roles []*m.DescriptorType, path string, kidRequired bool) []report.Finding {
	f := checkContentProtectionValues(cps, path, kidRequired)
	for i, r := range roles {
		loc := report.Location{Path: fmt.Sprintf("%s/Role[%d]", path, i)}
		switch r.SchemeIdUri {
		case "urn:mpeg:dash:role:2011":
			if !roleValues[r.Value] {
				f = append(f, RuleRoleValue.New(loc, "%q is not a value of urn:mpeg:dash:role:2011", r.Value))
			}
		case "urn:mpeg:dash:stereoid:2011":
			if !strings.HasPrefix(r.Value, "l") && !strings.HasPrefix(r.Value, "r") {
				f = append(f, RuleRoleValue.New(loc, "stereoid value %q does not start with l or r", r.Value))
			}
		}
	}
	return f
}

// SCH-68 to SCH-70
func checkChaining(props []*m.DescriptorType, path string, mpdLevel bool) []report.Finding {
	var f []report.Finding
	for i, p := range props {
		loc := report.Location{Path: fmt.Sprintf("%s/SupplementalProperty[%d]", path, i)}
		switch p.SchemeIdUri {
		case "urn:mpeg:dash:chaining:2016":
			if strings.TrimSpace(p.Value) == "" {
				f = append(f, RuleChaining.New(loc, "chaining descriptor without @value"))
			}
		case "urn:mpeg:dash:fallback:2016":
			if !mpdLevel {
				f = append(f, RuleChaining.New(loc, "fallback descriptor must be on the MPD level"))
			}
			if strings.TrimSpace(p.Value) == "" {
				f = append(f, RuleChaining.New(loc, "fallback descriptor without @value"))
			}
		}
	}
	return f
}

// rawStructureRules runs rules that need the attributes as written.
func rawStructureRules(raw *rawElem) []report.Finding {
	if raw == nil {
		return nil
	}
	var f []report.Finding
	raw.walk(func(e *rawElem, _ []*rawElem) {
		switch e.Name {
		case "SegmentBase", "SegmentTemplate", "SegmentList":
			// SCH-43, SCH-50, SCH-51
			if e.has("indexRangeExact") && !e.has("indexRange") {
				f = append(f, RuleIndexRangeExact.New(report.Location{Line: e.Line, Path: e.Name},
					"%s@indexRangeExact without @indexRange", e.Name))
			}
		}
	})
	return f
}
