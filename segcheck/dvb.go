package segcheck

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Segment checks ported from the DVB module of the DASH-IF Conformance tool
// (ETSI TS 103 285). They apply to Representations with a DVB-DASH profile.

const (
	profileDVB2014     = "urn:dvb:dash:profile:dvb-dash:2014"
	profileDVB2017     = "urn:dvb:dash:profile:dvb-dash:2017"
	profileDVBOnDemand = "urn:dvb:dash:profile:dvb-dash:isoff-ext-on-demand:2014"
	refDVB             = "ETSI TS 103 285"
)

var (
	RuleDVBTraf = report.Register(report.Rule{
		ID: "SEG-DVB-TRAF", Severity: report.Error,
		Desc: "Each moof box must contain exactly one traf box", SpecRef: refDVB + " 4.3",
	})
	RuleDVBSidx = report.Register(report.Rule{
		ID: "SEG-DVB-SIDX", Severity: report.Error,
		Desc: "DVB on-demand Representations must have exactly one sidx box", SpecRef: refDVB + " 4.3",
	})
	RuleDVBDuration = report.Register(report.Rule{
		ID: "SEG-DVB-DURATION", Severity: report.Error,
		Desc: "DVB segments and subsegments must be between 1 and 15 seconds long", SpecRef: refDVB + " 4.5",
	})
	RuleDVBSAP = report.Register(report.Rule{
		ID: "SEG-DVB-SAP", Severity: report.Error,
		Desc: "DVB video segments must start with a SAP of type 1 or 2", SpecRef: refDVB + " 5.1.2",
	})
	RuleDVBAVC3 = report.Register(report.Rule{
		ID: "SEG-DVB-AVC3", Severity: report.Warning,
		Desc: "DVB AVC content should use the avc3 or avc4 sample entry", SpecRef: refDVB + " 5.1.2",
	})
	RuleDVBParamSets = report.Register(report.Rule{
		ID: "SEG-DVB-PARAMSETS", Severity: report.Error,
		Desc:    "With avc1 or avc2, the initialization segment must hold all SPS and PPS needed to decode",
		SpecRef: refDVB + " 5.1.2",
	})
	RuleDVBCodec = report.Register(report.Rule{
		ID: "SEG-DVB-CODEC", Severity: report.Warning,
		Desc: "The sample entry should be a codec that DVB-DASH supports", SpecRef: refDVB + " 5, 6, 7",
	})
	RuleDVBAVC = report.Register(report.Rule{
		ID: "SEG-DVB-AVC", Severity: report.Warning,
		Desc: "A DVB Period with AVC video should have a Representation for the avc_hd_50_level40 or avc_hd_60_level40 " +
			"player conformance point (Main or High profile up to level 4.0), unless it targets devices with other capabilities",
		SpecRef: refDVB + " 5.1.1, ETSI TS 101 154 L.1",
	})
	RuleDVBHEVC = report.Register(report.Rule{
		ID: "SEG-DVB-HEVC", Severity: report.Warning,
		Desc: "A DVB Period with HEVC video should have a Representation for an HEVC player conformance point " +
			"(Main or Main 10 profile, Main tier, level 4.1 up to 1080p, 5.2 up to 2160p, 6.1 up to 4320p), unless it targets devices with other capabilities",
		SpecRef: refDVB + " 5.2.3, ETSI TS 101 154 L.1",
	})
	RuleDVBHEVCSPS = report.Register(report.Rule{
		ID: "SEG-DVB-HEVC-SPS", Severity: report.Error,
		Desc:    "DVB HEVC must be progressive frames, 8-bit for Main profile and 8 or 10 bits with equal chroma bit depth for Main 10",
		SpecRef: refDVB + " 5.2.3, ETSI TS 101 154 L.3.3.2",
	})
	RuleDVBChannels = report.Register(report.Rule{
		ID: "SEG-DVB-CHANNELS", Severity: report.Error,
		Desc:    "The MPEG AudioChannelConfiguration value must match the channel count of the sample entry",
		SpecRef: refDVB + " 6.1.1",
	})
	RuleDVBSubtitle = report.Register(report.Rule{
		ID: "SEG-DVB-SUBTITLE", Severity: report.Error,
		Desc: "DVB stpp subtitles need a subt handler and an stpp sample entry with a namespace", SpecRef: refDVB + " 7.1",
	})
	RuleDVBPssh = report.Register(report.Rule{
		ID: "SEG-DVB-PSSH", Severity: report.Info,
		Desc:    "A DRM system in the MPD has no pssh in the MPD or the initialization segment, so it must come from the media segments or elsewhere",
		SpecRef: refDVB + " 8.5.0",
	})
	RuleDVBCenc = report.Register(report.Rule{
		ID: "SEG-DVB-CENC", Severity: report.Error,
		Desc:    "DVB encrypted content must use the cenc or cbcs scheme, and cbcs not with the 2014 DVB profile",
		SpecRef: refDVB + " 8.6.0",
	})
	RuleDVBCencFallback = report.Register(report.Rule{
		ID: "SEG-DVB-CENC-FALLBACK", Severity: report.Error,
		Desc: "A DVB Period with AVC video must have an AVC video AdaptationSet and an AdaptationSet of each other " +
			"media type that is unencrypted or uses the cenc scheme",
		SpecRef: refDVB + " 8.6.0",
	})
	RuleDVBContinuityEPT = report.Register(report.Rule{
		ID: "SEG-DVB-CONTINUITY-EPT", Severity: report.Error,
		Desc:    "All Representations of a period-continuous AdaptationSet must share the earliest presentation time",
		SpecRef: refDVB + " 10.5.2.3",
	})
	RuleDVBSwitchable = report.Register(report.Rule{
		ID: "SEG-DVB-SWITCHABLE", Severity: report.Error,
		Desc:    "The Representations of a DVB AdaptationSet must have the same sample entry type, track ID and default KID",
		SpecRef: refDVB + " 4.3, 8.3",
	})
)

var dvbSampleEntries = []string{"avc", "hev1", "hvc1", "mp4a", "ec-3", "ac-4", "dtsc", "dtsh", "dtse", "dtsl", "stpp"}

func isDVB(rep *addressing.Rep) bool {
	return rep.HasProfile(profileDVB2014) || rep.HasProfile(profileDVB2017)
}

func dvbInitRules(rep *addressing.Rep, init *Init, info sampleEntryInfo, loc report.Location) []report.Finding {
	if !isDVB(rep) {
		return nil
	}
	var f []report.Finding
	fourCC := info.fourCC
	// DVB-88
	if !slices.ContainsFunc(dvbSampleEntries, func(p string) bool { return strings.HasPrefix(fourCC, p) }) {
		f = append(f, RuleDVBCodec.New(loc, "sample entry %s", fourCC))
	}
	v := info.video
	switch {
	case strings.HasPrefix(fourCC, "avc"):
		// DVB-73, DVB-74
		if fourCC == "avc1" || fourCC == "avc2" {
			f = append(f, RuleDVBAVC3.New(loc, "sample entry %s", fourCC))
			if !v.hasSPS || !v.hasPPS {
				f = append(f, RuleDVBParamSets.New(loc, "%s sample entry without SPS and PPS in avcC", fourCC))
			}
		}
	case fourCC == "hvc1" || fourCC == "hev1":
		f = append(f, dvbHEVCInitRules(rep, info, loc)...)
		// DVB-92 (upstream only checks for 8 or 10 bits)
		if v.hasConfig {
			const progressive, interlaced, frameOnly = 1 << 47, 1 << 46, 1 << 44
			if v.constraints&progressive == 0 || v.constraints&interlaced != 0 || v.constraints&frameOnly == 0 {
				f = append(f, RuleDVBHEVCSPS.New(loc, "general_progressive_source_flag, general_interlaced_source_flag and "+
					"general_frame_only_constraint_flag must be 1, 0 and 1 (constraint flags %012x)", v.constraints))
			}
			switch {
			case v.hevcCompatible(1) && (v.bitDepth != 8 || v.chromaBitDepth != 8):
				f = append(f, RuleDVBHEVCSPS.New(loc, "Main profile with %d-bit luma and %d-bit chroma", v.bitDepth, v.chromaBitDepth))
			case v.hevcCompatible(2) && (v.bitDepth != 8 && v.bitDepth != 10 || v.chromaBitDepth != v.bitDepth):
				f = append(f, RuleDVBHEVCSPS.New(loc, "Main 10 profile with %d-bit luma and %d-bit chroma", v.bitDepth, v.chromaBitDepth))
			}
		}
	}
	// DVB-84 (upstream also fails when another channel configuration scheme is used)
	if init.Handler == "soun" {
		for _, acc := range rep.AudioChannelConfigs {
			if acc.SchemeIdUri == "urn:mpeg:dash:23003:3:audio_channel_configuration:2011" {
				if n, err := strconv.Atoi(acc.Value); err == nil && n != int(info.channels) {
					f = append(f, RuleDVBChannels.New(loc, "AudioChannelConfiguration %d but the sample entry has %d channels", n, info.channels))
				}
			}
		}
	}
	// DVB-98 to DVB-100 (upstream only checks @codecs equal to "stpp")
	if strings.HasPrefix(rep.Codecs, "stpp") {
		if init.Handler != "subt" {
			f = append(f, RuleDVBSubtitle.New(loc, "handler %q, not subt", init.Handler))
		}
		switch {
		case fourCC != "stpp":
			f = append(f, RuleDVBSubtitle.New(loc, "sample entry %s, not stpp", fourCC))
		case info.stppNS == "":
			f = append(f, RuleDVBSubtitle.New(loc, "stpp sample entry without namespace"))
		}
	}
	if init.Encrypted {
		// DVB-78 (upstream asks for cenc, as before TS 103 285 V1.3.1 allowed cbcs)
		switch {
		case init.Scheme != "cenc" && init.Scheme != "cbcs":
			f = append(f, RuleDVBCenc.New(loc, "scheme %q", init.Scheme))
		case init.Scheme == "cbcs" && rep.HasProfile(profileDVB2014):
			f = append(f, RuleDVBCenc.New(loc, "cbcs scheme with the profile %s", profileDVB2014))
		}
		// DVB-76 (upstream: a SHOULD that TS 103 285 V1.4.1 does not have)
		for _, cp := range rep.ContentProtections {
			id, isSystem := strings.CutPrefix(strings.ToLower(string(cp.SchemeIdUri)), "urn:uuid:")
			if !isSystem || cp.Pssh != nil || slices.Contains(init.PsshSystems, id) {
				continue
			}
			f = append(f, RuleDVBPssh.New(loc, "DRM system %s has no pssh in the MPD or the initialization segment", id))
		}
	}
	return f
}

func dvbMediaRules(rep *addressing.Rep, init *Init, media *Media, mf *mp4.File, mediaTS float64, last bool, loc report.Location) []report.Finding {
	if !isDVB(rep) || media == nil {
		return nil
	}
	var f []report.Finding
	if init.Video.codec == "hevc" {
		f = append(f, dvbHEVCMediaRules(rep, init, mf, loc)...)
	}
	// DVB-71
	for _, ms := range mf.Segments {
		for fi, frag := range ms.Fragments {
			if frag.Moof != nil && len(frag.Moof.Trafs) != 1 {
				f = append(f, RuleDVBTraf.New(loc, "moof %d has %d traf boxes", fi, len(frag.Moof.Trafs)))
			}
		}
	}
	// DVB-86, DVB-87
	dur := float64(media.Duration) / mediaTS
	if dur > 15 {
		f = append(f, RuleDVBDuration.New(loc, "%.3fs segment", dur))
	}
	if dur < 1 && !last {
		f = append(f, RuleDVBDuration.New(loc, "%.3fs segment", dur))
	}
	// DVB-95; with @startWithSAP 1 or 2, SEG-SAP already reports this.
	if init.Handler == "vide" && !media.FirstSync && rep.StartWithSAP != 1 && rep.StartWithSAP != 2 {
		f = append(f, RuleDVBSAP.New(loc, "segment does not start with a sync sample"))
	}
	return f
}

// CheckDVBSidx checks the sidx box of a DVB on-demand Representation (DVB-72, DVB-86, DVB-87, DVB-95).
func CheckDVBSidx(rep *addressing.Rep, sidx *mp4.SidxBox, nSidx int) []report.Finding {
	if !isDVB(rep) || rep.Kind != addressing.KindSegmentBase || sidx == nil {
		return nil
	}
	var f []report.Finding
	loc := rep.Loc()
	if rep.HasProfile(profileDVBOnDemand) && nSidx != 1 {
		f = append(f, RuleDVBSidx.New(loc, "%d sidx boxes", nSidx))
	}
	for i, ref := range sidx.SidxRefs {
		d := float64(ref.SubSegmentDuration) / float64(sidx.Timescale)
		if d > 15 || (d < 1 && i < len(sidx.SidxRefs)-1) {
			f = append(f, RuleDVBDuration.New(loc, "subsegment %d is %.3fs", i, d))
			break
		}
	}
	for i, ref := range sidx.SidxRefs {
		if ref.StartsWithSAP != 1 || (ref.SAPType != 1 && ref.SAPType != 2) {
			f = append(f, RuleDVBSAP.New(loc, "subsegment %d does not start with SAP type 1 or 2", i))
			break
		}
	}
	return f
}

// dvbAdaptationSetRules ports DVB-101 to DVB-103.
func dvbAdaptationSetRules(reps []*addressing.Rep, inits []*Init) []report.Finding {
	if !isDVB(reps[0]) {
		return nil
	}
	entries, tracks, kids := map[string]bool{}, map[uint32]bool{}, map[string]bool{}
	encrypted, n := false, 0
	for _, init := range inits {
		if init == nil {
			continue
		}
		n++
		entries[init.SampleEntry] = true
		tracks[init.TrackID] = true
		if init.Encrypted {
			encrypted = true
			kids[init.DefaultKID] = true
		}
	}
	var f []report.Finding
	loc := report.Location{Path: reps[0].ASPath}
	if len(entries) > 1 {
		f = append(f, RuleDVBSwitchable.New(loc, "%d different sample entry types", len(entries)))
	}
	if len(tracks) > 1 {
		f = append(f, RuleDVBSwitchable.New(loc, "%d different track IDs", len(tracks)))
	}
	if encrypted && len(kids) != 1 {
		f = append(f, RuleDVBSwitchable.New(loc, "%d different default KIDs, or a mix of clear and encrypted tracks", len(kids)))
	}
	return f
}

// hevcCompatible reports whether HEVC video conforms to the profile with
// general_profile_idc p, directly or through a compatibility flag.
func (v videoConfig) hevcCompatible(p int) bool {
	return v.profile == p || v.compatibility&(1<<(31-p)) != 0
}

// avcConformancePoint reports whether avc_hd_50_level40 or avc_hd_60_level40
// players (H.264 Main and High profile up to level 4.0, up to 1920x1080) can
// decode the video. constraint_set1_flag marks a stream that obeys the Main
// profile constraints, such as Constrained Baseline.
func (v videoConfig) avcConformancePoint() bool {
	const constraintSet1 = 1 << 6
	mainOrHigh := v.profile == 77 || v.profile == 100 || v.constraints&constraintSet1 != 0
	return mainOrHigh && v.level <= 40 && v.width <= 1920 && v.height <= 1080
}

// hevcConformancePoint reports whether the players of at least one HEVC
// conformance point in ETSI TS 101 154 table L.1 can decode the video. All
// of them decode Main and Main 10 profile at Main tier; the level they reach
// depends on the resolution.
func (v videoConfig) hevcConformancePoint() bool {
	if v.highTier || !v.hevcCompatible(1) && !v.hevcCompatible(2) {
		return false
	}
	switch {
	case v.width <= 1920 && v.height <= 1080:
		return v.level <= 123
	case v.width <= 3840 && v.height <= 2160:
		return v.level <= 156
	case v.width <= 7680 && v.height <= 4320:
		return v.level <= 183
	}
	return false
}

// dvbPeriodRules checks that each Period has AVC and HEVC video that the
// players of a DVB conformance point can decode (DVB-89 to DVB-94). Upstream
// checks each Representation against fixed profile, tier and level lists.
func dvbPeriodRules(reps []*addressing.Rep, inits []*Init) []report.Finding {
	var f []report.Finding
	type state struct{ found, unknown, any bool }
	for start := 0; start < len(reps); {
		end := start
		for end < len(reps) && reps[end].PeriodIndex == reps[start].PeriodIndex {
			end++
		}
		var avc, hevc state
		for i := start; i < end; i++ {
			if !isDVB(reps[i]) {
				continue
			}
			init := inits[i]
			if init == nil {
				// An unchecked Representation may be the decodable one.
				switch {
				case strings.HasPrefix(reps[i].Codecs, "avc"):
					avc.unknown = true
				case strings.HasPrefix(reps[i].Codecs, "hev1"), strings.HasPrefix(reps[i].Codecs, "hvc1"):
					hevc.unknown = true
				}
				continue
			}
			v := init.Video
			switch {
			case !v.hasConfig:
			case v.codec == "avc":
				avc.any = true
				avc.found = avc.found || v.avcConformancePoint()
			case v.codec == "hevc":
				hevc.any = true
				hevc.found = hevc.found || v.hevcConformancePoint()
			}
		}
		loc := report.Location{Path: fmt.Sprintf("Period[%d]", reps[start].PeriodIndex)}
		if avc.any {
			f = append(f, dvbClearFallback(reps[start:end], inits[start:end], loc)...)
		}
		if avc.any && !avc.found && !avc.unknown {
			f = append(f, RuleDVBAVC.New(loc, "no AVC Representation is Main or High profile up to level 4.0 and 1920x1080"))
		}
		if hevc.any && !hevc.found && !hevc.unknown {
			f = append(f, RuleDVBHEVC.New(loc, "no HEVC Representation fits an HEVC player conformance point"))
		}
		start = end
	}
	return f
}

// dvbClearFallback checks that a Period with AVC video can be played by
// players without cbcs support: it needs an AVC video AdaptationSet and an
// AdaptationSet of each other media type that is unencrypted or uses cenc.
func dvbClearFallback(reps []*addressing.Rep, inits []*Init, loc report.Location) []report.Finding {
	type set struct {
		media        string
		avc, clear   bool
		unknown, dvb bool
	}
	var sets []*set
	byPath := map[string]*set{}
	for i, rep := range reps {
		s := byPath[rep.ASPath]
		if s == nil {
			s = &set{clear: true}
			byPath[rep.ASPath] = s
			sets = append(sets, s)
		}
		s.dvb = s.dvb || isDVB(rep)
		init := inits[i]
		if init == nil {
			s.unknown = true
			continue
		}
		s.media = init.Handler
		if init.Video.codec == "avc" {
			s.avc = true
		}
		if init.Encrypted && init.Scheme != "cenc" {
			s.clear = false
		}
	}
	need := map[string]bool{} // media types without a playable AdaptationSet
	for _, s := range sets {
		if s.unknown {
			return nil // an unchecked AdaptationSet may be the playable one
		}
		if !s.dvb || s.media == "" {
			continue
		}
		key := s.media
		if s.media == "vide" {
			if !s.avc {
				continue
			}
			key = "AVC video"
		}
		if _, seen := need[key]; !seen {
			need[key] = true
		}
		if s.clear {
			need[key] = false
		}
	}
	var f []report.Finding
	for _, key := range slices.Sorted(maps.Keys(need)) {
		if need[key] {
			f = append(f, RuleDVBCencFallback.New(loc, "every %s AdaptationSet uses the cbcs scheme", mediaName(key)))
		}
	}
	return f
}

// mediaName turns a handler type into words for messages.
func mediaName(handler string) string {
	switch handler {
	case "soun":
		return "audio"
	case "subt", "text":
		return "subtitle"
	}
	return handler
}

var continuitySchemes = []string{"urn:dvb:dash:period_continuity:2014", "urn:mpeg:dash:period-continuity:2015"}

// dvbContinuityEPT checks DVB-80 and DVB-81: when an AdaptationSet continues
// one in an earlier Period, the Representations of each share the same
// earliest presentation time (EPT1 and EPT2). Only Representations whose
// first segment of the Period was checked take part.
func dvbContinuityEPT(reps []*addressing.Rep, firstEPTs []float64) []report.Finding {
	const tolerance = 0.001
	var f []report.Finding
	reported := map[string]bool{}
	check := func(asPath string, idx []int, which string) {
		if reported[asPath] {
			return
		}
		reported[asPath] = true
		var known []string
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, i := range idx {
			if math.IsNaN(firstEPTs[i]) {
				continue
			}
			lo, hi = min(lo, firstEPTs[i]), max(hi, firstEPTs[i])
			known = append(known, fmt.Sprintf("%s %.3fs", reps[i].ID, firstEPTs[i]))
		}
		if len(known) > 1 && hi-lo > tolerance {
			f = append(f, RuleDVBContinuityEPT.New(report.Location{Path: asPath}, "%s differs: %s", which, strings.Join(known, ", ")))
		}
	}
	byAS := map[string][]int{}
	var order []string
	for i, r := range reps {
		if _, ok := byAS[r.ASPath]; !ok {
			order = append(order, r.ASPath)
		}
		byAS[r.ASPath] = append(byAS[r.ASPath], i)
	}
	for _, asPath := range order {
		idx := byAS[asPath]
		r := reps[idx[0]]
		if !isDVB(r) || r.ASID == "" {
			continue
		}
		for _, d := range r.Supplementals {
			if !slices.Contains(continuitySchemes, string(d.SchemeIdUri)) {
				continue
			}
			for _, prevPath := range order {
				p := reps[byAS[prevPath][0]]
				if p.PeriodID == d.Value && p.ASID == r.ASID && p.PeriodIndex < r.PeriodIndex {
					check(prevPath, byAS[prevPath], "EPT1")
					check(asPath, idx, "EPT2")
				}
			}
		}
	}
	return f
}
