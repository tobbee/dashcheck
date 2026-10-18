package segcheck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/Eyevinn/mp4ff/hevc"
	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// HEVC signalling in the MPD checked against the bitstream (ETSI TS 103 285
// 5.2.2 to 5.2.7) and SEI placement (5.2.4, ETSI TS 101 154 L.3.3.8).

const (
	schemeColourPrimaries   = "urn:mpeg:mpegB:cicp:ColourPrimaries"
	schemeMatrixCoeffs      = "urn:mpeg:mpegB:cicp:MatrixCoefficients"
	schemeTransferCharacter = "urn:mpeg:mpegB:cicp:TransferCharacteristics"
)

var (
	RuleDVBHEVCCodecs = report.Register(report.Rule{
		ID: "SEG-DVB-HEVC-CODECS", Severity: report.Error,
		Desc: "HEVC @codecs must follow ISO/IEC 14496-15 A.3 and signal a profile, tier, level and constraints " +
			"that the media meets",
		SpecRef: refDVB + " 5.2.2",
	})
	RuleDVBColour = report.Register(report.Rule{
		ID: "SEG-DVB-COLOUR", Severity: report.Error,
		Desc: "ColourPrimaries, MatrixCoefficients and TransferCharacteristics EssentialProperty values must " +
			"match the VUI of the video",
		SpecRef: refDVB + " 5.2.5",
	})
	RuleDVBHLG10 = report.Register(report.Rule{
		ID: "SEG-DVB-HLG10", Severity: report.Warning,
		Desc: "HLG10 video should be signalled with the 2017 profile, EssentialProperty ColourPrimaries 9, " +
			"MatrixCoefficients 9 and TransferCharacteristics 14, and SupplementalProperty TransferCharacteristics 18; " +
			"only MPDs for players known to support BT.2020 may leave out the profile and EssentialProperty descriptors",
		SpecRef: refDVB + " 5.2.6",
	})
	RuleDVBPQ10 = report.Register(report.Rule{
		ID: "SEG-DVB-PQ10", Severity: report.Error,
		Desc: "PQ10 video must be signalled with the 2017 profile and EssentialProperty ColourPrimaries 9, " +
			"MatrixCoefficients 9 and TransferCharacteristics 16",
		SpecRef: refDVB + " 5.2.7",
	})
	RuleDVBSEI = report.Register(report.Rule{
		ID: "SEG-DVB-SEI", Severity: report.Error,
		Desc: "HEVC SEI arrays must follow the parameter set arrays in hvcC; in samples, prefix SEI must follow " +
			"the parameter sets and precede the first VCL NAL unit, and suffix SEI must follow the last one",
		SpecRef: refDVB + " 5.2.4, ETSI TS 101 154 L.3.3.8",
	})
)

// hevcCodecsRe is the codecs ABNF of TS 103 285 5.2.2. Trailing zero
// constraint bytes may be left out (ISO/IEC 14496-15 E.3), here all of them.
var hevcCodecsRe = regexp.MustCompile(`^(hev1|hvc1)\.([ABC]?)(\d{1,3})\.([0-9A-Fa-f]{1,8})\.([LH])(\d{1,3})((?:\.[0-9A-Fa-f]{1,2}){0,6})$`)

// dvbHEVCInitRules checks the MPD signalling of HEVC video against the
// initialization segment (DVB-65 to DVB-70, DVB-96).
func dvbHEVCInitRules(rep *addressing.Rep, info sampleEntryInfo, loc report.Location) []report.Finding {
	v := info.video
	if !v.hasConfig {
		return nil
	}
	var f []report.Finding
	// DVB-65 (upstream requires the derived string as a prefix of @codecs).
	// Another sample entry type is reported by XCHK-CODECS.
	if strings.HasPrefix(rep.Codecs, "hev1.") || strings.HasPrefix(rep.Codecs, "hvc1.") {
		if problem := checkHEVCCodecs(rep.Codecs, v); problem != "" {
			f = append(f, RuleDVBHEVCCodecs.New(loc, "@codecs %q: %s", rep.Codecs, problem))
		}
	}
	// DVB-96, for hvcC
	if v.seiArrayEarly {
		f = append(f, RuleDVBSEI.New(loc, "an SEI array comes before a VPS, SPS or PPS array in hvcC"))
	}
	if !v.hasSPS {
		return f // hev1 with parameter sets only in the media
	}
	// DVB-66 to DVB-68
	vui := map[string]int{schemeColourPrimaries: v.primaries, schemeMatrixCoeffs: v.matrix, schemeTransferCharacter: v.transfer}
	for _, d := range rep.Essentials {
		scheme := string(d.SchemeIdUri)
		want, ok := vui[scheme]
		if !ok {
			continue
		}
		name := scheme[strings.LastIndex(scheme, ":")+1:]
		switch got, err := strconv.Atoi(d.Value); {
		case err != nil:
			f = append(f, RuleDVBColour.New(loc, "EssentialProperty %s has the value %q, not an integer", name, d.Value))
		case !v.colourDescribed:
			f = append(f, RuleDVBColour.New(loc, "EssentialProperty %s is %d, but the VUI has no colour description", name, got))
		case got != want:
			f = append(f, RuleDVBColour.New(loc, "EssentialProperty %s is %d, but the VUI has %d", name, got, want))
		}
	}
	if !v.colourDescribed {
		return f
	}
	// DVB-69, DVB-70 (upstream only reports information)
	switch {
	case v.transfer == 16:
		if missing := missingHDRSignalling(rep, 16, false); len(missing) > 0 {
			f = append(f, RuleDVBPQ10.New(loc, "PQ10 video (VUI transfer_characteristics 16) without %s", strings.Join(missing, ", ")))
		}
	case v.primaries == 9 && v.transfer == 14 && (v.hlgSEI || hasDescriptor(rep.Supplementals, schemeTransferCharacter, "18")):
		f = append(f, hlgFindings(rep, "VUI transfer_characteristics 14 with an HLG alternative_transfer_characteristics SEI", loc)...)
	}
	return f
}

// dvbHEVCMediaRules checks SEI placement in the samples of a media segment
// and looks for HLG10 signalling that is only in the media.
func dvbHEVCMediaRules(rep *addressing.Rep, init *Init, mf *mp4.File, loc report.Location) []report.Finding {
	var f []report.Finding
	v := init.Video
	lookForHLG := v.colourDescribed && v.primaries == 9 && v.transfer == 14 && !v.hlgSEI &&
		!hasDescriptor(rep.Supplementals, schemeTransferCharacter, "18")
	foundHLG := false
	sample := 0
	for _, ms := range mf.Segments {
		for _, frag := range ms.Fragments {
			samples, err := frag.GetFullSamples(init.trex)
			if err != nil {
				continue // reported by CheckMedia
			}
			for _, s := range samples {
				sample++
				if problem := seiOrder(s.Data); problem != "" {
					// DVB-96
					f = append(f, RuleDVBSEI.New(loc, "sample %d: %s", sample, problem))
					return f
				}
				if lookForHLG && !foundHLG && sample == 1 {
					foundHLG = sampleHasHLGSEI(s.Data)
				}
			}
		}
	}
	if foundHLG {
		f = append(f, hlgFindings(rep, "an HLG alternative_transfer_characteristics SEI in the media", loc)...)
	}
	return f
}

// hlgFindings checks the HLG10 signalling of TS 103 285 5.2.6.
func hlgFindings(rep *addressing.Rep, evidence string, loc report.Location) []report.Finding {
	missing := missingHDRSignalling(rep, 14, true)
	if len(missing) == 0 {
		return nil
	}
	return []report.Finding{RuleDVBHLG10.New(loc, "HLG10 video (%s) without %s", evidence, strings.Join(missing, ", "))}
}

// missingHDRSignalling lists the profile and descriptors of TS 103 285 5.2.6
// and 5.2.7 that the Representation lacks.
func missingHDRSignalling(rep *addressing.Rep, transfer int, hlg bool) []string {
	var missing []string
	if !rep.HasProfile(profileDVB2017) {
		missing = append(missing, "the profile "+profileDVB2017)
	}
	want := []struct {
		scheme, value string
	}{{schemeColourPrimaries, "9"}, {schemeMatrixCoeffs, "9"}, {schemeTransferCharacter, strconv.Itoa(transfer)}}
	for _, w := range want {
		if !hasDescriptor(rep.Essentials, w.scheme, w.value) {
			missing = append(missing, fmt.Sprintf("EssentialProperty %s=%s", w.scheme, w.value))
		}
	}
	if hlg && !hasDescriptor(rep.Supplementals, schemeTransferCharacter, "18") {
		missing = append(missing, fmt.Sprintf("SupplementalProperty %s=18", schemeTransferCharacter))
	}
	return missing
}

func hasDescriptor(ds []*m.DescriptorType, scheme, value string) bool {
	for _, d := range ds {
		if string(d.SchemeIdUri) == scheme && d.Value == value {
			return true
		}
	}
	return false
}

// checkHEVCCodecs compares an HEVC codecs string with the decoder
// configuration. The string may signal more capability than needed (TS 103
// 285 5.2.2), so the media must conform to the signalled profile, not exceed
// the tier and level, and set every signalled constraint flag.
func checkHEVCCodecs(codecs string, v videoConfig) string {
	mt := hevcCodecsRe.FindStringSubmatch(codecs)
	if mt == nil {
		return "not of the form hvc1.PROFILE.COMPATIBILITY.TIERLEVEL.CONSTRAINTS"
	}
	space := strings.Index("_ABC", mt[2]) // "" is 0
	profile, _ := strconv.Atoi(mt[3])
	highTier := mt[5] == "H"
	level, _ := strconv.Atoi(mt[6])
	var constraints uint64
	for i, b := range strings.Split(strings.TrimPrefix(mt[7], "."), ".") {
		if b == "" {
			continue
		}
		n, _ := strconv.ParseUint(b, 16, 8)
		constraints |= n << (40 - 8*i)
	}
	switch {
	case space != v.profileSpace:
		return fmt.Sprintf("profile space %d, but the media has %d", space, v.profileSpace)
	case profile > 31 || !v.hevcCompatible(profile):
		return fmt.Sprintf("profile %d, but the media is profile %d without that compatibility flag", profile, v.profile)
	case v.highTier && !highTier:
		return "Main tier, but the media is High tier"
	case highTier == v.highTier && level < v.level:
		return fmt.Sprintf("level %d, but the media has level %d", level, v.level)
	case constraints&^v.constraints != 0:
		return fmt.Sprintf("constraint flags %012x that the media does not set (%012x)", constraints, v.constraints)
	}
	return ""
}

// seiOrder checks the NAL unit order of an access unit against TS 101 154
// L.3.3.8.
func seiOrder(sample []byte) string {
	types := hevc.FindNaluTypes(sample)
	lastVCL := -1
	for i, t := range types {
		if hevc.IsVideoNaluType(t) {
			lastVCL = i
		}
	}
	seenVCL, seenPrefixSEI := false, false
	for i, t := range types {
		switch {
		case hevc.IsVideoNaluType(t):
			seenVCL = true
		case t == hevc.NALU_SEI_PREFIX:
			if seenVCL {
				return "prefix SEI after a VCL NAL unit"
			}
			seenPrefixSEI = true
		case t == hevc.NALU_SEI_SUFFIX:
			if i < lastVCL {
				return "suffix SEI before the last VCL NAL unit"
			}
		case t == hevc.NALU_VPS || t == hevc.NALU_SPS || t == hevc.NALU_PPS:
			if seenPrefixSEI {
				return fmt.Sprintf("%s after a prefix SEI", t)
			}
		}
	}
	return ""
}

// sampleHasHLGSEI looks for an HLG alternative_transfer_characteristics SEI
// in the prefix SEI NAL units of a sample.
func sampleHasHLGSEI(sample []byte) bool {
	for len(sample) >= 4 {
		n := int(uint32(sample[0])<<24 | uint32(sample[1])<<16 | uint32(sample[2])<<8 | uint32(sample[3]))
		sample = sample[4:]
		if n <= 0 || n > len(sample) {
			return false
		}
		nalu := sample[:n]
		sample = sample[n:]
		if hevc.GetNaluType(nalu[0]) == hevc.NALU_SEI_PREFIX && hasHLGPreference(nalu) {
			return true
		}
	}
	return false
}
