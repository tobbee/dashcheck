package mpdcheck

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/report"
)

// Rules ported from the Dolby-contributed DASH-IF Schematron files
// dvb-dash.sch (DVB-DASH 2017 audio and Preselection rules) and
// ac4-generic.sch (AC-4 signaling in ETSI TS 103 190-1 and -2).

const (
	profileDVB2017        = "urn:dvb:dash:profile:dvb-dash:2017"
	schemePreselection    = "urn:mpeg:dash:preselection:2016"
	schemeDolbyACC2014    = "tag:dolby.com,2014:dash:audio_channel_configuration:2011"
	schemeDolbyACC2015    = "tag:dolby.com,2015:dash:audio_channel_configuration:2015"
	schemeMPEGACC         = "urn:mpeg:mpegB:cicp:ChannelConfiguration"
	schemeAudioFrameRate  = "tag:dolby.com,2017:dash:audio_frame_rate:2017"
	schemeVirtualized     = "tag:dolby.com,2016:dash:virtualized_content:2016"
	refDVBAudio           = "ETSI TS 103 285 6.1"
	refAC4Part1           = "ETSI TS 103 190-1 F.1"
	refAC4Part2           = "ETSI TS 103 190-2 G.2"
	audioCodecPattern     = `mp4a|ac-3|ec-3|ac-4|dtsc|dtsh|dtse|dtsl`
	videoCodecPattern     = `avc|hvc|hev`
	ac4CodecItemPattern   = `^ac-4\.((02\.01\.0[0-3])|(00\.00\.0[0-3]))$`
	ac4PresentationFields = 4
)

var (
	RuleDVBAudioCommon = report.Register(report.Rule{
		ID: "MPD-DVB-AUDIO-COMMON", Severity: report.Error,
		Desc: "DVB audio Representations in an AdaptationSet must share @mimeType", SpecRef: refDVBAudio,
	})
	RuleDVBAudioSimilar = report.Register(report.Rule{
		ID: "MPD-DVB-AUDIO-SIMILAR", Severity: report.Warning,
		Desc:    "DVB audio Representations in an AdaptationSet should share @codecs, @audioSamplingRate and AudioChannelConfiguration",
		SpecRef: refDVBAudio,
	})
	RuleDVBAuxiliary = report.Register(report.Rule{
		ID: "MPD-DVB-AUXILIARY", Severity: report.Error,
		Desc:    "DVB auxiliary audio AdaptationSets must not have @lang, Role, Accessibility or AudioChannelConfiguration",
		SpecRef: "ETSI TS 103 285 6.8",
	})
	RuleDVBPreselection = report.Register(report.Rule{
		ID: "MPD-DVB-PRESELECTION", Severity: report.Error,
		Desc:    "DVB audio Preselections need a role, a main one per bundle, and components that are AdaptationSet ids",
		SpecRef: "ETSI TS 103 285 6.8",
	})
	RuleDVBAC4 = report.Register(report.Rule{
		ID: "MPD-DVB-AC4", Severity: report.Error,
		Desc:    "DVB AC-4 needs presentation_version 1, md_compat at most 3, a 48, 96 or 192 kHz sampling rate, and audio frame rates matching video",
		SpecRef: "ETSI TS 103 285 6.3",
	})
	RuleDVBAC4Version = report.Register(report.Rule{
		ID: "MPD-DVB-AC4-VERSION", Severity: report.Warning,
		Desc: "DVB AC-4 should use bitstream_version 2", SpecRef: "ETSI TS 103 285 6.3",
	})
	RuleAC4Codecs = report.Register(report.Rule{
		ID: "MPD-AC4-CODECS", Severity: report.Error,
		Desc: "AC-4 @codecs must be ac-4 followed by three two-digit hexadecimal fields", SpecRef: refAC4Part1,
	})
	RuleAC4Attrs = report.Register(report.Rule{
		ID: "MPD-AC4-ATTRS", Severity: report.Error,
		Desc:    "AC-4 AdaptationSets need the right @mimeType, @audioSamplingRate and @startWithSAP",
		SpecRef: refAC4Part2,
	})
	RuleAC4ChannelConfig = report.Register(report.Rule{
		ID: "MPD-AC4-CHANNELCONFIG", Severity: report.Error,
		Desc:    "AC-4 AudioChannelConfiguration must use a valid scheme and value, and the MPEG scheme when it can",
		SpecRef: refAC4Part2,
	})
	RuleAC4ChannelConfigCount = report.Register(report.Rule{
		ID: "MPD-AC4-CHANNELCONFIG-COUNT", Severity: report.Warning,
		Desc:    "AC-4 Representations should have exactly one AudioChannelConfiguration and use specified schemes",
		SpecRef: refAC4Part2,
	})
	RuleAC4Role = report.Register(report.Rule{
		ID: "MPD-AC4-ROLE", Severity: report.Error,
		Desc: "AC-4 (bitstream version 0) Role values must be main, alternate, commentary or dub", SpecRef: refAC4Part1,
	})
	RuleAC4Virtualized = report.Register(report.Rule{
		ID: "MPD-AC4-VIRTUALIZED", Severity: report.Error,
		Desc: "The AC-4 immersive audio for headphones descriptor must have @value 1", SpecRef: "ETSI TS 103 190-2 G.2.12",
	})
)

var (
	audioCodecRe = regexp.MustCompile(audioCodecPattern)
	videoCodecRe = regexp.MustCompile(videoCodecPattern)
	ac4ItemRe    = regexp.MustCompile(`(?i)` + ac4CodecItemPattern)
	hex4AnyRe    = regexp.MustCompile(`^[0-9a-fA-F]{4}$`)
	hex6AnyRe    = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
)

// dolbyToMPEG maps Dolby channel configurations to MPEG ChannelConfiguration values.
var dolbyToMPEG = map[string]int{
	"000002": 1, "000001": 2, "000003": 3, "008003": 4, "000007": 5, "000047": 6, "020047": 7,
	"008001": 9, "000005": 10, "008047": 11, "00004F": 12, "02FF7F": 13, "06FF6F": 13,
	"000057": 14, "040047": 14, "00145F": 15, "04144F": 15, "000077": 16, "040067": 16,
	"000A77": 17, "040A67": 17, "000A7F": 18, "040A6F": 18, "00007F": 19, "04006F": 19,
	"01007F": 20, "05006F": 20,
}

// validMPEGACC are the MPEG ChannelConfiguration values allowed for AC-4.
var validMPEGACC = []int{1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 14, 16, 17, 19}

func isAuxiliary(as *m.AdaptationSetType) bool {
	return slices.ContainsFunc(as.EssentialProperties, func(d *m.DescriptorType) bool { return d.SchemeIdUri == schemePreselection })
}

// ac4Fields returns bitstream_version, presentation_version and md_compat from an ac-4 codecs string.
func ac4Fields(codecs string) (bsVer, presVer, mdCompat string, ok bool) {
	parts := strings.Split(strings.TrimSpace(codecs), ".")
	if len(parts) != ac4PresentationFields || !strings.EqualFold(parts[0], "ac-4") {
		return "", "", "", false
	}
	return parts[1], parts[2], parts[3], true
}

func dolbyRules(md *m.MPD) []report.Finding {
	var f []report.Finding
	mpdP := profiles(splitProfiles(string(md.Profiles)))
	for pi, p := range md.Periods {
		pPath := fmt.Sprintf("Period[%d]", pi)
		for ai, as := range p.AdaptationSets {
			asPath := fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)
			asP := effective(as.Profiles, mpdP)
			f = append(f, ac4AdaptationSetRules(as, asPath)...)
			if asP.has(profileDVB2017) && audioCodecRe.MatchString(as.Codecs+repCodecs(as)) {
				f = append(f, dvbAudioRules(p, as, asPath)...)
			}
			for _, rep := range as.Representations {
				f = append(f, ac4RepresentationRules(as, rep, fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id))...)
			}
		}
		if mpdP.has(profileDVB2017) {
			f = append(f, dvbPreselectionRules(p, pPath)...)
		}
	}
	return f
}

func repCodecs(as *m.AdaptationSetType) string {
	var b strings.Builder
	for _, r := range as.Representations {
		b.WriteString(r.Codecs)
		b.WriteByte(' ')
	}
	return b.String()
}

// dvbAudioRules ports dvb-dash.sch asserts 1 to 5 and 9 to 13.
func dvbAudioRules(p *m.Period, as *m.AdaptationSetType, asPath string) []report.Finding {
	var f []report.Finding
	loc := report.Location{Path: asPath}
	if isAuxiliary(as) {
		// SCH-DVB-5
		if as.Lang != "" || len(as.Roles) > 0 || len(as.Accessibilities) > 0 || len(as.AudioChannelConfigurations) > 0 {
			f = append(f, RuleDVBAuxiliary.New(loc, "auxiliary audio AdaptationSet with @lang, Role, Accessibility or AudioChannelConfiguration"))
		}
	} else {
		// SCH-DVB-1 to SCH-DVB-4
		values := func(get func(r *m.RepresentationType) string) map[string]bool {
			set := map[string]bool{}
			for _, r := range as.Representations {
				set[get(r)] = true
			}
			return set
		}
		if len(values(func(r *m.RepresentationType) string { return first(r.MimeType, as.MimeType) })) > 1 {
			f = append(f, RuleDVBAudioCommon.New(loc, "audio Representations have different @mimeType"))
		}
		if len(values(func(r *m.RepresentationType) string { return first(r.Codecs, as.Codecs) })) > 1 {
			f = append(f, RuleDVBAudioSimilar.New(loc, "audio Representations have different @codecs"))
		}
		if len(values(func(r *m.RepresentationType) string { return vecString(r.AudioSamplingRate, as.AudioSamplingRate) })) > 1 {
			f = append(f, RuleDVBAudioSimilar.New(loc, "audio Representations have different @audioSamplingRate"))
		}
		if len(values(func(r *m.RepresentationType) string {
			return channelConfigs(r.AudioChannelConfigurations, as.AudioChannelConfigurations)
		})) > 1 {
			f = append(f, RuleDVBAudioSimilar.New(loc, "audio Representations have different AudioChannelConfiguration"))
		}
	}
	// SCH-DVB-9 to SCH-DVB-13
	f = append(f, dvbAC4Check(p, as.Codecs, as.AudioSamplingRate, &as.RepresentationBaseType, loc)...)
	for _, r := range as.Representations {
		if r.Codecs != "" {
			f = append(f, dvbAC4Check(p, r.Codecs, r.AudioSamplingRate, &r.RepresentationBaseType,
				report.Location{Path: fmt.Sprintf("%s/Representation[%s]", asPath, r.Id), RepID: r.Id})...)
		}
	}
	return f
}

// dvbAC4Check ports dvb-dash.sch asserts 9 to 13 for one element.
func dvbAC4Check(p *m.Period, codecs string, sampling *m.UIntVectorType, rb *m.RepresentationBaseType, loc report.Location) []report.Finding {
	bsVer, presVer, mdCompat, ok := ac4Fields(codecs)
	if !ok {
		return nil
	}
	var f []report.Finding
	if bsVer != "02" {
		f = append(f, RuleDVBAC4Version.New(loc, "AC-4 bitstream_version %s, not 02", bsVer))
	}
	if presVer != "01" {
		f = append(f, RuleDVBAC4.New(loc, "AC-4 presentation_version %s, not 01", presVer))
	}
	if !slices.Contains([]string{"00", "01", "02", "03"}, mdCompat) {
		f = append(f, RuleDVBAC4.New(loc, "AC-4 md_compat %s is larger than 3", mdCompat))
	}
	if sampling != nil && !slices.Contains([]string{"48000", "96000", "192000"}, string(*sampling)) {
		f = append(f, RuleDVBAC4.New(loc, "AC-4 @audioSamplingRate %s is not 48000, 96000 or 192000", *sampling))
	}
	for _, d := range append(slices.Clone(rb.SupplementalProperties), rb.EssentialProperties...) {
		if d.SchemeIdUri != schemeAudioFrameRate {
			continue
		}
		if rate, err := strconv.ParseFloat(d.Value, 64); err == nil && rate > 60 && videoUpTo60(p) {
			f = append(f, RuleDVBAC4.New(loc, "audio frame rate %s Hz above 60 while some video has a frame rate of at most 60", d.Value))
		}
	}
	return f
}

func vecString(vals ...*m.UIntVectorType) string {
	for _, v := range vals {
		if v != nil {
			return string(*v)
		}
	}
	return ""
}

func channelConfigs(levels ...[]*m.DescriptorType) string {
	for _, l := range levels {
		if len(l) > 0 {
			var parts []string
			for _, d := range l {
				parts = append(parts, string(d.SchemeIdUri)+"="+d.Value)
			}
			return strings.Join(parts, ";")
		}
	}
	return ""
}

func videoUpTo60(p *m.Period) bool {
	for _, as := range p.AdaptationSets {
		if !videoCodecRe.MatchString(as.Codecs + repCodecs(as)) {
			continue
		}
		for _, fr := range []string{string(as.FrameRate), as.MaxFrameRate} {
			if v, ok := parseFrameRate(fr); ok && v <= 60 {
				return true
			}
		}
	}
	return false
}

// dvbPreselectionRules ports dvb-dash.sch asserts 6 to 8.
func dvbPreselectionRules(p *m.Period, pPath string) []report.Finding {
	var f []report.Finding
	asIDs := map[string]*m.AdaptationSetType{}
	for _, as := range p.AdaptationSets {
		if as.Id != nil {
			asIDs[strconv.FormatUint(uint64(*as.Id), 10)] = as
		}
	}
	bundles := map[string][]*m.PreselectionType{}
	for i, ps := range p.Preselections {
		if !audioCodecRe.MatchString(ps.Codecs) {
			continue
		}
		loc := report.Location{Path: fmt.Sprintf("%s/Preselection[%d]", pPath, i)}
		f = append(f, dvbAC4Check(p, ps.Codecs, ps.AudioSamplingRate, &ps.RepresentationBaseType, loc)...)
		hasRole := slices.ContainsFunc(ps.Roles, func(d *m.DescriptorType) bool { return d.SchemeIdUri == "urn:mpeg:dash:role:2011" })
		if !hasRole {
			f = append(f, RuleDVBPreselection.New(loc, "audio Preselection without a urn:mpeg:dash:role:2011 Role"))
		}
		if ps.PreselectionComponents == nil {
			continue
		}
		for _, c := range strings.Fields(string(*ps.PreselectionComponents)) {
			as, ok := asIDs[c]
			if !ok {
				f = append(f, RuleDVBPreselection.New(loc, "@preselectionComponents %q is not an AdaptationSet id", c))
				continue
			}
			if !isAuxiliary(as) {
				bundles[c] = append(bundles[c], ps)
			}
		}
	}
	for bundle, pss := range bundles {
		if len(pss) < 2 {
			continue
		}
		hasMain := slices.ContainsFunc(pss, func(ps *m.PreselectionType) bool {
			return slices.ContainsFunc(ps.Roles, func(d *m.DescriptorType) bool {
				return d.SchemeIdUri == "urn:mpeg:dash:role:2011" && d.Value == "main"
			})
		})
		if !hasMain {
			f = append(f, RuleDVBPreselection.New(report.Location{Path: pPath},
				"%d Preselections for audio bundle %s, but none has Role main", len(pss), bundle))
		}
	}
	return f
}

// ac4AdaptationSetRules ports ac4-generic.sch asserts 1 to 5.
func ac4AdaptationSetRules(as *m.AdaptationSetType, asPath string) []report.Finding {
	codecs := as.Codecs
	if codecs == "" && len(as.Representations) > 0 {
		codecs = as.Representations[0].Codecs
	}
	if !strings.Contains(strings.ToLower(codecs), "ac-4") {
		return nil
	}
	var f []report.Finding
	loc := report.Location{Path: asPath}
	for item := range strings.FieldsSeq(strings.ReplaceAll(codecs, ",", " ")) {
		if strings.HasPrefix(strings.ToLower(item), "ac-4") && !ac4ItemRe.MatchString(item) {
			f = append(f, RuleAC4Codecs.New(loc, "@codecs %q is not a valid AC-4 codecs string", item))
		}
	}
	bsVer, _, _, ok := ac4Fields(codecs)
	if !ok {
		return f
	}
	mime := first(as.MimeType, firstRepMime(as))
	switch bsVer {
	case "00":
		if mime != "audio/mp4" && mime != "video/mp4" {
			f = append(f, RuleAC4Attrs.New(loc, "AC-4 @mimeType %q is not audio/mp4 or video/mp4", mime))
		}
	case "01", "02":
		if as.AudioSamplingRate == nil && !allReps(as, func(r *m.RepresentationType) bool { return r.AudioSamplingRate != nil }) {
			f = append(f, RuleAC4Attrs.New(loc, "AC-4 without @audioSamplingRate"))
		}
		if mime != "audio/mp4" {
			f = append(f, RuleAC4Attrs.New(loc, "AC-4 @mimeType %q is not audio/mp4", mime))
		}
		if as.StartWithSAP != 1 && !allReps(as, func(r *m.RepresentationType) bool { return r.StartWithSAP == 1 }) {
			f = append(f, RuleAC4Attrs.New(loc, "AC-4 without @startWithSAP 1"))
		}
	}
	return f
}

func firstRepMime(as *m.AdaptationSetType) string {
	if len(as.Representations) > 0 {
		return as.Representations[0].MimeType
	}
	return ""
}

func allReps(as *m.AdaptationSetType, pred func(r *m.RepresentationType) bool) bool {
	return len(as.Representations) > 0 && !slices.ContainsFunc(as.Representations, func(r *m.RepresentationType) bool { return !pred(r) })
}

// ac4RepresentationRules ports ac4-generic.sch asserts 6 to 17.
func ac4RepresentationRules(as *m.AdaptationSetType, rep *m.RepresentationType, rPath string) []report.Finding {
	codecs := first(rep.Codecs, as.Codecs)
	bsVer, _, _, ok := ac4Fields(codecs)
	if !ok {
		return nil
	}
	var f []report.Finding
	loc := report.Location{Path: rPath, RepID: rep.Id}
	accs := rep.AudioChannelConfigurations
	if len(accs) == 0 {
		accs = as.AudioChannelConfigurations
	}
	switch bsVer {
	case "00":
		// SCH-AC4-6
		if len(accs) == 0 {
			f = append(f, RuleAC4ChannelConfig.New(loc, "AC-4 Representation without AudioChannelConfiguration"))
		}
		// SCH-AC4-7
		for _, r := range as.Roles {
			if r.SchemeIdUri == "urn:mpeg:dash:role:2011" && !slices.Contains([]string{"main", "alternate", "commentary", "dub"}, r.Value) {
				f = append(f, RuleAC4Role.New(loc, "Role %q is not main, alternate, commentary or dub", r.Value))
			}
		}
		// SCH-AC4-9 to SCH-AC4-11
		n := 0
		for _, d := range accs {
			switch d.SchemeIdUri {
			case schemeDolbyACC2014:
				n++
				if !hex4AnyRe.MatchString(d.Value) {
					f = append(f, RuleAC4ChannelConfig.New(loc, "Dolby 2014 channel configuration %q is not 4 hex digits", d.Value))
				} else if x, ok := dolbyToMPEG[strings.ToUpper("00"+d.Value)]; ok {
					f = append(f, RuleAC4ChannelConfig.New(loc, "channel configuration %s maps to MPEG ChannelConfiguration %d; use %s", d.Value, x, schemeMPEGACC))
				}
			case schemeMPEGACC:
				n++
			}
		}
		if n > 1 {
			f = append(f, RuleAC4ChannelConfigCount.New(loc, "%d AudioChannelConfiguration elements, recommended is one", n))
		}
	case "01", "02":
		// SCH-AC4-8
		for _, d := range append(slices.Clone(rep.SupplementalProperties), as.SupplementalProperties...) {
			if d.SchemeIdUri == schemeVirtualized && d.Value != "1" {
				f = append(f, RuleAC4Virtualized.New(loc, "virtualized_content @value %q, not 1", d.Value))
			}
		}
		// SCH-AC4-12 to SCH-AC4-17
		n := 0
		for _, d := range accs {
			switch d.SchemeIdUri {
			case schemeDolbyACC2015:
				n++
				if !hex6AnyRe.MatchString(d.Value) {
					f = append(f, RuleAC4ChannelConfig.New(loc, "Dolby 2015 channel configuration %q is not 6 hex digits", d.Value))
				} else if x, ok := dolbyToMPEG[strings.ToUpper(d.Value)]; ok {
					f = append(f, RuleAC4ChannelConfig.New(loc, "channel configuration %s maps to MPEG ChannelConfiguration %d; use %s", d.Value, x, schemeMPEGACC))
				}
			case schemeMPEGACC:
				n++
				if v, err := strconv.Atoi(d.Value); err != nil || !slices.Contains(validMPEGACC, v) {
					f = append(f, RuleAC4ChannelConfig.New(loc, "MPEG ChannelConfiguration %q is not 1-7, 9-12, 14, 16, 17 or 19", d.Value))
				}
			default:
				f = append(f, RuleAC4ChannelConfigCount.New(loc, "unspecified AudioChannelConfiguration scheme %q for AC-4", d.SchemeIdUri))
			}
		}
		if n > 1 {
			f = append(f, RuleAC4ChannelConfigCount.New(loc, "%d AudioChannelConfiguration elements, recommended is one", n))
		}
	}
	return f
}
