package segcheck

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Segment rules ported from the IOP module of the DASH-IF Conformance tool
// (DASH-IF IOP 4.3, clause 6.2.5.2 on bitstream switching and 3.10.3 on on-demand).

const (
	iopPrefix          = "http://dashif.org/guidelines/dash"
	profileIOPOnDemand = "http://dashif.org/guidelines/dash-if-ondemand"
	profileOnDemand    = "urn:mpeg:dash:profile:isoff-on-demand:2011"
	refIOPSwitching    = "DASH-IF IOP 6.2.5.2"
)

var (
	RuleIOPBitstream = report.Register(report.Rule{
		ID: "SEG-IOP-BITSTREAM", Severity: report.Error,
		Desc:    "With bitstream switching, AVC must use avc3 and HEVC hev1 sample entries",
		SpecRef: refIOPSwitching,
	})
	RuleIOPParamSets = report.Register(report.Rule{
		ID: "SEG-IOP-PARAMSETS", Severity: report.Error,
		Desc:    "With bitstream switching, avcC must hold SPS and PPS, and hvcC VPS, SPS and PPS",
		SpecRef: refIOPSwitching,
	})
	RuleIOPEditList = report.Register(report.Rule{
		ID: "SEG-IOP-EDITLIST", Severity: report.Error,
		Desc:    "Video edit lists are only allowed with the on-demand profile, and then in all Representations",
		SpecRef: refIOPSwitching,
	})
	RuleIOPFirstCTO = report.Register(report.Rule{
		ID: "SEG-IOP-FIRSTCT", Severity: report.Error,
		Desc:    "With bitstream switching, the first video sample's composition and decode times must be equal",
		SpecRef: refIOPSwitching,
	})
	RuleIOPSidx = report.Register(report.Rule{
		ID: "SEG-IOP-SIDX", Severity: report.Error,
		Desc: "IOP on-demand Representations must have exactly one sidx box", SpecRef: "DASH-IF IOP 3.10.3.2",
	})
	RuleIOPASCodecs = report.Register(report.Rule{
		ID: "SEG-IOP-ASCODECS", Severity: report.Error,
		Desc:    "AdaptationSet @codecs must signal the highest profile and level of its Representations",
		SpecRef: refIOPSwitching,
	})
)

// isIOP reports whether a Representation claims a DASH-IF IOP profile.
func isIOP(rep *addressing.Rep) bool {
	return rep.HasProfilePrefix(iopPrefix)
}

// IsIOPOnDemand reports whether a Representation uses the IOP on-demand profile.
func IsIOPOnDemand(rep *addressing.Rep) bool {
	return rep.HasProfile(profileIOPOnDemand) || (isIOP(rep) && rep.HasProfile(profileOnDemand))
}

// iopSwitching reports whether the IOP bitstream switching rules apply to a video track.
// Upstream applies them whenever @bitstreamSwitching is present; dashcheck requires it to be true.
func iopSwitching(rep *addressing.Rep, init *Init) bool {
	return isIOP(rep) && rep.BitstreamSwitching && init.Video.codec != ""
}

func iopInitRules(rep *addressing.Rep, init *Init, loc report.Location) []report.Finding {
	if !iopSwitching(rep, init) {
		return nil
	}
	var f []report.Finding
	v := init.Video
	switch v.codec {
	case "avc":
		if init.SampleEntry != "avc3" {
			f = append(f, RuleIOPBitstream.New(loc, "AVC sample entry %s, not avc3", init.SampleEntry))
		}
		if !v.hasSPS || !v.hasPPS {
			f = append(f, RuleIOPParamSets.New(loc, "avcC without SPS and PPS"))
		}
	case "hevc":
		if init.SampleEntry != "hev1" {
			f = append(f, RuleIOPBitstream.New(loc, "HEVC sample entry %s, not hev1", init.SampleEntry))
		}
		if !v.hasVPS || !v.hasSPS || !v.hasPPS {
			f = append(f, RuleIOPParamSets.New(loc, "hvcC without VPS, SPS and PPS"))
		}
	}
	if init.HasEditList && !IsIOPOnDemand(rep) {
		f = append(f, RuleIOPEditList.New(loc, "edit list without the on-demand profile"))
	}
	return f
}

func iopMediaRules(rep *addressing.Rep, init *Init, media *Media, loc report.Location) []report.Finding {
	if !iopSwitching(rep, init) || media.FirstCTO == 0 {
		return nil
	}
	return []report.Finding{RuleIOPFirstCTO.New(loc, "first sample has composition time offset %d", media.FirstCTO)}
}

// CheckAdaptationSet runs checks across the Representations of one
// AdaptationSet, given their initialization segments (nil if not available).
func CheckAdaptationSet(reps []*addressing.Rep, inits []*Init) []report.Finding {
	if len(reps) == 0 {
		return nil
	}
	f := iopAdaptationSetRules(reps, inits)
	f = append(f, cmafAdaptationSetRules(reps, inits)...)
	f = append(f, cmafMediaProfileSetRules(reps, inits)...)
	return append(f, dvbAdaptationSetRules(reps, inits)...)
}

// CheckPresentation runs the checks that need results from all
// Representations of the MPD, in MPD order: the initialization segments (nil
// where unavailable) and the presentation time of the first segment of each
// Period in seconds (NaN where not checked).
func CheckPresentation(reps []*addressing.Rep, inits []*Init, firstEPTs []float64) []report.Finding {
	if len(firstEPTs) != len(reps) {
		firstEPTs = make([]float64, len(reps))
		for i := range firstEPTs {
			firstEPTs[i] = math.NaN()
		}
	}
	f := dvbPeriodRules(reps, inits)
	f = append(f, dvbContinuityEPT(reps, firstEPTs)...)
	return append(f, waveRules(reps, inits)...)
}

func iopAdaptationSetRules(reps []*addressing.Rep, inits []*Init) []report.Finding {
	if !isIOP(reps[0]) {
		return nil
	}
	var f []report.Finding
	loc := report.Location{Path: reps[0].ASPath}
	maxProfile, maxLevel := -1, -1
	codec := ""
	nVideo, nEditLists := 0, 0
	for _, init := range inits {
		if init == nil || init.Video.codec == "" {
			continue
		}
		nVideo++
		if init.HasEditList {
			nEditLists++
		}
		codec = init.Video.codec
		maxProfile = max(maxProfile, init.Video.profile)
		maxLevel = max(maxLevel, init.Video.level)
	}
	// IOP-9
	if nEditLists > 0 && nEditLists < nVideo {
		f = append(f, RuleIOPEditList.New(loc, "edit list in %d of %d video Representations", nEditLists, nVideo))
	}
	// IOP-7, IOP-8
	if nVideo == 0 || reps[0].ASCodecs == "" {
		return f
	}
	for c := range strings.SplitSeq(reps[0].ASCodecs, ",") {
		c = strings.TrimSpace(c)
		profile, level, ok := codecsProfileLevel(c)
		if !ok || codecFamily(c) != codec {
			continue
		}
		if profile != maxProfile {
			f = append(f, RuleIOPASCodecs.New(loc, "@codecs %q has profile %d, highest in the Representations is %d", c, profile, maxProfile))
		}
		if level != maxLevel {
			f = append(f, RuleIOPASCodecs.New(loc, "@codecs %q has level %d, highest in the Representations is %d", c, level, maxLevel))
		}
	}
	return f
}

func codecFamily(c string) string {
	switch {
	case strings.HasPrefix(c, "avc"):
		return "avc"
	case strings.HasPrefix(c, "hvc"), strings.HasPrefix(c, "hev"):
		return "hevc"
	}
	return ""
}

// codecsProfileLevel extracts profile and level from an AVC (avc1.PPCCLL)
// or HEVC (hvc1.P.C.TLL...) codecs string.
func codecsProfileLevel(c string) (profile, level int, ok bool) {
	switch codecFamily(c) {
	case "avc":
		_, hex, found := strings.Cut(c, ".")
		if !found || len(hex) != 6 {
			return 0, 0, false
		}
		p, err1 := strconv.ParseUint(hex[0:2], 16, 8)
		l, err2 := strconv.ParseUint(hex[4:6], 16, 8)
		return int(p), int(l), err1 == nil && err2 == nil
	case "hevc":
		parts := strings.Split(c, ".")
		if len(parts) < 4 || len(parts[3]) < 2 {
			return 0, 0, false
		}
		ps := strings.TrimLeft(parts[1], "ABC") // general_profile_space prefix
		p, err1 := strconv.Atoi(ps)
		l, err2 := strconv.Atoi(parts[3][1:]) // L93 or H120
		return p, l, err1 == nil && err2 == nil
	}
	return 0, 0, false
}

// String describes the configuration, for messages and debugging.
func (v videoConfig) String() string {
	return fmt.Sprintf("%s profile %d level %d", v.codec, v.profile, v.level)
}
