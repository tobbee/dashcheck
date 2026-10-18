package segcheck

import (
	"slices"
	"strconv"

	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// CMAF media profile checks ported from the CMAF module of the DASH-IF
// Conformance tool (ISO/IEC 23000-19 Annexes A and B).

const profileCMFHDPresentation = "urn:mpeg:cmaf:presentation_profile:cmfhd:2016"

var (
	RuleCMAFMediaProfile = report.Register(report.Rule{
		ID: "SEG-CMAF-MEDIAPROFILE", Severity: report.Error,
		Desc:    "Tracks must not exceed the encoding parameters of the CMAF media profile brand they signal",
		SpecRef: refCMAF + " A.2, A.3, B.6",
	})
	RuleCMAFPresentationProfile = report.Register(report.Rule{
		ID: "SEG-CMAF-PRESENTATIONPROFILE", Severity: report.Error,
		Desc:    "With the cmfhd presentation profile, video switching sets need a cfhd track and audio switching sets a caac track",
		SpecRef: refCMAF + " A.1.2",
	})
	RuleCMAFHEVCSwitching = report.Register(report.Rule{
		ID: "SEG-CMAF-HEVC-SWITCHING", Severity: report.Error,
		Desc:    "HEVC tracks of a CMAF switching set must have identical SPS colour and extension information and SEI",
		SpecRef: refCMAF + " B.2.4",
	})
	RuleCMAFSubtitleProfile = report.Register(report.Rule{
		ID: "SEG-CMAF-SUBTITLEPROFILE", Severity: report.Error,
		Desc:    "All subtitle tracks of a CMAF switching set must conform to one media profile",
		SpecRef: refCMAF + " 7.3.4.1",
	})
)

// videoProfile holds the limits of a CMAF video media profile.
type videoProfile struct {
	codec                       string
	profiles                    []int // allowed AVC profile_idc or HEVC general_profile_idc
	maxLevel                    int
	primaries, transfer, matrix []int
	maxWidth, maxHeight         int
	maxFrameRate                float64
}

// videoProfiles are the brands checked by the DASH-IF tool. Its cud8 entry
// asks for an HEVC profile named "Main8", which never matches; dashcheck uses Main (1).
var videoProfiles = map[string]videoProfile{
	"cfsd": {"avc", []int{66, 77, 100}, 31, []int{1, 5, 6}, []int{1, 6}, []int{1, 5, 6}, 864, 576, 60},
	"cfhd": {"avc", []int{66, 77, 100}, 40, []int{1}, []int{1}, []int{1}, 1920, 1080, 60},
	"chdf": {"avc", []int{66, 77, 100}, 42, []int{1}, []int{1}, []int{1}, 1920, 1080, 60},
	"chhd": {"hevc", []int{1}, 123, []int{1}, []int{1}, []int{1}, 1920, 1080, 60},
	"chh1": {"hevc", []int{2}, 123, []int{1}, []int{1}, []int{1}, 1920, 1080, 60},
	"cud8": {"hevc", []int{1}, 150, []int{1}, []int{1}, []int{1}, 3840, 2160, 60},
	"cud1": {"hevc", []int{2}, 153, []int{1}, []int{1}, []int{1}, 3840, 2160, 60},
	"chr1": {"hevc", []int{2}, 153, []int{1}, []int{1}, []int{1}, 3840, 2160, 60},
}

// mediaProfileBrand returns the CMAF media profile brand of a track, or "".
func mediaProfileBrand(brands []string, handler, fourCC string) string {
	switch handler {
	case "vide":
		for _, b := range []string{"cfsd", "cfhd", "chdf", "chhd", "chh1", "cud8", "cud1", "chr1"} {
			if slices.Contains(brands, b) {
				return b
			}
		}
	case "soun":
		if slices.Contains(brands, "caac") {
			return "caac"
		}
	case "subt":
		for _, b := range []string{"im1i", "im1t"} {
			if slices.Contains(brands, b) {
				return b
			}
		}
	case "text":
		if fourCC == "wvtt" {
			return "cwvt"
		}
	}
	return ""
}

// cmafMediaProfileRules checks a track against every media profile brand it signals (CMAF-1, CMAF-22).
func cmafMediaProfileRules(rep *addressing.Rep, init *Init, info sampleEntryInfo, loc report.Location) []report.Finding {
	var f []report.Finding
	for _, brand := range init.Brands {
		if vp, ok := videoProfiles[brand]; ok && vp.codec == info.video.codec {
			f = append(f, checkVideoProfile(brand, vp, rep, info, loc)...)
		}
		if brand == "caac" && info.aac != nil {
			a := info.aac
			if a.sampleRate > 48000 {
				f = append(f, RuleCMAFMediaProfile.New(loc, "caac brand but sample rate %d Hz is above 48000", a.sampleRate))
			}
			if a.channels > 2 {
				f = append(f, RuleCMAFMediaProfile.New(loc, "caac brand but channel configuration %d is more than stereo", a.channels))
			}
			if !slices.Contains([]int{2, 5, 29}, a.objectType) {
				f = append(f, RuleCMAFMediaProfile.New(loc, "caac brand but audio object type %d is not 2, 5 or 29", a.objectType))
			}
		}
	}
	return f
}

func checkVideoProfile(brand string, vp videoProfile, rep *addressing.Rep, info sampleEntryInfo, loc report.Location) []report.Finding {
	var f []report.Finding
	v := info.video
	if !slices.Contains(vp.profiles, v.profile) {
		f = append(f, RuleCMAFMediaProfile.New(loc, "%s brand but profile %d is not one of %v", brand, v.profile, vp.profiles))
	}
	if v.level > vp.maxLevel {
		f = append(f, RuleCMAFMediaProfile.New(loc, "%s brand but level %d is above %d", brand, v.level, vp.maxLevel))
	}
	if v.highTier {
		f = append(f, RuleCMAFMediaProfile.New(loc, "%s brand but HEVC high tier", brand))
	}
	// Upstream treats absent colour information as 1 (BT.709); dashcheck only checks signaled values.
	if v.colourDescribed {
		if !slices.Contains(vp.primaries, v.primaries) {
			f = append(f, RuleCMAFMediaProfile.New(loc, "%s brand but colour primaries %d are not one of %v", brand, v.primaries, vp.primaries))
		}
		if !slices.Contains(vp.transfer, v.transfer) {
			f = append(f, RuleCMAFMediaProfile.New(loc, "%s brand but transfer characteristics %d are not one of %v", brand, v.transfer, vp.transfer))
		}
		if !slices.Contains(vp.matrix, v.matrix) {
			f = append(f, RuleCMAFMediaProfile.New(loc, "%s brand but matrix coefficients %d are not one of %v", brand, v.matrix, vp.matrix))
		}
	}
	if int(info.width) > vp.maxWidth || int(info.height) > vp.maxHeight {
		f = append(f, RuleCMAFMediaProfile.New(loc, "%s brand but size %dx%d exceeds %dx%d", brand, info.width, info.height, vp.maxWidth, vp.maxHeight))
	}
	// Upstream never checks the frame rate. dashcheck uses the SPS timing information, or else @frameRate.
	fps := v.frameRate
	if fps == 0 {
		fps, _ = parseRate(rep.FrameRate)
	}
	if fps > vp.maxFrameRate {
		f = append(f, RuleCMAFMediaProfile.New(loc, "%s brand but frame rate %.3f is above %.0f", brand, fps, vp.maxFrameRate))
	}
	return f
}

// parseRate parses a frame rate like "30" or "30000/1001".
func parseRate(s string) (float64, bool) {
	num, den, found := cut(s, '/')
	n, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, false
	}
	if !found {
		return n, true
	}
	d, err := strconv.ParseFloat(den, 64)
	if err != nil || d == 0 {
		return 0, false
	}
	return n / d, true
}

func cut(s string, sep byte) (before, after string, found bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

// cmafMediaProfileSetRules checks a switching set (CMAF-2, CMAF-9, CMAF-10, CMAF-20, CMAF-23).
func cmafMediaProfileSetRules(reps []*addressing.Rep, inits []*Init) []report.Finding {
	var f []report.Finding
	loc := report.Location{Path: reps[0].ASPath}
	var tracks []*Init
	for _, init := range inits {
		if init != nil && init.CMAF {
			tracks = append(tracks, init)
		}
	}
	if len(tracks) == 0 {
		return nil
	}
	handler := tracks[0].Handler
	if reps[0].HasProfile(profileCMFHDPresentation) {
		need := map[string]string{"vide": "cfhd", "soun": "caac"}[handler]
		if need != "" && !slices.ContainsFunc(tracks, func(i *Init) bool { return slices.Contains(i.Brands, need) }) {
			f = append(f, RuleCMAFPresentationProfile.New(loc, "cmfhd presentation profile but no %s track in the switching set", need))
		}
	}
	if handler == "vide" && tracks[0].Video.codec == "hevc" {
		for _, t := range tracks[1:] {
			if t.Video.spsSummary != tracks[0].Video.spsSummary {
				f = append(f, RuleCMAFHEVCSwitching.New(loc, "SPS colour or extension information differs between tracks"))
				break
			}
		}
		for _, t := range tracks[1:] {
			if t.Video.seiNalus != tracks[0].Video.seiNalus {
				f = append(f, RuleCMAFHEVCSwitching.New(loc, "SEI NAL units in hvcC differ between tracks"))
				break
			}
		}
	}
	if handler == "subt" || handler == "text" {
		for _, t := range tracks[1:] {
			if t.cmafBrand != tracks[0].cmafBrand {
				f = append(f, RuleCMAFSubtitleProfile.New(loc, "subtitle tracks with media profiles %q and %q", tracks[0].cmafBrand, t.cmafBrand))
				break
			}
		}
	}
	return f
}
