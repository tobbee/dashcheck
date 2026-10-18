package segcheck

import (
	"bytes"
	"math"
	"slices"
	"strings"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Checks ported from the WaveHLSInterop module of the DASH-IF Conformance
// tool, following CTA-5005-B (DASH-HLS Interoperability Specification,
// 2024 edition). MPDs do not signal this specification, so the checks run
// only when asked for (dashcheck.Options.HLSInterop).

const refHLS = "CTA-5005-B"

var (
	RuleHLSSidx = report.Register(report.Rule{
		ID: "SEG-HLS-SIDX", Severity: report.Error,
		Desc:    "CMAF track files delivered to clients must have a single sidx box between the CMAF header and the fragments",
		SpecRef: refHLS + " 4.1.2",
	})
	RuleHLSBitrate = report.Register(report.Rule{
		ID: "SEG-HLS-BITRATE", Severity: report.Warning,
		Desc:    "On demand, the average bitrate of each CMAF fragment should be within 10% of the average bitrate of the track",
		SpecRef: refHLS + " 4.1.2",
	})
	RuleHLSText = report.Register(report.Rule{
		ID: "SEG-HLS-TEXT", Severity: report.Error,
		Desc:    "Text components must be IMSC1 Text, IMSC1.1 Text or WebVTT tracks",
		SpecRef: refHLS + " 4.1.2",
	})
	RuleHLSEncryption = report.Register(report.Rule{
		ID: "SEG-HLS-ENCRYPTION", Severity: report.Error,
		Desc: "Encrypted tracks must use cbcs with constant 16-byte IVs and a 1:9 pattern for video and 10:0 for audio, " +
			"or cenc with 8-byte IVs, and address sample auxiliary information with saio",
		SpecRef: refHLS + " 4.3.2",
	})
	RuleHLSCbcsVariant = report.Register(report.Rule{
		ID: "SEG-HLS-CBCS-VARIANT", Severity: report.Error,
		Desc:    "Each cenc-encrypted component needs an equivalent cbcs-encrypted component for HLS",
		SpecRef: refHLS + " 4.3.2, 4.3.3",
	})
	RuleHLSSegmentKey = report.Register(report.Rule{
		ID: "SEG-HLS-SEGMENT-KEY", Severity: report.Error,
		Desc:    "Each CMAF segment must use a single encryption key and initialization vector",
		SpecRef: refHLS + " 4.3.2",
	})
	RuleHLSKeyRotation = report.Register(report.Rule{
		ID: "SEG-HLS-KEY-ROTATION", Severity: report.Error,
		Desc: "A segment with a key other than the default_KID must describe it in sgpd (seig) and sbgp boxes, " +
			"and carry pssh boxes if the CMAF header has them",
		SpecRef: refHLS + " 4.6.2",
	})
	RuleHLSRoles = report.Register(report.Rule{
		ID: "SEG-HLS-ROLES", Severity: report.Error,
		Desc:    "Track roles must be stored in kind boxes in the CMAF header, using the DASH Role scheme when possible",
		SpecRef: refHLS + " 4.7.2",
	})
)

// CheckHLSInteropInit checks a CMAF header against CTA-5005-B (WAVEHLS-3
// to WAVEHLS-9, WAVEHLS-16, WAVEHLS-18, WAVEHLS-19).
func CheckHLSInteropInit(rep *addressing.Rep, init *Init, loc report.Location) []report.Finding {
	var f []report.Finding
	// WAVEHLS-16 (upstream only accepts the stpp sample entry)
	switch {
	case init.SampleEntry == "wvtt":
	case init.SampleEntry == "stpp":
		if !strings.HasPrefix(rep.Codecs, "stpp.ttml.im1t") && !strings.HasPrefix(rep.Codecs, "stpp.ttml.im2t") {
			f = append(f, RuleHLSText.New(loc, "@codecs %q is not an IMSC1 or IMSC1.1 Text profile", rep.Codecs))
		}
	case init.Handler == "subt" || init.Handler == "text":
		f = append(f, RuleHLSText.New(loc, "%s text track", init.SampleEntry))
	}
	// WAVEHLS-18, WAVEHLS-19
	for _, r := range rep.Roles {
		if !slices.ContainsFunc(init.kinds, func(k *mp4.KindBox) bool { return k.SchemeURI == string(r.SchemeIdUri) && k.Value == r.Value }) {
			f = append(f, RuleHLSRoles.New(loc, "Role %s=%q has no kind box", r.SchemeIdUri, r.Value))
		}
	}
	if len(init.kinds) > 0 && !slices.ContainsFunc(init.kinds, func(k *mp4.KindBox) bool { return k.SchemeURI == "urn:mpeg:dash:role:2011" }) {
		f = append(f, RuleHLSRoles.New(loc, "no kind box uses the DASH Role scheme urn:mpeg:dash:role:2011"))
	}
	if !init.Encrypted {
		return f
	}
	// WAVEHLS-3 to WAVEHLS-9
	t := init.tenc
	switch init.Scheme {
	case "cbcs":
		if t == nil {
			break
		}
		if t.DefaultPerSampleIVSize != 0 || len(t.DefaultConstantIV) != 16 {
			f = append(f, RuleHLSEncryption.New(loc, "cbcs with %d-byte per-sample and %d-byte constant IVs, not a constant 16-byte IV",
				t.DefaultPerSampleIVSize, len(t.DefaultConstantIV)))
		}
		crypt, skip := t.DefaultCryptByteBlock, t.DefaultSkipByteBlock
		switch init.Handler {
		case "vide":
			if crypt != 1 || skip != 9 {
				f = append(f, RuleHLSEncryption.New(loc, "video encrypt:skip pattern %d:%d, not 1:9", crypt, skip))
			}
		case "soun":
			// 0:0 is the whole-block full-sample encryption of ISO/IEC 23001-7 10.4.1, which is the same.
			if !(crypt == 10 && skip == 0) && !(crypt == 0 && skip == 0) {
				f = append(f, RuleHLSEncryption.New(loc, "audio encrypt:skip pattern %d:%d, not 10:0", crypt, skip))
			}
		}
	case "cenc":
		if t != nil && t.DefaultPerSampleIVSize != 8 && t.DefaultPerSampleIVSize != 0 {
			f = append(f, RuleHLSEncryption.New(loc, "cenc with %d-byte IVs, not 8-byte (ISO/IEC 23000-19 8.2.3.1)", t.DefaultPerSampleIVSize))
		}
	default:
		f = append(f, RuleHLSEncryption.New(loc, "scheme %q, not cbcs or cenc", init.Scheme))
	}
	return f
}

// hlsMediaRules checks the encryption of a media segment (WAVEHLS-7,
// WAVEHLS-10 to WAVEHLS-13).
func hlsMediaRules(init *Init, mf *mp4.File, loc report.Location) []report.Finding {
	if !init.HLSInterop || !init.Encrypted || init.tenc == nil {
		return nil
	}
	var f []report.Finding
	type keyIV struct {
		kid string
		iv  []byte
	}
	var keys []keyIV
	for _, ms := range mf.Segments {
		for _, frag := range ms.Fragments {
			if frag.Moof == nil {
				continue
			}
			for _, traf := range frag.Moof.Trafs {
				// WAVEHLS-7
				if (traf.Senc != nil || traf.Saiz != nil) && traf.Saio == nil {
					f = append(f, RuleHLSEncryption.New(loc, "sample auxiliary information without a saio box"))
				}
				k := keyIV{init.tenc.DefaultKID.String(), init.tenc.DefaultConstantIV}
				var seig *mp4.SeigSampleGroupEntry
				if traf.Sgpd != nil {
					for _, e := range traf.Sgpd.SampleGroupEntries {
						if s, ok := e.(*mp4.SeigSampleGroupEntry); ok {
							seig = s
						}
					}
				}
				if seig != nil {
					k = keyIV{seig.KID.String(), seig.ConstantIV}
				}
				keys = append(keys, k)
				// WAVEHLS-10, WAVEHLS-11
				if k.kid == init.tenc.DefaultKID.String() {
					continue
				}
				if traf.Sbgp == nil || traf.Sbgp.GroupingType != "seig" {
					f = append(f, RuleHLSKeyRotation.New(loc, "key %s from a seig sample group without a seig sbgp box", k.kid))
				}
				if len(init.PsshSystems) > 0 && len(frag.Moof.Psshs) == 0 {
					f = append(f, RuleHLSKeyRotation.New(loc, "key %s but no pssh box, although the CMAF header has %d", k.kid, len(init.PsshSystems)))
				}
			}
		}
	}
	// WAVEHLS-12, WAVEHLS-13
	for _, k := range keys[min(1, len(keys)):] {
		if k.kid != keys[0].kid {
			f = append(f, RuleHLSSegmentKey.New(loc, "keys %s and %s in one segment", keys[0].kid, k.kid))
			break
		}
		if !bytes.Equal(k.iv, keys[0].iv) {
			f = append(f, RuleHLSSegmentKey.New(loc, "constant IVs %x and %x in one segment", keys[0].iv, k.iv))
			break
		}
	}
	return f
}

// CheckHLSInteropSidx checks the index of a CMAF track file (WAVEHLS-1) and,
// on demand, the bitrate of its fragments (WAVEHLS-2).
func CheckHLSInteropSidx(rep *addressing.Rep, sidx *mp4.SidxBox, nSidx int) []report.Finding {
	loc := rep.Loc()
	var f []report.Finding
	if nSidx != 1 {
		f = append(f, RuleHLSSidx.New(loc, "%d sidx boxes", nSidx))
	}
	if sidx == nil || sidx.Timescale == 0 || rep.Dynamic || len(sidx.SidxRefs) < 2 {
		return f
	}
	var size, dur float64
	for _, r := range sidx.SidxRefs {
		size += float64(r.ReferencedSize)
		dur += float64(r.SubSegmentDuration)
	}
	avg := size / dur
	n, worst := 0, 0.0
	for i, r := range sidx.SidxRefs {
		if r.SubSegmentDuration == 0 || i == len(sidx.SidxRefs)-1 {
			continue // the last fragment may be short
		}
		dev := (float64(r.ReferencedSize)/float64(r.SubSegmentDuration) - avg) / avg
		if math.Abs(dev) > 0.1 {
			n++
			if math.Abs(dev) > math.Abs(worst) {
				worst = dev
			}
		}
	}
	if n > 0 {
		ts := float64(sidx.Timescale)
		f = append(f, RuleHLSBitrate.New(loc, "%d of %d fragments differ more than 10%% from the track average of %.0f bit/s (worst %+.0f%%)",
			n, len(sidx.SidxRefs)-1, avg*8*ts, worst*100))
	}
	return f
}

// hlsPresentationRules checks that every cenc-encrypted media type in a
// Period also has a cbcs-encrypted AdaptationSet (WAVEHLS-5).
func hlsPresentationRules(reps []*addressing.Rep, inits []*Init) []report.Finding {
	type key struct {
		period  int
		handler string
	}
	cenc, cbcs := map[key]string{}, map[key]bool{}
	var order []key
	for i, init := range inits {
		if init == nil || !init.HLSInterop || !init.Encrypted {
			continue
		}
		k := key{reps[i].PeriodIndex, init.Handler}
		switch init.Scheme {
		case "cenc":
			if _, ok := cenc[k]; !ok {
				cenc[k] = reps[i].ASPath
				order = append(order, k)
			}
		case "cbcs":
			cbcs[k] = true
		}
	}
	var f []report.Finding
	for _, k := range order {
		if !cbcs[k] {
			f = append(f, RuleHLSCbcsVariant.New(report.Location{Path: cenc[k]}, "%s is encrypted with cenc and no %s AdaptationSet uses cbcs",
				mediaName(k.handler), mediaName(k.handler)))
		}
	}
	return f
}
