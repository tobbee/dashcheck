package mpdcheck

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/report"
)

// Profile-specific rules ported from the DASH-IF Schematron rules
// (schematron.sch): MPEG-DASH profiles and the DASH-IF Interoperability Points.

const (
	profileOnDemand    = "urn:mpeg:dash:profile:isoff-on-demand:2011"
	profileLive        = "urn:mpeg:dash:profile:isoff-live:2011"
	profileExtLive     = "urn:mpeg:dash:profile:isoff-ext-live:2014"
	profileExtOnDemand = "urn:mpeg:dash:profile:isoff-ext-on-demand:2014"
	profileIOPOnDemand = "http://dashif.org/guidelines/dash-if-ondemand"
	profileAC4         = "http://dashif.org/guidelines/dashif#ac-4"
	profileMPEGH       = "http://dashif.org/guidelines/dashif#mpeg-h-3da"
	// iopPrefix matches all DASH-IF IOP profiles (dash264, dash-if-simple, ...),
	// as the upstream rules do.
	iopPrefix = "http://dashif.org/guidelines/dash"

	refProfiles = "ISO/IEC 23009-1 8"
	refIOP      = "DASH-IF IOP 3.2"
)

var (
	RuleProfileType = report.Register(report.Rule{
		ID: "MPD-PROFILE-TYPE", Severity: report.Error,
		Desc: "On-demand profiles require a static MPD", SpecRef: refProfiles,
	})
	RuleProfileExtLiveType = report.Register(report.Rule{
		ID: "MPD-PROFILE-EXTLIVE-TYPE", Severity: report.Warning,
		Desc: "The DASH-IF rules require a static MPD for the extended live profile", SpecRef: refProfiles,
	})
	RuleProfileSegInfo = report.Register(report.Rule{
		ID: "MPD-PROFILE-SEGINFO", Severity: report.Error,
		Desc: "Segment information must be allowed by the profile", SpecRef: refProfiles,
	})
	RuleProfileMainRole = report.Register(report.Rule{
		ID: "MPD-PROFILE-MAINROLE", Severity: report.Error,
		Desc:    "With several video AdaptationSets in a Period, one must have Role main (extended and IOP profiles)",
		SpecRef: refProfiles,
	})
	RuleProfileSubset = report.Register(report.Rule{
		ID: "MPD-PROFILE-SUBSET", Severity: report.Error,
		Desc: "@profiles must be a subset of the @profiles on higher levels", SpecRef: refCommon,
	})
	RuleIOPStatic = report.Register(report.Rule{
		ID: "MPD-IOP-STATIC", Severity: report.Error,
		Desc: "A static IOP MPD must not have @timeShiftBufferDepth or @minimumUpdatePeriod", SpecRef: refIOP,
	})
	RuleIOPLive = report.Register(report.Rule{
		ID: "MPD-IOP-LIVE", Severity: report.Error,
		Desc:    "A dynamic IOP MPD must use the live profile, and live AdaptationSets must have @segmentAlignment true",
		SpecRef: "DASH-IF IOP 3.2.2.2",
	})
	RuleIOPMinBuffer = report.Register(report.Rule{
		ID: "MPD-IOP-MINBUFFER", Severity: report.Error,
		Desc: "An IOP MPD must have @minBufferTime", SpecRef: "DASH-IF IOP 3.10.2",
	})
	RuleIOPSwitching = report.Register(report.Rule{
		ID: "MPD-IOP-SWITCHING", Severity: report.Error,
		Desc:    "AdaptationSets that can be switched between must have the same segment alignment",
		SpecRef: "DASH-IF IOP 3.8",
	})
	RuleIOPVideoAttrs = report.Register(report.Rule{
		ID: "MPD-IOP-VIDEO", Severity: report.Error,
		Desc:    "IOP video AdaptationSets and Representations must signal @par, size, frame rate and @sar",
		SpecRef: "DASH-IF IOP 3.2.4",
	})
	RuleIOPScanType = report.Register(report.Rule{
		ID: "MPD-IOP-SCANTYPE", Severity: report.Error,
		Desc: "IOP video must be progressive if @scanType is present", SpecRef: "DASH-IF IOP 3.2.4",
	})
	RuleIOPAudioAttrs = report.Register(report.Rule{
		ID: "MPD-IOP-AUDIO", Severity: report.Error,
		Desc:    "IOP audio must signal @lang, @audioSamplingRate and AudioChannelConfiguration",
		SpecRef: "DASH-IF IOP 3.2.4",
	})
	RuleIOPOnDemand = report.Register(report.Rule{
		ID: "MPD-IOP-ONDEMAND", Severity: report.Error,
		Desc:    "IOP on-demand content needs SegmentBase@indexRange, subsegment alignment and subsegments starting with SAP 1 or 2",
		SpecRef: "DASH-IF IOP 3.10.3",
	})
	RuleIOPMimeType = report.Register(report.Rule{
		ID: "MPD-IOP-MIMETYPE", Severity: report.Error,
		Desc:    "IOP AdaptationSets must use an allowed @mimeType, the same for all Representations",
		SpecRef: "DASH-IF IOP 3.2.13",
	})
	RuleIOPCodecs = report.Register(report.Rule{
		ID: "MPD-IOP-CODECS", Severity: report.Error,
		Desc: "IOP @codecs must have a single value", SpecRef: "DASH-IF IOP 3.2.13",
	})
	RuleIOPDependency = report.Register(report.Rule{
		ID: "MPD-IOP-DEPENDENCY", Severity: report.Error,
		Desc:    "A Dolby Vision enhancement layer @dependencyId must refer to an earlier Representation",
		SpecRef: "DASH-IF IOP 10",
	})
	RuleIOPContentProtection = report.Register(report.Rule{
		ID: "MPD-IOP-CP", Severity: report.Error,
		Desc:    "IOP ContentProtection must be on the AdaptationSet, and mp4protection must have cenc:default_KID",
		SpecRef: "DASH-IF IOP 7",
	})
	RuleAudioChannelConfig = report.Register(report.Rule{
		ID: "MPD-AUDIO-CHANNELCONFIG", Severity: report.Error,
		Desc:    "AudioChannelConfiguration must use the scheme required by the AC-4 or MPEG-H profile",
		SpecRef: "DASH-IF IOP 9",
	})
	RuleThumbnail = report.Register(report.Rule{
		ID: "MPD-THUMBNAIL", Severity: report.Error,
		Desc:    "Thumbnail tile Representations need a tile layout value, @bandwidth, @width and @height",
		SpecRef: "DASH-IF IOP 6.2.6",
	})
)

// profiles is a parsed @profiles value.
type profiles []string

func (p profiles) has(profile string) bool { return slices.Contains(p, profile) }

func (p profiles) iop() bool {
	return slices.ContainsFunc(p, func(s string) bool { return strings.HasPrefix(s, iopPrefix) })
}

// effective returns the @profiles in force: the own value if present, else the inherited one.
func effective(own m.ListOfProfilesType, inherited profiles) profiles {
	if own != "" {
		return splitProfiles(string(own))
	}
	return inherited
}

var allowedIOPMimeTypes = map[string]bool{
	"video/mp4": true, "audio/mp4": true, "application/mp4": true,
	"application/ttml+xml": true, "text/vtt": true, "image/jpeg": true,
}

var tileRe = regexp.MustCompile(`^\d+x\d+$`)

func profileRules(md *m.MPD) []report.Finding {
	var f []report.Finding
	mpdLoc := report.Location{Path: "MPD"}
	mpdP := profiles(splitProfiles(string(md.Profiles)))
	dynamic := md.GetType() == "dynamic"
	iop := mpdP.iop()

	// SCH-5, SCH-7, SCH-8
	if dynamic {
		if mpdP.has(profileOnDemand) {
			f = append(f, RuleProfileType.New(mpdLoc, "on-demand profile in a dynamic MPD"))
		}
		if mpdP.has(profileExtOnDemand) || mpdP.has(profileIOPOnDemand) {
			f = append(f, RuleProfileType.New(mpdLoc, "extended or IOP on-demand profile in a dynamic MPD"))
		}
		if mpdP.has(profileExtLive) {
			f = append(f, RuleProfileExtLiveType.New(mpdLoc, "extended live profile in a dynamic MPD"))
		}
	}
	if iop {
		// SCH-83, SCH-84
		if !dynamic && md.TimeShiftBufferDepth != nil {
			f = append(f, RuleIOPStatic.New(mpdLoc, "static MPD with @timeShiftBufferDepth"))
		}
		if !dynamic && md.MinimumUpdatePeriod != nil {
			f = append(f, RuleIOPStatic.New(mpdLoc, "static MPD with @minimumUpdatePeriod"))
		}
		// SCH-85
		if dynamic && !mpdP.has(profileLive) {
			f = append(f, RuleIOPLive.New(mpdLoc, "dynamic IOP MPD without the live profile %s", profileLive))
		}
		// SCH-86
		if md.MinBufferTime == nil {
			f = append(f, RuleIOPMinBuffer.New(mpdLoc, "no @minBufferTime"))
		}
	}

	for pi, p := range md.Periods {
		pPath := fmt.Sprintf("Period[%d]", pi)
		pLoc := report.Location{Path: pPath}
		// SCH-15, SCH-16, SCH-18, SCH-87
		onDemand := mpdP.has(profileOnDemand) || mpdP.has(profileExtOnDemand) || mpdP.has(profileIOPOnDemand)
		if onDemand && (p.SegmentList != nil || p.SegmentTemplate != nil) {
			f = append(f, RuleProfileSegInfo.New(pLoc, "on-demand profile with SegmentList or SegmentTemplate on the Period"))
		}
		if (mpdP.has(profileExtLive) || iop) && p.SegmentList != nil {
			f = append(f, RuleProfileSegInfo.New(pLoc, "SegmentList on the Period with the extended live or an IOP profile"))
		}
		// SCH-17, SCH-19, SCH-88
		if mpdP.has(profileExtLive) || mpdP.has(profileExtOnDemand) || iop {
			nVideo, hasMain := 0, false
			for _, as := range p.AdaptationSets {
				if as.ContentType == "video" {
					nVideo++
				}
				for _, r := range as.Roles {
					if r.Value == "main" {
						hasMain = true
					}
				}
			}
			if nVideo > 1 && !hasMain {
				f = append(f, RuleProfileMainRole.New(pLoc, "%d video AdaptationSets but none has Role main", nVideo))
			}
		}
		for ai, as := range p.AdaptationSets {
			asPath := fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)
			asP := effective(as.Profiles, mpdP)
			f = append(f, adaptationSetProfileRules(p, ai, as, asP, mpdP, iop, asPath)...)
			for ri, rep := range as.Representations {
				repP := effective(rep.Profiles, asP)
				f = append(f, representationProfileRules(p, as, ri, rep, repP, asP, mpdP, iop, asPath)...)
			}
		}
	}
	return f
}

func adaptationSetProfileRules(p *m.Period, ai int, as *m.AdaptationSetType, asP, mpdP profiles, iop bool, asPath string) []report.Finding {
	var f []report.Finding
	loc := report.Location{Path: asPath}
	// SCH-100
	if as.Profiles != "" {
		f = append(f, checkSubset(asP, mpdP, "MPD", loc)...)
	}
	// SCH-101, SCH-102
	if asP.has(profileIOPOnDemand) || mpdP.has(profileIOPOnDemand) {
		if !as.SubsegmentAlignment {
			f = append(f, RuleIOPOnDemand.New(loc, "IOP on-demand AdaptationSet without @subsegmentAlignment true"))
		}
		if as.SubsegmentStartsWithSAP != 1 && as.SubsegmentStartsWithSAP != 2 {
			f = append(f, RuleIOPOnDemand.New(loc, "IOP on-demand AdaptationSet with @subsegmentStartsWithSAP %d, not 1 or 2", as.SubsegmentStartsWithSAP))
		}
	}
	if iop {
		ct := string(as.ContentType)
		// SCH-91, SCH-94, SCH-95, SCH-96
		if ct == "video" {
			var missing []string
			if as.Par == "" {
				missing = append(missing, "@par")
			}
			if as.MaxWidth == 0 && as.Width == 0 {
				missing = append(missing, "@maxWidth or @width")
			}
			if as.MaxHeight == 0 && as.Height == 0 {
				missing = append(missing, "@maxHeight or @height")
			}
			if as.MaxFrameRate == "" && as.FrameRate == "" {
				missing = append(missing, "@maxFrameRate or @frameRate")
			}
			if len(missing) > 0 {
				f = append(f, RuleIOPVideoAttrs.New(loc, "video AdaptationSet without %s", strings.Join(missing, ", ")))
			}
			// SCH-92
			if as.ScanType != "" && as.ScanType != "progressive" {
				f = append(f, RuleIOPScanType.New(loc, "@scanType %q", as.ScanType))
			}
		}
		// SCH-93
		if ct == "audio" && as.Lang == "" {
			f = append(f, RuleIOPAudioAttrs.New(loc, "audio AdaptationSet without @lang"))
		}
		// SCH-97
		if mpdP.has(profileLive) && !as.SegmentAlignment {
			f = append(f, RuleIOPLive.New(loc, "live profile AdaptationSet without @segmentAlignment true"))
		}
		// SCH-98 (upstream misspells the profile as isoff-ondemand, so its rule never fires)
		if mpdP.has(profileOnDemand) && !as.SubsegmentAlignment {
			f = append(f, RuleIOPOnDemand.New(loc, "on-demand profile AdaptationSet without @subsegmentAlignment true"))
		}
		// SCH-99
		if as.MimeType != "" && !allowedIOPMimeTypes[as.MimeType] {
			f = append(f, RuleIOPMimeType.New(loc, "@mimeType %q is not allowed by the IOP", as.MimeType))
		}
		// SCH-104
		mimes := map[string]bool{}
		for _, rep := range as.Representations {
			if rep.MimeType != "" {
				mimes[rep.MimeType] = true
			}
		}
		if len(mimes) > 1 {
			f = append(f, RuleIOPMimeType.New(loc, "Representations have different @mimeType values"))
		}
		// SCH-90
		f = append(f, checkSwitchingAlignment(p, ai, as, loc)...)
		// SCH-115, SCH-116, SCH-117
		for i, cp := range as.ContentProtections {
			if cp.SchemeIdUri == "urn:mpeg:dash:mp4protection:2011" && cp.DefaultKID == "" {
				f = append(f, RuleIOPContentProtection.New(report.Location{Path: fmt.Sprintf("%s/ContentProtection[%d]", asPath, i)},
					"mp4protection ContentProtection without cenc:default_KID"))
			}
		}
	}
	// SCH-118, SCH-119
	f = append(f, checkChannelConfigScheme(as.AudioChannelConfigurations, mpdP, asPath)...)
	return f
}

func representationProfileRules(p *m.Period, as *m.AdaptationSetType, ri int, rep *m.RepresentationType, repP, asP, mpdP profiles, iop bool, asPath string) []report.Finding {
	var f []report.Finding
	rPath := fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id)
	loc := report.Location{Path: rPath, RepID: rep.Id}
	// SCH-111
	if rep.Profiles != "" {
		if as.Profiles != "" {
			f = append(f, checkSubset(repP, asP, "AdaptationSet", loc)...)
		}
		f = append(f, checkSubset(repP, mpdP, "MPD", loc)...)
	}
	// SCH-113
	for si, sr := range rep.SubRepresentations {
		if sr.Profiles != "" {
			f = append(f, checkSubset(splitProfiles(string(sr.Profiles)), repP, "Representation",
				report.Location{Path: fmt.Sprintf("%s/SubRepresentation[%d]", rPath, si), RepID: rep.Id})...)
		}
	}
	hasTemplate := rep.SegmentTemplate != nil || as.SegmentTemplate != nil || p.SegmentTemplate != nil
	// SCH-35
	if repP.has(profileLive) && !hasTemplate {
		f = append(f, RuleProfileSegInfo.New(loc, "live profile Representation without SegmentTemplate"))
	}
	repIOP := repP.iop()
	// SCH-112, SCH-114
	if repIOP && repP.has(profileOnDemand) {
		base := firstSegmentBase(rep.SegmentBase, as.SegmentBase, p.SegmentBase)
		switch {
		case base == nil:
			f = append(f, RuleIOPOnDemand.New(loc, "IOP on-demand Representation without SegmentBase"))
		case base.IndexRange == "":
			f = append(f, RuleIOPOnDemand.New(loc, "IOP on-demand SegmentBase without @indexRange"))
		}
	}
	if !iop {
		return f
	}
	ct := string(as.ContentType)
	// SCH-105, SCH-106
	if ct == "video" {
		var missing []string
		if rep.Width == 0 && as.Width == 0 {
			missing = append(missing, "@width")
		}
		if rep.Height == 0 && as.Height == 0 {
			missing = append(missing, "@height")
		}
		if rep.FrameRate == "" && as.FrameRate == "" {
			missing = append(missing, "@frameRate")
		}
		if rep.Sar == "" && as.Sar == "" {
			missing = append(missing, "@sar")
		}
		if len(missing) > 0 {
			f = append(f, RuleIOPVideoAttrs.New(loc, "video Representation without %s", strings.Join(missing, ", ")))
		}
		if ri > 0 {
			prev := as.Representations[ri-1]
			switch {
			case rep.Width != prev.Width && as.MaxWidth == 0:
				f = append(f, RuleIOPVideoAttrs.New(loc, "@width differs between Representations but the AdaptationSet has no @maxWidth"))
			case rep.Height != prev.Height && as.MaxHeight == 0:
				f = append(f, RuleIOPVideoAttrs.New(loc, "@height differs between Representations but the AdaptationSet has no @maxHeight"))
			case rep.FrameRate != prev.FrameRate && as.MaxFrameRate == "":
				f = append(f, RuleIOPVideoAttrs.New(loc, "@frameRate differs between Representations but the AdaptationSet has no @maxFrameRate"))
			}
		}
		// SCH-107
		if rep.ScanType != "" && rep.ScanType != "progressive" {
			f = append(f, RuleIOPScanType.New(loc, "@scanType %q", rep.ScanType))
		}
	}
	// SCH-108
	if ct == "audio" {
		if rep.AudioSamplingRate == nil && as.AudioSamplingRate == nil {
			f = append(f, RuleIOPAudioAttrs.New(loc, "audio Representation without @audioSamplingRate"))
		}
		if len(rep.AudioChannelConfigurations) == 0 && len(as.AudioChannelConfigurations) == 0 {
			f = append(f, RuleIOPAudioAttrs.New(loc, "audio Representation without AudioChannelConfiguration"))
		}
	}
	// SCH-109
	for _, c := range []string{rep.Codecs, as.Codecs} {
		if strings.Contains(c, ",") {
			f = append(f, RuleIOPCodecs.New(loc, "@codecs %q has more than one value", c))
			break
		}
	}
	// SCH-110
	codecs := first(rep.Codecs, as.Codecs)
	if (strings.HasPrefix(codecs, "dvhe") || strings.HasPrefix(codecs, "dvav")) && rep.DependencyId != nil {
		for dep := range strings.FieldsSeq(string(*rep.DependencyId)) {
			found := false
			for _, prev := range as.Representations[:ri] {
				if prev.Id == dep {
					found = true
				}
			}
			if !found {
				f = append(f, RuleIOPDependency.New(loc, "@dependencyId %q is not an earlier Representation in the AdaptationSet", dep))
			}
		}
	}
	// SCH-115
	if len(rep.ContentProtections) > 0 {
		f = append(f, RuleIOPContentProtection.New(loc, "ContentProtection on the Representation instead of the AdaptationSet"))
	}
	// SCH-120 to SCH-123
	for _, ep := range rep.EssentialProperties {
		if ep.SchemeIdUri != "http://dashif.org/guidelines/thumbnail_tile" {
			continue
		}
		if !tileRe.MatchString(ep.Value) {
			f = append(f, RuleThumbnail.New(loc, "thumbnail_tile value %q is not columns x rows", ep.Value))
		}
		if rep.Bandwidth == 0 {
			f = append(f, RuleThumbnail.New(loc, "thumbnail Representation without @bandwidth"))
		}
		if rep.Width == 0 && as.Width == 0 {
			f = append(f, RuleThumbnail.New(loc, "thumbnail Representation without @width"))
		}
		if rep.Height == 0 && as.Height == 0 {
			f = append(f, RuleThumbnail.New(loc, "thumbnail Representation without @height"))
		}
	}
	// SCH-118, SCH-119
	f = append(f, checkChannelConfigScheme(rep.AudioChannelConfigurations, mpdP, rPath)...)
	return f
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstSegmentBase(vals ...*m.SegmentBaseType) *m.SegmentBaseType {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

func checkSubset(own, higher profiles, level string, loc report.Location) []report.Finding {
	var f []report.Finding
	for _, p := range own {
		if !higher.has(p) {
			f = append(f, RuleProfileSubset.New(loc, "profile %q is not in the %s @profiles", p, level))
		}
	}
	return f
}

// checkSwitchingAlignment checks AdaptationSets signaled as switchable with
// the adaptation-set-switching descriptor. The upstream rule matches any
// SupplementalProperty value; dashcheck requires the switching scheme.
func checkSwitchingAlignment(p *m.Period, ai int, as *m.AdaptationSetType, loc report.Location) []report.Finding {
	if !as.SegmentAlignment {
		return nil
	}
	var f []report.Finding
	for _, sp := range as.SupplementalProperties {
		if sp.SchemeIdUri != "urn:mpeg:dash:adaptation-set-switching:2016" {
			continue
		}
		for id := range strings.SplitSeq(sp.Value, ",") {
			for _, other := range p.AdaptationSets[ai+1:] {
				if other.Id != nil && fmt.Sprint(*other.Id) == strings.TrimSpace(id) && !other.SegmentAlignment {
					f = append(f, RuleIOPSwitching.New(loc, "switchable with AdaptationSet %s, which lacks @segmentAlignment true", strings.TrimSpace(id)))
				}
			}
		}
	}
	return f
}

func checkChannelConfigScheme(acc []*m.DescriptorType, mpdP profiles, path string) []report.Finding {
	var f []report.Finding
	for i, d := range acc {
		loc := report.Location{Path: fmt.Sprintf("%s/AudioChannelConfiguration[%d]", path, i)}
		if mpdP.has(profileAC4) && d.SchemeIdUri != "tag:dolby.com,2014:dash:audio_channel_configuration:2011" {
			f = append(f, RuleAudioChannelConfig.New(loc, "AC-4 profile but scheme %q", d.SchemeIdUri))
		}
		if mpdP.has(profileMPEGH) && d.SchemeIdUri != "urn:mpeg:mpegB:cicp:ChannelConfiguration" {
			f = append(f, RuleAudioChannelConfig.New(loc, "MPEG-H profile but scheme %q", d.SchemeIdUri))
		}
	}
	return f
}
