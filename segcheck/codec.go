package segcheck

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Eyevinn/mp4ff/aac"
	"github.com/Eyevinn/mp4ff/avc"
	"github.com/Eyevinn/mp4ff/hevc"
	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/Eyevinn/mp4ff/sei"
)

// sampleEntryInfo is what the checks need from the first sample entry.
type sampleEntryInfo struct {
	fourCC        string // original format for encrypted content
	encrypted     bool
	scheme        string // protection scheme type, e.g. cenc or cbcs
	colrNclx      bool   // a colr box of type nclx is present
	defaultKID    string // tenc default KID of encrypted content
	tenc          *mp4.TencBox
	noAudioConfig string // the missing audio decoder configuration box, if any
	dac4          *mp4.Dac4Box
	stppNS        string // namespace of an stpp sample entry
	codec         string // full codecs string if it can be derived, else the 4CC
	derived       bool   // codec is a full derived string
	width         uint16
	height        uint16
	sampleRate    uint32
	channels      uint16
	video         videoConfig
	aac           *audioConfig
}

// videoConfig describes an AVC or HEVC decoder configuration record.
type videoConfig struct {
	codec          string // "avc", "hevc" or ""
	hasConfig      bool   // avcC or hvcC present
	hasSPS, hasPPS bool
	hasVPS         bool // HEVC only
	profile, level int
	constraints    uint64 // AVC constraint_set flags or HEVC general_constraint_indicator_flags
	compatibility  uint32 // HEVC general_profile_compatibility_flags
	highTier       bool   // HEVC general_tier_flag
	bitDepth       int    // HEVC luma bit depth
	chromaBitDepth int    // HEVC chroma bit depth
	width, height  int    // from the sample entry
	profileSpace   int    // HEVC general_profile_space
	seiArrayEarly  bool   // HEVC: an SEI array comes before a parameter set array in hvcC
	hlgSEI         bool   // HEVC: hvcC has an alternative_transfer_characteristics SEI preferring HLG

	// From the SPS video usability information.
	colourDescribed             bool
	primaries, transfer, matrix int
	frameRate                   float64 // 0 if not signaled
	spsSummary                  string  // HEVC SPS fields that must match in a switching set
	seiNalus                    string  // HEVC SEI NAL units in hvcC
}

// audioConfig describes an AAC AudioSpecificConfig.
type audioConfig struct {
	objectType, sampleRate, channels int
}

func describeSampleEntry(stsd *mp4.StsdBox) (sampleEntryInfo, error) {
	if stsd == nil || len(stsd.Children) == 0 {
		return sampleEntryInfo{}, fmt.Errorf("no sample entry")
	}
	var info sampleEntryInfo
	switch se := stsd.Children[0].(type) {
	case *mp4.VisualSampleEntryBox:
		info.fourCC = se.Type()
		info.width, info.height = se.Width, se.Height
		if se.Sinf != nil && se.Sinf.Frma != nil {
			info.fourCC = se.Sinf.Frma.DataFormat
			info.encrypted = true
			if se.Sinf.Schm != nil {
				info.scheme = se.Sinf.Schm.SchemeType
			}
			if se.Sinf.Schi != nil && se.Sinf.Schi.Tenc != nil {
				info.defaultKID = se.Sinf.Schi.Tenc.DefaultKID.String()
				info.tenc = se.Sinf.Schi.Tenc
			}
		}
		for _, c := range se.Children {
			if colr, ok := c.(*mp4.ColrBox); ok && colr.ColorType == "nclx" {
				info.colrNclx = true
			}
		}
		info.codec, info.derived = visualCodec(info.fourCC, se)
		info.video = describeVideoConfig(se)
	case *mp4.AudioSampleEntryBox:
		info.fourCC = se.Type()
		info.sampleRate = uint32(se.SampleRate)
		info.channels = se.ChannelCount
		if se.Sinf != nil && se.Sinf.Frma != nil {
			info.fourCC = se.Sinf.Frma.DataFormat
			info.encrypted = true
			if se.Sinf.Schm != nil {
				info.scheme = se.Sinf.Schm.SchemeType
			}
			if se.Sinf.Schi != nil && se.Sinf.Schi.Tenc != nil {
				info.defaultKID = se.Sinf.Schi.Tenc.DefaultKID.String()
				info.tenc = se.Sinf.Schi.Tenc
			}
		}
		info.codec, info.derived = audioCodec(info.fourCC, se)
		info.aac = describeAAC(se)
		switch {
		case info.fourCC == "mp4a" && info.aac == nil:
			info.noAudioConfig = "an AudioSpecificConfig in esds"
		case info.fourCC == "ec-3" && se.Dec3 == nil:
			info.noAudioConfig = "a dec3 box"
		case info.fourCC == "ac-3" && se.Dac3 == nil:
			info.noAudioConfig = "a dac3 box"
		case info.fourCC == "ac-4" && se.Dac4 == nil:
			info.noAudioConfig = "a dac4 box"
		}
		info.dac4 = se.Dac4
	case *mp4.StppBox:
		info.fourCC = se.Type()
		info.codec = info.fourCC
		info.stppNS = se.Namespace
	default:
		info.fourCC = stsd.Children[0].Type()
		info.codec = info.fourCC
	}
	return info, nil
}

func visualCodec(fourCC string, se *mp4.VisualSampleEntryBox) (string, bool) {
	switch {
	case se.AvcC != nil && len(se.AvcC.SPSnalus) > 0:
		sps, err := avc.ParseSPSNALUnit(se.AvcC.SPSnalus[0], false)
		if err == nil {
			return avc.CodecString(fourCC, sps), true
		}
	case se.HvcC != nil:
		spss := se.HvcC.GetNalusForType(hevc.NALU_SPS)
		if len(spss) > 0 {
			sps, err := hevc.ParseSPSNALUnit(spss[0])
			if err == nil {
				return hevc.CodecString(fourCC, sps), true
			}
		}
	case se.Av1C != nil:
		return se.Av1C.CodecString(fourCC), true
	}
	return fourCC, false
}

func describeVideoConfig(se *mp4.VisualSampleEntryBox) videoConfig {
	var vc videoConfig
	switch {
	case se.AvcC != nil:
		vc = videoConfig{codec: "avc", hasConfig: true, hasSPS: len(se.AvcC.SPSnalus) > 0, hasPPS: len(se.AvcC.PPSnalus) > 0,
			profile: int(se.AvcC.AVCProfileIndication), level: int(se.AvcC.AVCLevelIndication),
			constraints: uint64(se.AvcC.ProfileCompatibility)}
		if vc.hasSPS {
			if sps, err := avc.ParseSPSNALUnit(se.AvcC.SPSnalus[0], true); err == nil && sps.VUI != nil {
				v := sps.VUI
				if v.ColourDescriptionFlag {
					vc.colourDescribed = true
					vc.primaries, vc.transfer, vc.matrix = int(v.ColourPrimaries), int(v.TransferCharacteristics), int(v.MatrixCoefficients)
				}
				if v.TimingInfoPresentFlag && v.NumUnitsInTick > 0 {
					vc.frameRate = float64(v.TimeScale) / float64(2*v.NumUnitsInTick)
				}
			}
		}
	case se.HvcC != nil:
		vc = videoConfig{codec: "hevc", hasConfig: true,
			hasVPS:         len(se.HvcC.GetNalusForType(hevc.NALU_VPS)) > 0,
			hasSPS:         len(se.HvcC.GetNalusForType(hevc.NALU_SPS)) > 0,
			hasPPS:         len(se.HvcC.GetNalusForType(hevc.NALU_PPS)) > 0,
			profile:        int(se.HvcC.GeneralProfileIDC),
			level:          int(se.HvcC.GeneralLevelIDC),
			highTier:       se.HvcC.GeneralTierFlag,
			bitDepth:       int(se.HvcC.BitDepthLumaMinus8) + 8,
			chromaBitDepth: int(se.HvcC.BitDepthChromaMinus8) + 8,
			constraints:    se.HvcC.GeneralConstraintIndicatorFlags,
			compatibility:  se.HvcC.GeneralProfileCompatibilityFlags,
		}
		vc.profileSpace = int(se.HvcC.GeneralProfileSpace)
		var seiHex []string
		for _, n := range append(se.HvcC.GetNalusForType(hevc.NALU_SEI_PREFIX), se.HvcC.GetNalusForType(hevc.NALU_SEI_SUFFIX)...) {
			seiHex = append(seiHex, fmt.Sprintf("%x", n))
			vc.hlgSEI = vc.hlgSEI || hasHLGPreference(n)
		}
		vc.seiNalus = strings.Join(seiHex, ",")
		seenSEI := false
		for i := range se.HvcC.NaluArrays {
			switch se.HvcC.NaluArrays[i].NaluType() {
			case hevc.NALU_SEI_PREFIX, hevc.NALU_SEI_SUFFIX:
				seenSEI = true
			case hevc.NALU_VPS, hevc.NALU_SPS, hevc.NALU_PPS:
				vc.seiArrayEarly = vc.seiArrayEarly || seenSEI
			}
		}
		if spss := se.HvcC.GetNalusForType(hevc.NALU_SPS); len(spss) > 0 {
			if sps, err := hevc.ParseSPSNALUnit(spss[0]); err == nil {
				vc.spsSummary = hevcSPSSummary(sps)
				if v := sps.VUI; v != nil {
					if v.ColourDescriptionFlag {
						vc.colourDescribed = true
						vc.primaries, vc.transfer, vc.matrix = int(v.ColourPrimaries), int(v.TransferCharacteristics), int(v.MatrixCoefficients)
					}
					if v.TimingInfoPresentFlag && v.NumUnitsInTick > 0 {
						vc.frameRate = float64(v.TimeScale) / float64(v.NumUnitsInTick)
					}
				}
			}
		}
	}
	vc.width, vc.height = int(se.Width), int(se.Height)
	return vc
}

// hevcSPSSummary collects the SPS fields that the DASH-IF tool compares
// within a CMAF switching set.
func hevcSPSSummary(sps *hevc.SPS) string {
	s := fmt.Sprintf("vui=%v ext=%v range=%v", sps.VUI != nil, sps.ExtensionPresentFlag, sps.RangeExtensionFlag)
	if sps.RangeExtension != nil {
		s += fmt.Sprintf(" extprec=%v", sps.RangeExtension.ExtendedPrecisionProcessingFlag)
	}
	if v := sps.VUI; v != nil {
		s += fmt.Sprintf(" signal=%v colour=%v %d/%d/%d chromaloc=%v neutral=%v", v.VideoSignalTypePresentFlag,
			v.ColourDescriptionFlag, v.ColourPrimaries, v.TransferCharacteristics, v.MatrixCoefficients,
			v.ChromaLocInfoPresentFlag, v.NeutralChromaIndicationFlag)
	}
	return s
}

func describeAAC(se *mp4.AudioSampleEntryBox) *audioConfig {
	if se.Esds == nil || se.Esds.DecConfigDescriptor == nil || se.Esds.DecConfigDescriptor.ObjectType != 0x40 ||
		se.Esds.DecConfigDescriptor.DecSpecificInfo == nil {
		return nil
	}
	asc, err := aac.DecodeAudioSpecificConfig(bytes.NewReader(se.Esds.DecConfigDescriptor.DecSpecificInfo.DecConfig))
	if err != nil {
		return nil
	}
	return &audioConfig{objectType: int(asc.ObjectType), sampleRate: asc.SamplingFrequency, channels: int(asc.ChannelConfiguration)}
}

func audioCodec(fourCC string, se *mp4.AudioSampleEntryBox) (string, bool) {
	switch fourCC {
	case "mp4a":
		if se.Esds == nil || se.Esds.DecConfigDescriptor == nil {
			return fourCC, false
		}
		dcd := se.Esds.DecConfigDescriptor
		if dcd.ObjectType != 0x40 {
			return fmt.Sprintf("mp4a.%02X", dcd.ObjectType), true
		}
		if dcd.DecSpecificInfo == nil {
			return fourCC, false
		}
		asc, err := aac.DecodeAudioSpecificConfig(bytes.NewReader(dcd.DecSpecificInfo.DecConfig))
		if err != nil {
			return fourCC, false
		}
		return fmt.Sprintf("mp4a.40.%d", asc.ObjectType), true
	case "ac-3", "ec-3":
		return fourCC, true
	case "Opus":
		return "opus", true
	}
	return fourCC, false
}

// compareCodecs checks the MPD @codecs value against the codec derived from
// the sample entry. It returns "" if they agree, or a reason. major tells
// whether the sample entry type itself differs.
func compareCodecs(mpdCodecs string, info sampleEntryInfo) (reason string, major bool) {
	if mpdCodecs == "" {
		return "", false
	}
	var candidates []string
	for c := range strings.SplitSeq(mpdCodecs, ",") {
		candidates = append(candidates, strings.TrimSpace(c))
	}
	fourCC := strings.ToLower(info.fourCC)
	for _, c := range candidates {
		prefix, _, _ := strings.Cut(c, ".")
		if strings.ToLower(prefix) != fourCC {
			continue
		}
		if !info.derived || strings.EqualFold(c, info.codec) || implicitAAC(c, info.codec) {
			return "", false
		}
		return fmt.Sprintf("@codecs %q but sample entry gives %q", mpdCodecs, info.codec), false
	}
	return fmt.Sprintf("@codecs %q but sample entry is %q", mpdCodecs, info.fourCC), true
}

// implicitAAC reports whether an AAC @codecs value is consistent with the
// AudioSpecificConfig. With implicit signaling of SBR (HE-AAC, mp4a.40.5) and
// PS (HE-AACv2, mp4a.40.29), the config carries the core object type.
func implicitAAC(mpdCodecs, derived string) bool {
	switch strings.ToLower(mpdCodecs) {
	case "mp4a.40.5":
		return derived == "mp4a.40.2"
	case "mp4a.40.29":
		return derived == "mp4a.40.2" || derived == "mp4a.40.5"
	}
	return false
}

// hasHLGPreference reports whether an HEVC SEI NAL unit has an
// alternative_transfer_characteristics message with
// preferred_transfer_characteristics 18 (BT.2100 HLG).
func hasHLGPreference(nalu []byte) bool {
	if len(nalu) < 3 {
		return false
	}
	msgs, _ := sei.ExtractSEIData(bytes.NewReader(nalu[2:]))
	for _, m := range msgs {
		if m.Type() == sei.SEIAlternativeTransferCharacteristicsType && len(m.Payload()) > 0 && m.Payload()[0] == 18 {
			return true
		}
	}
	return false
}
