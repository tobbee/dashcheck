// Package mpdcheck validates an MPD: XML well-formedness and schema,
// followed by semantic rules on the typed MPD model from dash-mpd.
package mpdcheck

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/report"
	"github.com/tobbee/dashcheck/schema"
)

const (
	refMPD      = "ISO/IEC 23009-1 5.3.1"
	refPeriod   = "ISO/IEC 23009-1 5.3.2"
	refAS       = "ISO/IEC 23009-1 5.3.3"
	refRep      = "ISO/IEC 23009-1 5.3.5"
	refCommon   = "ISO/IEC 23009-1 5.3.7"
	refTemplate = "ISO/IEC 23009-1 5.3.9.4"
	refTimeline = "ISO/IEC 23009-1 5.3.9.6"
)

var (
	RuleParse = report.Register(report.Rule{
		ID: "MPD-PARSE", Severity: report.Error,
		Desc: "The MPD must be parseable into the MPD data model",
	})
	RuleDynAST = report.Register(report.Rule{
		ID: "MPD-DYN-AST", Severity: report.Error,
		Desc: "A dynamic MPD must have @availabilityStartTime", SpecRef: refMPD,
	})
	RuleDynPublishTime = report.Register(report.Rule{
		ID: "MPD-DYN-PUBLISHTIME", Severity: report.Error,
		Desc: "A dynamic MPD must have @publishTime", SpecRef: refMPD,
	})
	RuleDynTSBD = report.Register(report.Rule{
		ID: "MPD-DYN-TSBD", Severity: report.Info,
		Desc: "A dynamic MPD without @timeShiftBufferDepth has an infinite time-shift buffer", SpecRef: refMPD,
	})
	RuleDynUTCTiming = report.Register(report.Rule{
		ID: "MPD-DYN-UTCTIMING", Severity: report.Warning,
		Desc: "A dynamic MPD should have at least one UTCTiming element", SpecRef: "DASH-IF Timing Model 13.1",
	})
	RuleUTCTimingScheme = report.Register(report.Rule{
		ID: "MPD-UTCTIMING-SCHEME", Severity: report.Warning,
		Desc:    "UTCTiming should use http-xsdate, http-iso, http-head or direct (2014 schemes)",
		SpecRef: "DASH-IF Timing Model 13.1",
	})
	RulePeriodID = report.Register(report.Rule{
		ID: "MPD-PERIOD-ID", Severity: report.Error,
		Desc: "Period@id must be present in dynamic MPDs and unique within the MPD", SpecRef: refPeriod,
	})
	RulePeriodOrder = report.Register(report.Rule{
		ID: "MPD-PERIOD-ORDER", Severity: report.Error,
		Desc: "Periods must be ordered by increasing start time", SpecRef: refPeriod,
	})
	RuleASID = report.Register(report.Rule{
		ID: "MPD-AS-ID", Severity: report.Error,
		Desc: "AdaptationSet@id must be unique within a Period", SpecRef: refAS,
	})
	RuleRepID = report.Register(report.Rule{
		ID: "MPD-REP-ID", Severity: report.Error,
		Desc: "Representation@id must be unique within a Period", SpecRef: refRep,
	})
	RuleMimeType = report.Register(report.Rule{
		ID: "MPD-MIMETYPE", Severity: report.Error,
		Desc: "@mimeType must be present on the AdaptationSet or the Representation", SpecRef: refCommon,
	})
	RuleCodecs = report.Register(report.Rule{
		ID: "MPD-CODECS", Severity: report.Warning,
		Desc: "@codecs should be present on the AdaptationSet or the Representation", SpecRef: "DASH-IF IOP",
	})
	RuleContentType = report.Register(report.Rule{
		ID: "MPD-CONTENTTYPE", Severity: report.Warning,
		Desc: "AdaptationSet@contentType should agree with @mimeType", SpecRef: refAS,
	})
	RuleTemplateRepID = report.Register(report.Rule{
		ID: "MPD-TEMPLATE-REPID", Severity: report.Error,
		Desc:    "A SegmentTemplate shared by several Representations needs $RepresentationID$ or $Bandwidth$",
		SpecRef: refTemplate,
	})
	RuleTimelineOrder = report.Register(report.Rule{
		ID: "MPD-TIMELINE-ORDER", Severity: report.Error,
		Desc: "SegmentTimeline entries must not overlap", SpecRef: refTimeline,
	})
	RuleTimelineGap = report.Register(report.Rule{
		ID: "MPD-TIMELINE-GAP", Severity: report.Warning,
		Desc: "SegmentTimeline should have no gaps", SpecRef: "DASH-IF Timing Model 9.2",
	})
	RuleTimelineNegR = report.Register(report.Rule{
		ID: "MPD-TIMELINE-NEGR", Severity: report.Error,
		Desc: "A negative S@r must be on the last S or be followed by an S with @t", SpecRef: refTimeline,
	})
	RuleMaxSegDur = report.Register(report.Rule{
		ID: "MPD-MAXSEGDUR", Severity: report.Warning,
		Desc: "No segment may be longer than MPD@maxSegmentDuration", SpecRef: refMPD,
	})
	RuleDurationYM = report.Register(report.Rule{
		ID: "MPD-DURATION-YM", Severity: report.Warning,
		Desc: "xs:duration values should not use years or months, whose length varies", SpecRef: "DASH-IF IOP",
	})
)

// Check validates the MPD and returns the parsed MPD (nil if it could not be parsed).
func Check(data []byte, mpdURL string) (*m.MPD, []report.Finding) {
	findings, err := schema.Validate(data, mpdURL)
	if err != nil {
		findings = append(findings, RuleParse.New(report.Location{URL: mpdURL}, "schema setup failed: %v", err))
	}
	findings = append(findings, checkDurations(data)...)
	md, err := m.ReadFromString(string(data))
	if err != nil {
		findings = append(findings, RuleParse.New(report.Location{URL: mpdURL}, "%v", err))
		return nil, findings
	}
	findings = append(findings, Rules(md)...)
	findings = append(findings, structureRules(md)...)
	findings = append(findings, profileRules(md)...)
	findings = append(findings, descriptorRules(md)...)
	findings = append(findings, dolbyRules(md)...)
	findings = append(findings, lowLatencyRules(md)...)
	raw := parseRaw(data)
	findings = append(findings, rawStructureRules(raw)...)
	findings = append(findings, dvbRules(md, raw, len(data), mpdURL)...)
	findings = append(findings, hbbtvCodecRules(md)...)
	findings = append(findings, dvbSchemaRules(md, data, mpdURL)...)
	return md, findings
}

// Rules runs the semantic rules on a parsed MPD.
func Rules(md *m.MPD) []report.Finding {
	var f []report.Finding
	mpdLoc := report.Location{Path: "MPD"}
	dynamic := md.GetType() == "dynamic"
	if dynamic {
		if md.AvailabilityStartTime == "" {
			f = append(f, RuleDynAST.New(mpdLoc, "dynamic MPD without @availabilityStartTime"))
		}
		if md.PublishTime == "" {
			f = append(f, RuleDynPublishTime.New(mpdLoc, "dynamic MPD without @publishTime"))
		}
		if md.TimeShiftBufferDepth == nil {
			f = append(f, RuleDynTSBD.New(mpdLoc, "no @timeShiftBufferDepth"))
		}
		if len(md.UTCTimings) == 0 {
			f = append(f, RuleDynUTCTiming.New(mpdLoc, "dynamic MPD without UTCTiming"))
		}
	}
	for i, u := range md.UTCTimings {
		loc := report.Location{Path: fmt.Sprintf("MPD/UTCTiming[%d]", i)}
		switch string(u.SchemeIdUri) {
		case "urn:mpeg:dash:utc:http-xsdate:2014", "urn:mpeg:dash:utc:http-iso:2014",
			"urn:mpeg:dash:utc:http-head:2014", "urn:mpeg:dash:utc:direct:2014":
		case "urn:mpeg:dash:utc:ntp:2014", "urn:mpeg:dash:utc:sntp:2014", "urn:mpeg:dash:utc:http-ntp:2014":
			fd := RuleUTCTimingScheme.New(loc, "scheme %s cannot be used by browser-based players", u.SchemeIdUri)
			f = append(f, fd)
		default:
			f = append(f, RuleUTCTimingScheme.New(loc, "unknown or legacy scheme %q", u.SchemeIdUri))
		}
	}

	periodIDs := map[string]int{}
	var prevStart time.Duration = -1
	var maxSegDur time.Duration
	if md.MaxSegmentDuration != nil {
		maxSegDur = time.Duration(*md.MaxSegmentDuration)
	}
	for pi, p := range md.Periods {
		pPath := fmt.Sprintf("Period[%d]", pi)
		pLoc := report.Location{Path: pPath}
		switch {
		case p.Id == "" && dynamic:
			f = append(f, RulePeriodID.New(pLoc, "Period without @id in dynamic MPD"))
		case p.Id != "":
			if prev, ok := periodIDs[p.Id]; ok {
				f = append(f, RulePeriodID.New(pLoc, "Period@id %q also used by Period[%d]", p.Id, prev))
			}
			periodIDs[p.Id] = pi
		}
		if p.Start != nil {
			start := time.Duration(*p.Start)
			if start < prevStart {
				f = append(f, RulePeriodOrder.New(pLoc, "Period@start %v before previous Period@start %v", start, prevStart))
			}
			prevStart = start
		}
		if p.SegmentTemplate != nil && p.SegmentTemplate.SegmentTimeline != nil {
			f = append(f, checkTimeline(p.SegmentTemplate.SegmentTimeline, timescale(p.SegmentTemplate), maxSegDur, pLoc)...)
		}
		asIDs := map[uint32]int{}
		repIDs := map[string]string{}
		for ai, as := range p.AdaptationSets {
			asPath := fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)
			asLoc := report.Location{Path: asPath}
			if as.Id != nil {
				if prev, ok := asIDs[*as.Id]; ok {
					f = append(f, RuleASID.New(asLoc, "AdaptationSet@id %d also used by AdaptationSet[%d]", *as.Id, prev))
				}
				asIDs[*as.Id] = ai
			}
			f = append(f, checkContentType(as, asLoc)...)
			f = append(f, checkSharedTemplate(p.SegmentTemplate, as, asLoc)...)
			if as.SegmentTemplate != nil && as.SegmentTemplate.SegmentTimeline != nil {
				ts := timescale(p.SegmentTemplate, as.SegmentTemplate)
				f = append(f, checkTimeline(as.SegmentTemplate.SegmentTimeline, ts, maxSegDur, asLoc)...)
			}
			for _, rep := range as.Representations {
				rPath := fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id)
				rLoc := report.Location{Path: rPath, RepID: rep.Id}
				if prev, ok := repIDs[rep.Id]; ok && rep.Id != "" {
					f = append(f, RuleRepID.New(rLoc, "Representation@id %q also used in %s", rep.Id, prev))
				}
				repIDs[rep.Id] = asPath
				if rep.MimeType == "" && as.MimeType == "" {
					f = append(f, RuleMimeType.New(rLoc, "no @mimeType on Representation or AdaptationSet"))
				}
				if rep.Codecs == "" && as.Codecs == "" {
					f = append(f, RuleCodecs.New(rLoc, "no @codecs on Representation or AdaptationSet"))
				}
				if rep.SegmentTemplate != nil && rep.SegmentTemplate.SegmentTimeline != nil {
					ts := timescale(p.SegmentTemplate, as.SegmentTemplate, rep.SegmentTemplate)
					f = append(f, checkTimeline(rep.SegmentTemplate.SegmentTimeline, ts, maxSegDur, rLoc)...)
				}
				if dur := templateDuration(p.SegmentTemplate, as.SegmentTemplate, rep.SegmentTemplate); dur != nil && maxSegDur > 0 {
					ts := timescale(p.SegmentTemplate, as.SegmentTemplate, rep.SegmentTemplate)
					d := ticksToDur(uint64(*dur), ts)
					if d > maxSegDur {
						f = append(f, RuleMaxSegDur.New(rLoc, "SegmentTemplate@duration %v exceeds @maxSegmentDuration %v", d, maxSegDur))
					}
				}
			}
		}
	}
	return f
}

// templateDuration returns the inherited SegmentTemplate@duration, or nil.
func templateDuration(levels ...*m.SegmentTemplateType) *uint32 {
	var d *uint32
	for _, t := range levels {
		if t != nil && t.Duration != nil {
			d = t.Duration
		}
	}
	return d
}

// timescale returns the inherited @timescale of a chain of SegmentTemplates, highest level first.
func timescale(levels ...*m.SegmentTemplateType) uint32 {
	ts := uint32(1)
	for _, t := range levels {
		if t != nil && t.Timescale != nil {
			ts = *t.Timescale
		}
	}
	return ts
}

func ticksToDur(ticks uint64, timescale uint32) time.Duration {
	return time.Duration(float64(ticks) / float64(timescale) * float64(time.Second))
}

func checkContentType(as *m.AdaptationSetType, loc report.Location) []report.Finding {
	ct := string(as.ContentType)
	mime := as.MimeType
	if ct == "" || mime == "" {
		return nil
	}
	major, _, _ := strings.Cut(mime, "/")
	switch {
	case ct == "video" && major == "video", ct == "audio" && major == "audio":
		return nil
	case ct == "text" && (major == "application" || major == "text"):
		return nil
	case ct == "image" && major == "image":
		return nil
	case ct != "video" && ct != "audio" && ct != "text" && ct != "image":
		return nil
	}
	return []report.Finding{RuleContentType.New(loc, "@contentType %q but @mimeType %q", ct, mime)}
}

func checkSharedTemplate(pt *m.SegmentTemplateType, as *m.AdaptationSetType, loc report.Location) []report.Finding {
	if len(as.Representations) < 2 {
		return nil
	}
	media := ""
	for _, t := range []*m.SegmentTemplateType{pt, as.SegmentTemplate} {
		if t != nil && t.Media != "" {
			media = t.Media
		}
	}
	if media == "" {
		return nil
	}
	for _, r := range as.Representations {
		if r.SegmentTemplate != nil && r.SegmentTemplate.Media != "" {
			return nil // Representations override @media
		}
	}
	if strings.Contains(media, "$RepresentationID$") || strings.Contains(media, "$Bandwidth") {
		return nil
	}
	return []report.Finding{RuleTemplateRepID.New(loc,
		"@media %q is shared by %d Representations but has neither $RepresentationID$ nor $Bandwidth$",
		media, len(as.Representations))}
}

func checkTimeline(stl *m.SegmentTimelineType, timescale uint32, maxSegDur time.Duration, loc report.Location) []report.Finding {
	var f []report.Finding
	var end uint64
	maxD := uint64(0)
	for i, s := range stl.S {
		if s.T != nil && i > 0 {
			switch {
			case *s.T < end:
				f = append(f, RuleTimelineOrder.New(loc, "S[%d]@t=%d overlaps previous end %d", i, *s.T, end))
			case *s.T > end:
				f = append(f, RuleTimelineGap.New(loc, "gap of %d ticks before S[%d]@t=%d", *s.T-end, i, *s.T))
			}
		}
		if s.T != nil {
			end = *s.T
		}
		if s.R < 0 {
			if i+1 < len(stl.S) && stl.S[i+1].T == nil {
				f = append(f, RuleTimelineNegR.New(loc, "S[%d]@r=%d is followed by an S without @t", i, s.R))
			}
			// The repeat count is unknown; continue from the next S@t.
			if i+1 < len(stl.S) && stl.S[i+1].T != nil {
				end = *stl.S[i+1].T
			}
		} else {
			end += s.D * uint64(s.R+1)
		}
		maxD = max(maxD, s.D)
	}
	if maxSegDur > 0 && timescale > 0 {
		if d := ticksToDur(maxD, timescale); d > maxSegDur {
			f = append(f, RuleMaxSegDur.New(loc, "SegmentTimeline has a %v segment, more than @maxSegmentDuration %v", d, maxSegDur))
		}
	}
	return f
}

var (
	attrRe     = regexp.MustCompile(`([A-Za-z:]+)="(-?P[^"]*)"`)
	durationRe = regexp.MustCompile(`^-?P(\d+(\.\d+)?[YMWD])*(T(\d+(\.\d+)?[HMS])*)?$`)
)

// checkDurations finds xs:duration values with year or month components.
// It works on the raw text since the parsed model has converted durations.
func checkDurations(data []byte) []report.Finding {
	var f []report.Finding
	for _, idx := range attrRe.FindAllSubmatchIndex(data, -1) {
		name := string(data[idx[2]:idx[3]])
		val := string(data[idx[4]:idx[5]])
		if !durationRe.MatchString(val) {
			continue
		}
		datePart, _, _ := strings.Cut(val, "T")
		if strings.ContainsAny(datePart, "YM") {
			line := 1 + strings.Count(string(data[:idx[0]]), "\n")
			f = append(f, RuleDurationYM.New(report.Location{Line: line}, "@%s=%q uses years or months", name, val))
		}
	}
	return f
}
