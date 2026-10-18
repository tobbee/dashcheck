package segcheck

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Eyevinn/mp4ff/aac"
	"github.com/Eyevinn/mp4ff/avc"
	"github.com/Eyevinn/mp4ff/hevc"
	"github.com/Eyevinn/mp4ff/mp4"
)

// sampleEntryInfo is what the checks need from the first sample entry.
type sampleEntryInfo struct {
	fourCC     string // original format for encrypted content
	encrypted  bool
	codec      string // full codecs string if it can be derived, else the 4CC
	derived    bool   // codec is a full derived string
	width      uint16
	height     uint16
	sampleRate uint32
	channels   uint16
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
		}
		info.codec, info.derived = visualCodec(info.fourCC, se)
	case *mp4.AudioSampleEntryBox:
		info.fourCC = se.Type()
		info.sampleRate = uint32(se.SampleRate)
		info.channels = se.ChannelCount
		if se.Sinf != nil && se.Sinf.Frma != nil {
			info.fourCC = se.Sinf.Frma.DataFormat
			info.encrypted = true
		}
		info.codec, info.derived = audioCodec(info.fourCC, se)
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
		if !info.derived || strings.EqualFold(c, info.codec) {
			return "", false
		}
		return fmt.Sprintf("@codecs %q but sample entry gives %q", mpdCodecs, info.codec), false
	}
	return fmt.Sprintf("@codecs %q but sample entry is %q", mpdCodecs, info.fourCC), true
}
