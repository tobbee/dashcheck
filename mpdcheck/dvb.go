package mpdcheck

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// MPD rules ported from the DVB module of the DASH-IF Conformance tool
// (ETSI TS 103 285, DVB-DASH). They apply to MPDs with a DVB-DASH profile.

const (
	profileDVBLive     = "urn:dvb:dash:profile:dvb-dash:isoff-ext-live:2014"
	profileDVBOnDemand = "urn:dvb:dash:profile:dvb-dash:isoff-ext-on-demand:2014"
	refDVB             = "ETSI TS 103 285"
)

var (
	RuleDVBLimits = report.Register(report.Rule{
		ID: "MPD-DVB-LIMITS", Severity: report.Error,
		Desc:    "A DVB MPD must be at most 256 kB with at most 64 Periods, 16 AdaptationSets per Period and 16 Representations per AdaptationSet",
		SpecRef: refDVB + " 4.5",
	})
	RuleDVBMUP = report.Register(report.Rule{
		ID: "MPD-DVB-MUP", Severity: report.Warning,
		Desc: "DVB MPD@minimumUpdatePeriod should be at least 1 second", SpecRef: refDVB,
	})
	RuleDVBProfile = report.Register(report.Rule{
		ID: "MPD-DVB-PROFILE", Severity: report.Error,
		Desc:    "Each DVB Representation must use exactly one of the live and on-demand profiles, and on-demand means a single segment",
		SpecRef: refDVB + " 11.1",
	})
	RuleDVBPeriod = report.Register(report.Rule{
		ID: "MPD-DVB-PERIOD", Severity: report.Error,
		Desc:    "DVB Periods must be either live or on-demand, without Period.SegmentList, and without Period.SegmentTemplate for on-demand",
		SpecRef: refDVB + " 4.2.2",
	})
	RuleDVBMainRole = report.Register(report.Rule{
		ID: "MPD-DVB-MAINROLE", Severity: report.Error,
		Desc:    "With several video or audio AdaptationSets in a Period, one of them must have Role main",
		SpecRef: refDVB + " 4.2.2, 6.1.2",
	})
	RuleDVBFont = report.Register(report.Rule{
		ID: "MPD-DVB-FONT", Severity: report.Error,
		Desc: "Downloadable font descriptors are only allowed on subtitle AdaptationSets", SpecRef: refDVB + " 7.2.1.1",
	})
	RuleDVBAudio = report.Register(report.Rule{
		ID: "MPD-DVB-AUDIO", Severity: report.Error,
		Desc:    "DVB audio needs a Role, @mimeType, @codecs, @audioSamplingRate and AudioChannelConfiguration, and E-AC-3 the Dolby channel scheme",
		SpecRef: refDVB + " 6.1",
	})
	RuleDVBFallback = report.Register(report.Rule{
		ID: "MPD-DVB-FALLBACK", Severity: report.Error,
		Desc: "A fallback audio AdaptationSet must refer to an existing audio AdaptationSet", SpecRef: refDVB + " 6.6.3",
	})
	RuleDVBAudioShare = report.Register(report.Rule{
		ID: "MPD-DVB-AUDIO-SHARE", Severity: report.Warning,
		Desc: "Audio should be at most 20% of the total bandwidth", SpecRef: refDVB + " 11.3.0",
	})
	RuleDVBCodec = report.Register(report.Rule{
		ID: "MPD-DVB-CODEC", Severity: report.Warning,
		Desc: "Codecs should be ones that DVB-DASH supports", SpecRef: refDVB + " 5, 6",
	})
	RuleDVBContentProtection = report.Register(report.Rule{
		ID: "MPD-DVB-CP", Severity: report.Error,
		Desc: "DVB ContentProtection descriptors must be on the AdaptationSet", SpecRef: refDVB + " 8.3",
	})
	RuleDVBEvent = report.Register(report.Rule{
		ID: "MPD-DVB-EVENT", Severity: report.Error,
		Desc: "DVB TV-Anytime events need @presentationTime and valid XML data", SpecRef: refDVB + " 9.1.2.1",
	})
	RuleDVBReporting = report.Register(report.Rule{
		ID: "MPD-DVB-REPORTING", Severity: report.Error,
		Desc:    "DVB metric reporting needs an absolute HTTP(S) reporting URL and a probability between 1 and 1000",
		SpecRef: refDVB + " 10.12.3",
	})
	RuleDVBResolution = report.Register(report.Rule{
		ID: "MPD-DVB-RESOLUTION", Severity: report.Warning,
		Desc: "Video resolutions should be ones listed by DVB-DASH", SpecRef: refDVB + " 10.3",
	})
	RuleDVBSCTE = report.Register(report.Rule{
		ID: "MPD-DVB-SCTE", Severity: report.Error,
		Desc:    "SCTE-35 event streams need a frame-rate multiple @timescale, and events a @presentationTime and a finite @duration without @messageData or @contentEncoding",
		SpecRef: refDVB + " 9.1.9",
	})
	RuleDVBSCTEValue = report.Register(report.Rule{
		ID: "MPD-DVB-SCTE-VALUE", Severity: report.Warning,
		Desc: "SCTE-35 event streams should not have @value", SpecRef: refDVB + " 9.1.9",
	})
	RuleDVBSubtitle = report.Register(report.Rule{
		ID: "MPD-DVB-SUBTITLE", Severity: report.Error,
		Desc: "DVB subtitle @codecs must start with stpp", SpecRef: refDVB + " 7.1.1",
	})
	RuleDVBTLSBitrate = report.Register(report.Rule{
		ID: "MPD-DVB-TLS-BITRATE", Severity: report.Info,
		Desc:    "Over TLS, terminals without UHD support handle 12 Mbit/s, UHD terminals 39 Mbit/s and UHD HFR terminals 51 Mbit/s",
		SpecRef: refDVB + " 10.2",
	})
	RuleDVBUTCTiming = report.Register(report.Rule{
		ID: "MPD-DVB-UTCTIMING", Severity: report.Warning,
		Desc:    "A DVB MPD with an availabilityStartTime should have a UTCTiming with an NTP or HTTP scheme",
		SpecRef: refDVB + " 4.7.2",
	})
	RuleDVBVideo = report.Register(report.Rule{
		ID: "MPD-DVB-VIDEO", Severity: report.Error,
		Desc:    "DVB video AdaptationSets need size, frame rate and @par, and @scanType when interlaced video is used",
		SpecRef: refDVB + " 4.4",
	})
	RuleDVBFrameRates = report.Register(report.Rule{
		ID: "MPD-DVB-FRAMERATES", Severity: report.Warning,
		Desc: "Frame rates in an AdaptationSet should be integer multiples of each other", SpecRef: refDVB + " 11.2.2",
	})
)

var dvbCodecPrefixes = []string{"avc", "hev1", "hvc1", "mp4a", "ec-3", "ac-4", "dtsc", "dtsh", "dtse", "dtsl", "stpp"}

var dvbProgressive = []string{"3840x2160", "3200x1800", "2560x1440", "1920x1080", "1600x900", "1280x720", "1024x576",
	"960x540", "852x480", "768x432", "720x404", "704x396", "640x360", "512x288", "480x270", "384x216", "320x180", "192x108"}

var dvbInterlaced = []string{"1920x1080", "704x576", "533x576", "352x288"}

var dvbUTCSchemes = []string{"urn:mpeg:dash:utc:ntp:2014", "urn:mpeg:dash:utc:http-head:2014",
	"urn:mpeg:dash:utc:http-xsdate:2014", "urn:mpeg:dash:utc:http-iso:2014", "urn:mpeg:dash:utc:http-ntp:2014"}

func isDVB(p profiles) bool { return p.has(profileDVB2014) || p.has(profileDVB2017) }

func dvbRules(md *m.MPD, raw *rawElem, size int, mpdURL string) []report.Finding {
	mpdP := profiles(splitProfiles(string(md.Profiles)))
	if !isDVB(mpdP) {
		return nil
	}
	var f []report.Finding
	mpdLoc := report.Location{Path: "MPD"}
	// DVB-14 to DVB-17
	if size > 256*1024 {
		f = append(f, RuleDVBLimits.New(mpdLoc, "MPD is %d bytes", size))
	}
	if len(md.Periods) > 64 {
		f = append(f, RuleDVBLimits.New(mpdLoc, "%d Periods", len(md.Periods)))
	}
	// DVB-1
	if md.MinimumUpdatePeriod != nil && time.Duration(*md.MinimumUpdatePeriod) < time.Second {
		f = append(f, RuleDVBMUP.New(mpdLoc, "@minimumUpdatePeriod %v", time.Duration(*md.MinimumUpdatePeriod)))
	}
	// DVB-42
	if md.GetType() == "dynamic" || md.AvailabilityStartTime != "" {
		if !slices.ContainsFunc(md.UTCTimings, func(u *m.DescriptorType) bool { return slices.Contains(dvbUTCSchemes, string(u.SchemeIdUri)) }) {
			f = append(f, RuleDVBUTCTiming.New(mpdLoc, "no UTCTiming with an NTP or HTTP scheme"))
		}
	}
	reps, _ := addressing.Reps(md, mpdURL)
	byAS := map[string][]*addressing.Rep{}
	for _, r := range reps {
		byAS[r.ASPath] = append(byAS[r.ASPath], r)
	}
	for pi, p := range md.Periods {
		pPath := fmt.Sprintf("Period[%d]", pi)
		pLoc := report.Location{Path: pPath}
		if len(p.AdaptationSets) > 16 {
			f = append(f, RuleDVBLimits.New(pLoc, "%d AdaptationSets", len(p.AdaptationSets)))
		}
		// DVB-22
		if p.SegmentList != nil {
			f = append(f, RuleDVBPeriod.New(pLoc, "SegmentList on the Period"))
		}
		f = append(f, dvbPeriodProfiles(p, byAS, pPath)...)
		f = append(f, dvbMainRoles(p, pLoc)...)
		f = append(f, dvbBandwidth(p, pLoc)...)
		if strings.HasPrefix(mpdURL, "https://") {
			f = append(f, dvbTLSBitrate(p, pLoc)...)
		}
		audioIDs := map[string]bool{}
		for _, as := range p.AdaptationSets {
			if as.ContentType == "audio" && as.Id != nil {
				audioIDs[strconv.FormatUint(uint64(*as.Id), 10)] = true
			}
		}
		for ai, as := range p.AdaptationSets {
			asPath := fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)
			loc := report.Location{Path: asPath}
			if len(as.Representations) > 16 {
				f = append(f, RuleDVBLimits.New(loc, "%d Representations", len(as.Representations)))
			}
			ct := string(as.ContentType)
			// DVB-2, DVB-56
			if ct == "video" || ct == "audio" {
				for _, d := range append(slices.Clone(as.SupplementalProperties), as.EssentialProperties...) {
					if d.SchemeIdUri == "urn:dvb:dash:fontdownload:2014" && d.Value == "1" {
						f = append(f, RuleDVBFont.New(loc, "downloadable font descriptor on a %s AdaptationSet", ct))
					}
				}
			}
			switch ct {
			case "audio":
				f = append(f, dvbAudio(as, audioIDs, asPath)...)
			case "video":
				f = append(f, dvbVideo(as, asPath)...)
			case "text":
				// DVB-38
				if !strings.HasPrefix(first(as.Codecs, firstRepCodecs(as)), "stpp") {
					f = append(f, RuleDVBSubtitle.New(loc, "@codecs %q", first(as.Codecs, firstRepCodecs(as))))
				}
			}
			for _, rep := range as.Representations {
				rLoc := report.Location{Path: fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id), RepID: rep.Id}
				// DVB-13
				if len(rep.ContentProtections) > 0 {
					f = append(f, RuleDVBContentProtection.New(rLoc, "ContentProtection on the Representation"))
				}
				// DVB-12
				for c := range strings.SplitSeq(first(rep.Codecs, as.Codecs), ",") {
					c = strings.TrimSpace(c)
					if c != "" && !slices.ContainsFunc(dvbCodecPrefixes, func(p string) bool { return strings.HasPrefix(c, p) }) {
						f = append(f, RuleDVBCodec.New(rLoc, "codec %q", c))
					}
				}
				// DVB-30
				if ct == "video" {
					f = append(f, dvbResolution(as, rep, rLoc)...)
				}
			}
		}
	}
	f = append(f, dvbRawRules(raw, md)...)
	return f
}

func firstRepCodecs(as *m.AdaptationSetType) string {
	if len(as.Representations) > 0 {
		return as.Representations[0].Codecs
	}
	return ""
}

// dvbPeriodProfiles ports DVB-23, DVB-24 and DVB-27 to DVB-29.
func dvbPeriodProfiles(p *m.Period, byAS map[string][]*addressing.Rep, pPath string) []report.Finding {
	var f []report.Finding
	nLive, nOnDemand := 0, 0
	for ai := range p.AdaptationSets {
		for _, r := range byAS[fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)] {
			live, onDemand := r.HasProfile(profileDVBLive), r.HasProfile(profileDVBOnDemand)
			if !r.HasProfile(profileDVB2014) && !r.HasProfile(profileDVB2017) {
				f = append(f, RuleDVBProfile.New(r.Loc(), "no DVB-DASH profile in the effective @profiles"))
			}
			switch {
			case live && onDemand:
				f = append(f, RuleDVBProfile.New(r.Loc(), "both the DVB live and on-demand profiles"))
			case !live && !onDemand:
				f = append(f, RuleDVBProfile.New(r.Loc(), "neither the DVB live nor the on-demand profile"))
			case onDemand:
				nOnDemand++
				if r.Kind != addressing.KindSegmentBase {
					f = append(f, RuleDVBProfile.New(r.Loc(), "DVB on-demand profile with %s addressing instead of a single segment", r.Kind))
				}
			default:
				nLive++
			}
		}
	}
	if nLive > 0 && nOnDemand > 0 {
		f = append(f, RuleDVBPeriod.New(report.Location{Path: pPath}, "both live and on-demand Representations"))
	}
	if nOnDemand > 0 && p.SegmentTemplate != nil {
		f = append(f, RuleDVBPeriod.New(report.Location{Path: pPath}, "SegmentTemplate on an on-demand Period"))
	}
	return f
}

// dvbMainRoles ports DVB-25 and DVB-26.
func dvbMainRoles(p *m.Period, loc report.Location) []report.Finding {
	var f []report.Finding
	for _, ct := range []string{"video", "audio"} {
		n, main := 0, false
		for _, as := range p.AdaptationSets {
			if string(as.ContentType) != ct {
				continue
			}
			n++
			if slices.ContainsFunc(as.Roles, func(r *m.DescriptorType) bool {
				return r.SchemeIdUri == "urn:mpeg:dash:role:2011" && r.Value == "main"
			}) {
				main = true
			}
		}
		if n > 1 && !main {
			f = append(f, RuleDVBMainRole.New(loc, "%d %s AdaptationSets but none has Role main", n, ct))
		}
	}
	return f
}

// dvbAudio ports DVB-3 to DVB-5 and DVB-10. DVB-6 to DVB-9 share the rules of dvb-dash.sch.
func dvbAudio(as *m.AdaptationSetType, audioIDs map[string]bool, asPath string) []report.Finding {
	var f []report.Finding
	loc := report.Location{Path: asPath}
	if !slices.ContainsFunc(as.Roles, func(r *m.DescriptorType) bool { return r.SchemeIdUri == "urn:mpeg:dash:role:2011" }) {
		f = append(f, RuleDVBAudio.New(loc, "audio AdaptationSet without urn:mpeg:dash:role:2011 Role"))
	}
	for _, rep := range as.Representations {
		rLoc := report.Location{Path: fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id), RepID: rep.Id}
		var missing []string
		if first(rep.MimeType, as.MimeType) == "" {
			missing = append(missing, "@mimeType")
		}
		if first(rep.Codecs, as.Codecs) == "" {
			missing = append(missing, "@codecs")
		}
		if rep.AudioSamplingRate == nil && as.AudioSamplingRate == nil {
			missing = append(missing, "@audioSamplingRate")
		}
		if len(rep.AudioChannelConfigurations) == 0 && len(as.AudioChannelConfigurations) == 0 {
			missing = append(missing, "AudioChannelConfiguration")
		}
		if len(missing) > 0 {
			f = append(f, RuleDVBAudio.New(rLoc, "audio Representation without %s", strings.Join(missing, ", ")))
		}
		// DVB-3 (upstream also requires the 2014 scheme for AC-4, which uses newer schemes)
		if strings.HasPrefix(first(rep.Codecs, as.Codecs), "ec-3") {
			accs := rep.AudioChannelConfigurations
			if len(accs) == 0 {
				accs = as.AudioChannelConfigurations
			}
			for _, a := range accs {
				if a.SchemeIdUri != schemeDolbyACC2014 || !hex4AnyRe.MatchString(a.Value) {
					f = append(f, RuleDVBAudio.New(rLoc, "E-AC-3 AudioChannelConfiguration %s=%q, not the Dolby 2014 scheme with 4 hex digits", a.SchemeIdUri, a.Value))
				}
			}
		}
	}
	// DVB-10
	for _, sp := range as.SupplementalProperties {
		if sp.SchemeIdUri == "urn:dvb:dash:fallback_adaptation_set:2014" && !audioIDs[sp.Value] {
			f = append(f, RuleDVBFallback.New(loc, "fallback to AdaptationSet %q, which is not an audio AdaptationSet", sp.Value))
		}
	}
	return f
}

// dvbVideo ports DVB-57 to DVB-59.
func dvbVideo(as *m.AdaptationSetType, asPath string) []report.Finding {
	var f []report.Finding
	loc := report.Location{Path: asPath}
	allReps := func(has func(r *m.RepresentationType) bool) bool {
		for _, r := range as.Representations {
			if !has(r) {
				return false
			}
		}
		return true
	}
	var missing []string
	if as.MaxWidth == 0 && as.Width == 0 {
		missing = append(missing, "@maxWidth or @width")
	}
	if as.Width == 0 && !allReps(func(r *m.RepresentationType) bool { return r.Width != 0 }) {
		missing = append(missing, "@width on every Representation")
	}
	if as.MaxHeight == 0 && as.Height == 0 {
		missing = append(missing, "@maxHeight or @height")
	}
	if as.Height == 0 && !allReps(func(r *m.RepresentationType) bool { return r.Height != 0 }) {
		missing = append(missing, "@height on every Representation")
	}
	if as.MaxFrameRate == "" && as.FrameRate == "" {
		missing = append(missing, "@maxFrameRate or @frameRate")
	}
	if as.FrameRate == "" && !allReps(func(r *m.RepresentationType) bool { return r.FrameRate != "" }) {
		missing = append(missing, "@frameRate on every Representation")
	}
	if as.Par == "" {
		missing = append(missing, "@par")
	}
	if len(missing) > 0 {
		f = append(f, RuleDVBVideo.New(loc, "video AdaptationSet without %s", strings.Join(missing, ", ")))
	}
	interlaced := as.ScanType == "interlaced" || slices.ContainsFunc(as.Representations, func(r *m.RepresentationType) bool { return r.ScanType == "interlaced" })
	if interlaced && as.ScanType == "" && !allReps(func(r *m.RepresentationType) bool { return r.ScanType != "" }) {
		f = append(f, RuleDVBVideo.New(loc, "interlaced video but @scanType is not present everywhere"))
	}
	var rates []float64
	for _, r := range as.Representations {
		if fr, ok := parseFrameRate(first(string(r.FrameRate), string(as.FrameRate))); ok && fr > 0 {
			rates = append(rates, fr)
		}
	}
	slices.Sort(rates)
	for _, r := range rates[min(1, len(rates)):] {
		if q := r / rates[0]; math.Abs(q-math.Round(q)) > 1e-6 {
			f = append(f, RuleDVBFrameRates.New(loc, "frame rates %v and %v are not integer multiples", rates[0], r))
			break
		}
	}
	return f
}

func dvbResolution(as *m.AdaptationSetType, rep *m.RepresentationType, loc report.Location) []report.Finding {
	w, h := firstU32(rep.Width, as.Width), firstU32(rep.Height, as.Height)
	if w == 0 || h == 0 {
		return nil
	}
	res := fmt.Sprintf("%dx%d", w, h)
	table := dvbProgressive
	if first(string(rep.ScanType), string(as.ScanType)) == "interlaced" {
		table = dvbInterlaced
	}
	if !slices.Contains(table, res) {
		return []report.Finding{RuleDVBResolution.New(loc, "resolution %s is not in the table", res)}
	}
	return nil
}

func firstU32(vals ...uint32) uint32 {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}

// bandwidths returns the Representation bandwidths of a Period per content type.
func bandwidths(p *m.Period) map[string][]uint32 {
	bw := map[string][]uint32{}
	for _, as := range p.AdaptationSets {
		ct := string(as.ContentType)
		for _, r := range as.Representations {
			bw[ct] = append(bw[ct], r.Bandwidth)
		}
	}
	return bw
}

// dvbBandwidth ports DVB-11. The audio share is largest with the lowest video
// and subtitle bandwidths and the highest audio bandwidth.
func dvbBandwidth(p *m.Period, loc report.Location) []report.Finding {
	bw := bandwidths(p)
	if len(bw["video"]) == 0 || len(bw["audio"]) == 0 {
		return nil
	}
	video, audio := slices.Min(bw["video"]), slices.Max(bw["audio"])
	var text uint32
	if len(bw["text"]) > 0 {
		text = slices.Min(bw["text"])
	}
	total := float64(video) + float64(audio) + float64(text)
	if share := float64(audio) / total; share > 0.2 {
		return []report.Finding{RuleDVBAudioShare.New(loc, "audio %d bit/s is %.0f%% of %0.f bit/s with the lowest video bandwidth", audio, share*100, total)}
	}
	return nil
}

// dvbTLSBitrate ports DVB-39 to DVB-41 for the highest total bandwidth.
func dvbTLSBitrate(p *m.Period, loc report.Location) []report.Finding {
	bw := bandwidths(p)
	var total uint64
	for _, ct := range []string{"video", "audio", "text"} {
		if len(bw[ct]) > 0 {
			total += uint64(slices.Max(bw[ct]))
		}
	}
	for _, limit := range []struct {
		bps  uint64
		desc string
	}{{51_000_000, "UHD HFR terminals"}, {39_000_000, "UHD terminals"}, {12_000_000, "terminals without UHD"}} {
		if total > limit.bps {
			return []report.Finding{RuleDVBTLSBitrate.New(loc, "highest total bandwidth %d bit/s exceeds what %s handle over TLS (%d bit/s)", total, limit.desc, limit.bps)}
		}
	}
	return nil
}

// dvbRawRules ports the DVB checks that need attributes as written:
// events (DVB-18, DVB-19), metric reporting (DVB-20, DVB-21) and SCTE-35 (DVB-31 to DVB-37).
func dvbRawRules(raw *rawElem, md *m.MPD) []report.Finding {
	if raw == nil {
		return nil
	}
	var f []report.Finding
	periods := raw.children("Period")
	for pi, p := range periods {
		frameRate := 0.0
		if pi < len(md.Periods) {
			for _, as := range md.Periods[pi].AdaptationSets {
				if fr, ok := parseFrameRate(string(as.FrameRate)); ok {
					frameRate = fr
				}
			}
		}
		for _, es := range p.children("EventStream") {
			loc := report.Location{Line: es.Line, Path: fmt.Sprintf("Period[%d]/EventStream", pi)}
			switch {
			case es.attr("schemeIdUri") == "urn:dvb:iptv:cpm:2014" && es.attr("value") == "1":
				for _, ev := range es.children("Event") {
					evLoc := report.Location{Line: ev.Line, Path: loc.Path + "/Event"}
					if !ev.has("presentationTime") {
						f = append(f, RuleDVBEvent.New(evLoc, "TV-Anytime event without @presentationTime"))
					}
					if text := strings.TrimSpace(ev.Text); text != "" && !wellFormed("<doc>"+text+"</doc>") {
						f = append(f, RuleDVBEvent.New(evLoc, "event data is not valid XML"))
					}
				}
			case es.attr("schemeIdUri") == "urn:scte:scte35:2014:xml+bin":
				if es.has("value") {
					f = append(f, RuleDVBSCTEValue.New(loc, "SCTE-35 EventStream with @value"))
				}
				ts, err := strconv.ParseFloat(es.attr("timescale"), 64)
				switch {
				case err != nil:
					f = append(f, RuleDVBSCTE.New(loc, "SCTE-35 EventStream without @timescale"))
				case frameRate > 0 && math.Abs(ts/frameRate-math.Round(ts/frameRate)) > 1e-6:
					f = append(f, RuleDVBSCTE.New(loc, "@timescale %s is not a multiple of the frame rate %.3f", es.attr("timescale"), frameRate))
				}
				for _, ev := range es.children("Event") {
					evLoc := report.Location{Line: ev.Line, Path: loc.Path + "/Event"}
					if !ev.has("presentationTime") {
						f = append(f, RuleDVBSCTE.New(evLoc, "SCTE-35 event without @presentationTime"))
					}
					if d := ev.attr("duration"); d == "" || d == "4294967295" || strings.EqualFold(d, "0xFFFFFFFF") {
						f = append(f, RuleDVBSCTE.New(evLoc, "SCTE-35 event without finite @duration"))
					}
					if ev.has("messageData") {
						f = append(f, RuleDVBSCTE.New(evLoc, "SCTE-35 event with @messageData"))
					}
					if ev.has("contentEncoding") {
						f = append(f, RuleDVBSCTE.New(evLoc, "SCTE-35 event with @contentEncoding"))
					}
				}
			}
		}
	}
	for _, metrics := range raw.children("Metrics") {
		for _, rp := range metrics.children("Reporting") {
			if rp.attr("schemeIdUri") != "urn:dvb:dash:reporting:2014" || rp.attr("value") != "1" {
				continue
			}
			loc := report.Location{Line: rp.Line, Path: "MPD/Metrics/Reporting"}
			u := rp.attr("reportingUrl")
			if pu, err := url.Parse(u); u == "" || err != nil || !pu.IsAbs() || (pu.Scheme != "http" && pu.Scheme != "https") {
				f = append(f, RuleDVBReporting.New(loc, "@reportingUrl %q is not an absolute HTTP(S) URL", u))
			}
			if p := rp.attr("probability"); p != "" {
				if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 1000 {
					f = append(f, RuleDVBReporting.New(loc, "@probability %q is not an integer from 1 to 1000", p))
				}
			}
		}
	}
	return f
}

func wellFormed(s string) bool {
	dec := xml.NewDecoder(bytes.NewReader([]byte(s)))
	for {
		_, err := dec.Token()
		if err != nil {
			return err.Error() == "EOF"
		}
	}
}
