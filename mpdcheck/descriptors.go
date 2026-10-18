package mpdcheck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/report"
)

// Descriptor and HbbTV rules ported from the DASH-IF Schematron rules (schematron.sch).

const (
	schemeSRD          = "urn:mpeg:dash:srd:2014"
	schemeFramePacking = "urn:mpeg:dash:14496:10:frame_packing_arrangement_type:2011"
	schemeStereoFormat = "urn:mpeg:dash:13818:1:stereo_video_format_type:2011"
	schemeChannelPos   = "urn:mpeg:dash:outputChannelPositionList:2012"
	profileHbbTV       = "urn:hbbtv:dash:profile:isoff-live:2012"

	refSRD    = "ISO/IEC 23009-1 H.2"
	refHbbTV  = "HbbTV 2.0 E.2"
	refFrame  = "ISO/IEC 23009-1 5.8.4.6"
	refAudioC = "ISO/IEC 23009-1 5.8.4.7"
)

var (
	RuleFramePacking = report.Register(report.Rule{
		ID: "MPD-FRAMEPACKING", Severity: report.Error,
		Desc: "FramePacking must use a known scheme with a value 0 to 6 that fits the codec or container", SpecRef: refFrame,
	})
	RuleChannelPositions = report.Register(report.Rule{
		ID: "MPD-CHANNEL-POSITIONS", Severity: report.Error,
		Desc: "outputChannelPositionList values must be space-separated lists", SpecRef: refAudioC,
	})
	RuleSRDValue = report.Register(report.Rule{
		ID: "MPD-SRD-VALUE", Severity: report.Error,
		Desc:    "SRD descriptors need a @value of 5, 7 or 8 comma-separated integers",
		SpecRef: refSRD,
	})
	RuleSRDPlacement = report.Register(report.Rule{
		ID: "MPD-SRD-PLACEMENT", Severity: report.Error,
		Desc: "SRD descriptors may only be on AdaptationSets and SubRepresentations", SpecRef: refSRD,
	})
	RuleSRDEssential = report.Register(report.Rule{
		ID: "MPD-SRD-ESSENTIAL", Severity: report.Error,
		Desc:    "When every AdaptationSet in a Period has an SRD, at least one must be a SupplementalProperty",
		SpecRef: refSRD,
	})
	RuleSRDTotal = report.Register(report.Rule{
		ID: "MPD-SRD-TOTAL", Severity: report.Error,
		Desc:    "Each SRD source needs a total size, given by every descriptor if they differ",
		SpecRef: refSRD,
	})
	RuleSRDBounds = report.Register(report.Rule{
		ID: "MPD-SRD-BOUNDS", Severity: report.Error,
		Desc: "SRD objects must fit within the total size: x+w <= W and y+h <= H", SpecRef: refSRD,
	})
	RuleHbbTV = report.Register(report.Rule{
		ID: "MPD-HBBTV", Severity: report.Error,
		Desc:    "MPDs with the HbbTV profile must follow its restrictions on segments and alignment",
		SpecRef: refHbbTV,
	})
)

// srd is a parsed SRD descriptor.
type srd struct {
	loc        report.Location
	vals       []int // source_id, x, y, w, h, [W, H, [spatial_set_id]]
	valid      bool
	totalGiven bool
}

var srdValueRe = regexp.MustCompile(`^(\d+,){4,7}\d+$`)

func parseSRD(d *m.DescriptorType, loc report.Location) (srd, []report.Finding) {
	s := srd{loc: loc}
	// SCH-79 to SCH-82
	switch {
	case d.Value == "":
		return s, []report.Finding{RuleSRDValue.New(loc, "SRD descriptor without @value")}
	case !srdValueRe.MatchString(d.Value):
		return s, []report.Finding{RuleSRDValue.New(loc, "SRD @value %q is not 5 to 8 comma-separated integers", d.Value)}
	}
	for v := range strings.SplitSeq(d.Value, ",") {
		n, _ := strconv.Atoi(v)
		s.vals = append(s.vals, n)
	}
	if len(s.vals) == 6 {
		return s, []report.Finding{RuleSRDValue.New(loc, "SRD @value %q has total_width but no total_height", d.Value)}
	}
	s.valid = true
	s.totalGiven = len(s.vals) >= 7
	return s, nil
}

func descriptorRules(md *m.MPD) []report.Finding {
	var f []report.Finding
	mpdP := profiles(splitProfiles(string(md.Profiles)))
	for pi, p := range md.Periods {
		pPath := fmt.Sprintf("Period[%d]", pi)
		var srds []srd
		// SCH-78: SRD on the Period level is misplaced.
		for i, d := range p.SupplementalProperties {
			if d.SchemeIdUri == schemeSRD {
				f = append(f, RuleSRDPlacement.New(report.Location{Path: fmt.Sprintf("%s/SupplementalProperty[%d]", pPath, i)},
					"SRD descriptor on the Period"))
			}
		}
		allEssential := len(p.AdaptationSets) > 0
		for ai, as := range p.AdaptationSets {
			asPath := fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)
			collect := func(rb *m.RepresentationBaseType, path string, allowed bool) {
				for i, d := range rb.EssentialProperties {
					if d.SchemeIdUri == schemeSRD {
						loc := report.Location{Path: fmt.Sprintf("%s/EssentialProperty[%d]", path, i)}
						if !allowed {
							f = append(f, RuleSRDPlacement.New(loc, "SRD descriptor on a Representation"))
						}
						s, sf := parseSRD(d, loc)
						f = append(f, sf...)
						srds = append(srds, s)
					}
				}
				for i, d := range rb.SupplementalProperties {
					if d.SchemeIdUri == schemeSRD {
						loc := report.Location{Path: fmt.Sprintf("%s/SupplementalProperty[%d]", path, i)}
						if !allowed {
							f = append(f, RuleSRDPlacement.New(loc, "SRD descriptor on a Representation"))
						}
						s, sf := parseSRD(d, loc)
						f = append(f, sf...)
						srds = append(srds, s)
					}
				}
			}
			collect(&as.RepresentationBaseType, asPath, true)
			hasEssentialSRD := false
			for _, d := range as.EssentialProperties {
				if d.SchemeIdUri == schemeSRD {
					hasEssentialSRD = true
				}
			}
			if !hasEssentialSRD {
				allEssential = false
			}
			f = append(f, checkFramePacking(as.FramePackings, as.Codecs, as.MimeType, asPath)...)
			f = append(f, checkChannelPositions(as.AudioChannelConfigurations, asPath)...)
			for _, rep := range as.Representations {
				rPath := fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id)
				collect(&rep.RepresentationBaseType, rPath, false)
				f = append(f, checkFramePacking(rep.FramePackings, first(rep.Codecs, as.Codecs), first(rep.MimeType, as.MimeType), rPath)...)
				f = append(f, checkChannelPositions(rep.AudioChannelConfigurations, rPath)...)
				for si, sr := range rep.SubRepresentations {
					collect(&sr.RepresentationBaseType, fmt.Sprintf("%s/SubRepresentation[%d]", rPath, si), true)
				}
			}
		}
		// SCH-71
		if allEssential {
			f = append(f, RuleSRDEssential.New(report.Location{Path: pPath}, "every AdaptationSet has an SRD EssentialProperty"))
		}
		f = append(f, checkSRDGroups(srds, pPath)...)
		f = append(f, hbbtvRules(p, mpdP, pPath)...)
	}
	return f
}

// checkSRDGroups checks the descriptors of each SRD source (SCH-72 to SCH-77).
func checkSRDGroups(srds []srd, pPath string) []report.Finding {
	var f []report.Finding
	groups := map[int][]srd{}
	var order []int
	for _, s := range srds {
		if !s.valid {
			continue
		}
		id := s.vals[0]
		if _, ok := groups[id]; !ok {
			order = append(order, id)
		}
		groups[id] = append(groups[id], s)
	}
	for _, id := range order {
		g := groups[id]
		totals := map[[2]int]bool{}
		var total [2]int
		for _, s := range g {
			if s.totalGiven {
				total = [2]int{s.vals[5], s.vals[6]}
				totals[total] = true
			}
		}
		loc := report.Location{Path: pPath}
		switch {
		case len(totals) == 0:
			f = append(f, RuleSRDTotal.New(loc, "no SRD descriptor of source %d gives total_width and total_height", id))
			continue
		case len(totals) > 1:
			for _, s := range g {
				if !s.totalGiven {
					f = append(f, RuleSRDTotal.New(s.loc, "source %d has several total sizes, so each descriptor must give one", id))
				}
			}
		}
		for _, s := range g {
			W, H := total[0], total[1]
			if s.totalGiven {
				W, H = s.vals[5], s.vals[6]
			}
			x, y, w, h := s.vals[1], s.vals[2], s.vals[3], s.vals[4]
			if x+w > W {
				f = append(f, RuleSRDBounds.New(s.loc, "x+w = %d exceeds total_width %d", x+w, W))
			}
			if y+h > H {
				f = append(f, RuleSRDBounds.New(s.loc, "y+h = %d exceeds total_height %d", y+h, H))
			}
		}
	}
	return f
}

// SCH-59 to SCH-62
func checkFramePacking(fps []*m.DescriptorType, codecs, mimeType, path string) []report.Finding {
	var f []report.Finding
	for i, d := range fps {
		loc := report.Location{Path: fmt.Sprintf("%s/FramePacking[%d]", path, i)}
		switch d.SchemeIdUri {
		case schemeFramePacking:
			if !strings.Contains(codecs, "avc") && !strings.Contains(codecs, "svc") && !strings.Contains(codecs, "mvc") {
				f = append(f, RuleFramePacking.New(loc, "frame_packing_arrangement_type scheme for @codecs %q, which is not AVC, SVC or MVC", codecs))
			}
		case schemeStereoFormat:
			if mimeType != "video/mp2t" {
				f = append(f, RuleFramePacking.New(loc, "stereo_video_format_type scheme for @mimeType %q, not video/mp2t", mimeType))
			}
		default:
			f = append(f, RuleFramePacking.New(loc, "unknown FramePacking scheme %q", d.SchemeIdUri))
		}
		if n, err := strconv.Atoi(d.Value); err != nil || n < 0 || n > 6 {
			f = append(f, RuleFramePacking.New(loc, "FramePacking @value %q is not 0 to 6", d.Value))
		}
	}
	return f
}

// SCH-63
func checkChannelPositions(accs []*m.DescriptorType, path string) []report.Finding {
	var f []report.Finding
	for i, d := range accs {
		if d.SchemeIdUri == schemeChannelPos && len(strings.Fields(d.Value)) < 2 {
			f = append(f, RuleChannelPositions.New(report.Location{Path: fmt.Sprintf("%s/AudioChannelConfiguration[%d]", path, i)},
				"outputChannelPositionList @value %q is not a list", d.Value))
		}
	}
	return f
}

// hbbtvRules ports SCH-30 to SCH-32, SCH-37 to SCH-39 and SCH-103.
func hbbtvRules(p *m.Period, mpdP profiles, pPath string) []report.Finding {
	var f []report.Finding
	for ai, as := range p.AdaptationSets {
		asPath := fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)
		asP := effective(as.Profiles, mpdP)
		if asP.has(profileHbbTV) {
			loc := report.Location{Path: asPath}
			if as.SubsegmentAlignment {
				f = append(f, RuleHbbTV.New(loc, "@subsegmentAlignment true is not part of the HbbTV profile"))
			}
			if as.SubsegmentStartsWithSAP == 1 || as.SubsegmentStartsWithSAP == 2 {
				f = append(f, RuleHbbTV.New(loc, "@subsegmentStartsWithSAP %d is not part of the HbbTV profile", as.SubsegmentStartsWithSAP))
			}
			if as.SubsegmentStartsWithSAP == 3 && len(as.Representations) < 2 {
				f = append(f, RuleHbbTV.New(loc, "@subsegmentStartsWithSAP 3 needs more than one Representation"))
			}
			if mpdP.iop() && !as.SegmentAlignment {
				f = append(f, RuleHbbTV.New(loc, "@segmentAlignment must be true with the HbbTV profile"))
			}
		}
		for ri, rep := range as.Representations {
			repP := effective(rep.Profiles, asP)
			if !repP.has(profileHbbTV) {
				continue
			}
			loc := report.Location{Path: fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id), RepID: rep.Id}
			if len(rep.BaseURLs) > 0 {
				f = append(f, RuleHbbTV.New(loc, "BaseURL in a Representation is not part of the HbbTV profile"))
			}
			if rep.SegmentTemplate == nil && as.SegmentTemplate == nil && p.SegmentTemplate == nil {
				f = append(f, RuleHbbTV.New(loc, "no SegmentTemplate, which the HbbTV profile requires"))
			}
			if as.SubsegmentStartsWithSAP == 3 && rep.MediaStreamStructureId != nil {
				for _, next := range as.Representations[ri+1:] {
					if next.MediaStreamStructureId != nil && *next.MediaStreamStructureId == *rep.MediaStreamStructureId {
						f = append(f, RuleHbbTV.New(loc, "@mediaStreamStructureId %q repeated with @subsegmentStartsWithSAP 3", *rep.MediaStreamStructureId))
					}
				}
			}
		}
	}
	return f
}
