package mpdcheck

import (
	"fmt"
	"slices"
	"strings"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// MPD rules ported from the Low Latency module of the DASH-IF Conformance
// tool, which implements the DASH-IF low-latency modes (IOP part 9, "9.X").

const (
	profileLowLatency = "http://www.dashif.org/guidelines/low-latency-live-v5"
	refLL             = "DASH-IF Low-Latency Modes 9.X.4"
)

var (
	RuleLLProfile = report.Register(report.Rule{
		ID: "MPD-LL-PROFILE", Severity: report.Warning,
		Desc: "Low-latency presentations should signal the DASH-IF low-latency profile", SpecRef: refLL + ".1",
	})
	RuleLLInband = report.Register(report.Rule{
		ID: "MPD-LL-INBAND", Severity: report.Warning,
		Desc:    "Low-latency Representations should carry MPD validity expiration events (urn:mpeg:dash:event:2012, @value 1)",
		SpecRef: refLL + ".2",
	})
	RuleLLInbandValue = report.Register(report.Rule{
		ID: "MPD-LL-INBAND-VALUE", Severity: report.Error,
		Desc: "The urn:mpeg:dash:event:2012 InbandEventStream must have @value 1", SpecRef: refLL + ".2",
	})
	RuleLLLeapSecond = report.Register(report.Rule{
		ID: "MPD-LL-LEAPSECOND", Severity: report.Warning,
		Desc: "Low-latency presentations should provide LeapSecondInformation", SpecRef: refLL + ".2",
	})
	RuleLLProducerReferenceTime = report.Register(report.Rule{
		ID: "MPD-LL-PRT", Severity: report.Error,
		Desc:    "Each low-latency AdaptationSet needs a valid ProducerReferenceTime with a unique @id",
		SpecRef: refLL + ".3",
	})
	RuleLLAddressing = report.Register(report.Rule{
		ID: "MPD-LL-ADDRESSING", Severity: report.Error,
		Desc:    "Low-latency AdaptationSets must use a SegmentTemplate with @duration and $Number$, or a SegmentTimeline",
		SpecRef: refLL + ".3",
	})
	RuleLLServiceDescription = report.Register(report.Rule{
		ID: "MPD-LL-SERVICEDESC", Severity: report.Error,
		Desc: "Low-latency presentations need a ServiceDescription with a target Latency", SpecRef: refLL + ".2",
	})
	RuleLLServiceDescriptionOptional = report.Register(report.Rule{
		ID: "MPD-LL-SERVICEDESC-OPTIONAL", Severity: report.Info,
		Desc:    "A low-latency ServiceDescription may give minimum and maximum latency and playback rates",
		SpecRef: refLL + ".2",
	})
	RuleLLUTCTiming = report.Register(report.Rule{
		ID: "MPD-LL-UTCTIMING", Severity: report.Error,
		Desc:    "Low-latency presentations need a UTCTiming with the http-xsdate, http-iso or http-ntp scheme",
		SpecRef: refLL + ".2",
	})
	RuleLLChunked = report.Register(report.Rule{
		ID: "MPD-LL-CHUNKED", Severity: report.Error,
		Desc:    "Chunked low-latency Representations need @availabilityTimeOffset and @availabilityTimeComplete false",
		SpecRef: refLL + ".5",
	})
	RuleLLMode = report.Register(report.Rule{
		ID: "MPD-LL-MODE", Severity: report.Error,
		Desc: "A low-latency AdaptationSet must be either chunked or segment based, not both", SpecRef: refLL + ".3",
	})
	RuleLLAlignment = report.Register(report.Rule{
		ID: "MPD-LL-ALIGNMENT", Severity: report.Error,
		Desc: "Low-latency AdaptationSets must set @segmentAlignment or @subsegmentAlignment", SpecRef: refLL + ".5",
	})
	RuleLLMimeType = report.Register(report.Rule{
		ID: "MPD-LL-MIMETYPE", Severity: report.Error,
		Desc: "Low-latency @mimeType must be the @contentType with /mp4", SpecRef: refLL + ".5",
	})
	RuleLLResync = report.Register(report.Rule{
		ID: "MPD-LL-RESYNC", Severity: report.Warning,
		Desc:    "Chunked low-latency Representations should have a Resync element and @qualityRanking",
		SpecRef: refLL + ".5",
	})
	RuleLLResyncValue = report.Register(report.Rule{
		ID: "MPD-LL-RESYNC-VALUE", Severity: report.Error,
		Desc:    "Resync elements need a valid @type, @marker true and @dT not above the target latency",
		SpecRef: refLL + ".5",
	})
	RuleLLTimelineRepeat = report.Register(report.Rule{
		ID: "MPD-LL-TIMELINE-REPEAT", Severity: report.Warning,
		Desc:    "Consecutive S elements with the same duration should be combined with @r",
		SpecRef: refLL + ".5",
	})
)

// isLowLatency reports whether a dynamic MPD uses low-latency features.
func isLowLatency(md *m.MPD, reps []*addressing.Rep) bool {
	if md.GetType() != "dynamic" {
		return false
	}
	if profiles(splitProfiles(string(md.Profiles))).has(profileLowLatency) {
		return true
	}
	return slices.ContainsFunc(reps, isChunked)
}

// isChunked reports whether a Representation is offered in chunks.
func isChunked(r *addressing.Rep) bool {
	return r.ATO > 0 || (r.ATC != nil && !*r.ATC)
}

func targetLatency(md *m.MPD) (ms uint32, ok bool) {
	for _, sd := range md.ServiceDescription {
		for _, l := range sd.Latencies {
			if l.Target != nil {
				return *l.Target, true
			}
		}
	}
	return 0, false
}

func lowLatencyRules(md *m.MPD) []report.Finding {
	reps, _ := addressing.Reps(md, "")
	if !isLowLatency(md, reps) {
		return nil
	}
	var f []report.Finding
	mpdLoc := report.Location{Path: "MPD"}
	// LL-1
	if !profiles(splitProfiles(string(md.Profiles))).has(profileLowLatency) {
		f = append(f, RuleLLProfile.New(mpdLoc, "low-latency features without the profile %s", profileLowLatency))
	}
	// LL-4
	if md.LeapSecondInformation == nil {
		f = append(f, RuleLLLeapSecond.New(mpdLoc, "no LeapSecondInformation"))
	}
	// LL-10
	f = append(f, checkServiceDescription(md, mpdLoc)...)
	// LL-11
	if !slices.ContainsFunc(md.UTCTimings, func(u *m.DescriptorType) bool {
		switch u.SchemeIdUri {
		case "urn:mpeg:dash:utc:http-xsdate:2014", "urn:mpeg:dash:utc:http-iso:2014", "urn:mpeg:dash:utc:http-ntp:2014":
			return true
		}
		return false
	}) {
		f = append(f, RuleLLUTCTiming.New(mpdLoc, "no UTCTiming with http-xsdate, http-iso or http-ntp"))
	}
	target, hasTarget := targetLatency(md)
	byAS := map[string][]*addressing.Rep{}
	for _, r := range reps {
		byAS[r.ASPath] = append(byAS[r.ASPath], r)
	}
	for pi, p := range md.Periods {
		for ai, as := range p.AdaptationSets {
			asPath := fmt.Sprintf("Period[%d]/AdaptationSet[%d]", pi, ai)
			loc := report.Location{Path: asPath}
			asReps := byAS[asPath]
			// LL-5, LL-6
			f = append(f, checkProducerReferenceTimes(as, asPath)...)
			// LL-7 to LL-9
			f = append(f, checkLLAddressing(asReps, loc)...)
			// LL-26
			if !as.SegmentAlignment && !as.SubsegmentAlignment {
				f = append(f, RuleLLAlignment.New(loc, "neither @segmentAlignment nor @subsegmentAlignment is true"))
			}
			// LL-37
			nChunked := 0
			for _, r := range asReps {
				if isChunked(r) {
					nChunked++
				}
			}
			if nChunked > 0 && nChunked < len(asReps) {
				f = append(f, RuleLLMode.New(loc, "%d of %d Representations are chunked", nChunked, len(asReps)))
			}
			// LL-29 to LL-33
			if nChunked > 0 {
				f = append(f, checkResync(as, asReps, asPath, target, hasTarget)...)
			}
			for _, r := range asReps {
				rLoc := r.Loc()
				// LL-20
				if ct := string(as.ContentType); ct != "" && r.MimeType != ct+"/mp4" && r.MimeType != ct+"/mp4, profiles='cmfc'" {
					f = append(f, RuleLLMimeType.New(rLoc, "@mimeType %q with @contentType %q", r.MimeType, ct))
				}
				// LL-16, LL-17
				if isChunked(r) {
					if r.ATO <= 0 {
						f = append(f, RuleLLChunked.New(rLoc, "chunked Representation without @availabilityTimeOffset"))
					}
					if r.ATC == nil || *r.ATC {
						f = append(f, RuleLLChunked.New(rLoc, "@availabilityTimeOffset without @availabilityTimeComplete false"))
					}
				}
			}
			for _, rep := range as.Representations {
				f = append(f, checkInbandEvents(as, rep, fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id))...)
			}
			// LL-39
			for _, t := range []*m.SegmentTemplateType{p.SegmentTemplate, as.SegmentTemplate} {
				if t != nil && t.SegmentTimeline != nil {
					f = append(f, checkTimelineRepeats(t.SegmentTimeline, loc)...)
				}
			}
		}
	}
	return f
}

func checkServiceDescription(md *m.MPD, loc report.Location) []report.Finding {
	if len(md.ServiceDescription) == 0 {
		return []report.Finding{RuleLLServiceDescription.New(loc, "no ServiceDescription")}
	}
	for _, sd := range md.ServiceDescription {
		for _, l := range sd.Latencies {
			if l.Target == nil {
				continue
			}
			var f []report.Finding
			if l.Min == nil && l.Max == nil {
				f = append(f, RuleLLServiceDescriptionOptional.New(loc, "Latency without @min and @max"))
			}
			if !slices.ContainsFunc(sd.PlaybackRates, func(pr *m.PlaybackRateType) bool { return pr.Min != 0 && pr.Max != 0 }) {
				f = append(f, RuleLLServiceDescriptionOptional.New(loc, "no PlaybackRate with @min and @max"))
			}
			return f
		}
	}
	return []report.Finding{RuleLLServiceDescription.New(loc, "no ServiceDescription has a Latency with @target")}
}

// checkProducerReferenceTimes requires a valid ProducerReferenceTime with an
// @id that is unique within the AdaptationSet. Upstream requires unique ids
// across the whole MPD.
func checkProducerReferenceTimes(as *m.AdaptationSetType, asPath string) []report.Finding {
	var f []report.Finding
	valid := false
	ids := map[uint32]int{}
	for i, prt := range as.ProducerReferenceTimes {
		loc := report.Location{Path: fmt.Sprintf("%s/ProducerReferenceTime[%d]", asPath, i)}
		var problems []string
		if prt.Type != "" && prt.Type != "encoder" && prt.Type != "captured" {
			problems = append(problems, fmt.Sprintf("@type %q", prt.Type))
		}
		if prt.UTCTiming == nil {
			problems = append(problems, "no UTCTiming")
		}
		if prt.WallClockTime == "" {
			problems = append(problems, "no @wallClockTime")
		}
		if len(problems) > 0 {
			f = append(f, RuleLLProducerReferenceTime.New(loc, "invalid ProducerReferenceTime: %s", strings.Join(problems, ", ")))
			continue
		}
		valid = true
		if prev, ok := ids[prt.Id]; ok {
			f = append(f, RuleLLProducerReferenceTime.New(loc, "ProducerReferenceTime@id %d also used by ProducerReferenceTime[%d]", prt.Id, prev))
		}
		ids[prt.Id] = i
	}
	if !valid {
		f = append(f, RuleLLProducerReferenceTime.New(report.Location{Path: asPath}, "no valid ProducerReferenceTime"))
	}
	return f
}

func checkLLAddressing(reps []*addressing.Rep, loc report.Location) []report.Finding {
	for _, r := range reps {
		switch r.Kind {
		case addressing.KindNumber, addressing.KindTimelineTime, addressing.KindTimelineNumber:
		default:
			return []report.Finding{RuleLLAddressing.New(loc, "%s addressing", r.Kind)}
		}
	}
	return nil
}

func checkInbandEvents(as *m.AdaptationSetType, rep *m.RepresentationType, path string) []report.Finding {
	var f []report.Finding
	streams := append(slices.Clone(as.InbandEventStreams), rep.InbandEventStreams...)
	found := false
	for _, s := range streams {
		if s.SchemeIdUri != "urn:mpeg:dash:event:2012" {
			continue
		}
		found = true
		if s.Value != "1" {
			f = append(f, RuleLLInbandValue.New(report.Location{Path: path, RepID: rep.Id}, "urn:mpeg:dash:event:2012 @value %q, not 1", s.Value))
		}
	}
	if !found {
		f = append(f, RuleLLInband.New(report.Location{Path: path, RepID: rep.Id}, "no urn:mpeg:dash:event:2012 InbandEventStream"))
	}
	return f
}

// checkResync ports the Resync rules for one chunked AdaptationSet. The
// Representation with the lowest bandwidth needs Resync of type 1 or 2.
func checkResync(as *m.AdaptationSetType, reps []*addressing.Rep, asPath string, target uint32, hasTarget bool) []report.Finding {
	var f []report.Finding
	timescale := map[string]uint32{}
	for _, r := range reps {
		timescale[r.ID] = r.Timescale
	}
	var lowest uint32
	for _, r := range as.Representations {
		if lowest == 0 || r.Bandwidth < lowest {
			lowest = r.Bandwidth
		}
	}
	for _, rep := range as.Representations {
		loc := report.Location{Path: fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id), RepID: rep.Id}
		resyncs := rep.Resyncs
		if len(resyncs) == 0 {
			resyncs = as.Resyncs
		}
		if rep.QualityRanking == nil {
			f = append(f, RuleLLResync.New(loc, "no @qualityRanking"))
		}
		if len(resyncs) == 0 {
			f = append(f, RuleLLResync.New(loc, "no Resync element"))
			continue
		}
		rs := resyncs[0]
		allowed := []uint32{1, 2, 3}
		if rep.Bandwidth == lowest {
			allowed = []uint32{1, 2}
		}
		if !slices.Contains(allowed, rs.Type) {
			f = append(f, RuleLLResyncValue.New(loc, "Resync@type %d, allowed %v", rs.Type, allowed))
		}
		if !rs.Marker {
			f = append(f, RuleLLResyncValue.New(loc, "Resync@marker is not true"))
		}
		if ts := timescale[rep.Id]; hasTarget && rs.DT != nil && ts > 0 {
			// @dT is in the timescale of the Representation; the target latency is in milliseconds.
			dtMS := float64(*rs.DT) * 1000 / float64(ts)
			if dtMS > float64(target) {
				f = append(f, RuleLLResyncValue.New(loc, "Resync@dT is %.0f ms, above the target latency %d ms", dtMS, target))
			} else if dtMS > float64(target)/2 {
				f = append(f, RuleLLResync.New(loc, "Resync@dT is %.0f ms, above half the target latency %d ms", dtMS, target))
			}
		}
	}
	return f
}

func checkTimelineRepeats(stl *m.SegmentTimelineType, loc report.Location) []report.Finding {
	for i := 1; i < len(stl.S); i++ {
		if stl.S[i].D == stl.S[i-1].D && stl.S[i].T == nil {
			return []report.Finding{RuleLLTimelineRepeat.New(loc, "S[%d] and S[%d] have the same @d and could be one S with @r", i-1, i)}
		}
	}
	return nil
}
