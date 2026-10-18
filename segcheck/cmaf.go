package segcheck

import (
	"slices"
	"strings"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Structural CMAF checks ported from the CMAF module of the DASH-IF
// Conformance tool (ISO/IEC 23000-19). They apply to tracks declared as CMAF.

const profileCMAF = "urn:mpeg:dash:profile:cmaf:2019"

var (
	RuleCMAFInit = report.Register(report.Rule{
		ID: "SEG-CMAF-INIT", Severity: report.Error,
		Desc:    "A CMAF header must describe the track: decoder configuration and size for video, sample rate and channels for audio",
		SpecRef: refCMAF + " 7.3.5",
	})
	RuleCMAFColr = report.Register(report.Rule{
		ID: "SEG-CMAF-COLR", Severity: report.Error,
		Desc: "A CMAF HEVC sample entry must contain a colr box of type nclx", SpecRef: refCMAF + " 7.3.2.4",
	})
	RuleCMAFTopLevel = report.Register(report.Rule{
		ID: "SEG-CMAF-TOPLEVEL", Severity: report.Error,
		Desc: "meta and udta boxes must not be at the top level of CMAF files", SpecRef: refCMAF + " 7.5.2",
	})
	RuleCMAFEncryption = report.Register(report.Rule{
		ID: "SEG-CMAF-ENCRYPTION", Severity: report.Error,
		Desc:    "Tracks must match the encryption of their CMAF representation profile (cmfhd clear, cmfhdc cenc, cmfhds cbcs)",
		SpecRef: refCMAF + " A.1",
	})
	RuleCMAFCompTime = report.Register(report.Rule{
		ID: "SEG-CMAF-COMPTIME", Severity: report.Error,
		Desc:    "CMAF video with composition offsets must use either trun version 1 or an edit list with trun version 0, not both",
		SpecRef: refCMAF + " 9.2.1",
	})
	RuleCMAFDataOffset = report.Register(report.Rule{
		ID: "SEG-CMAF-DATAOFFSET", Severity: report.Error,
		Desc:    "CMAF samples must be addressed by trun data offsets relative to the moof",
		SpecRef: refCMAF + " 7.3.5",
	})
	RuleCMAFSenc = report.Register(report.Rule{
		ID: "SEG-CMAF-SENC", Severity: report.Error,
		Desc:    "Each encrypted CMAF fragment must carry its sample encryption information",
		SpecRef: refCMAF + " 7.3.2.4",
	})
	RuleCMAFSidx = report.Register(report.Rule{
		ID: "SEG-CMAF-SIDX", Severity: report.Error,
		Desc:    "Each subsegment referenced by a CMAF sidx box must be a single media fragment",
		SpecRef: refCMAF + " 7.3.3.3",
	})
	RuleCMAFSubs = report.Register(report.Rule{
		ID: "SEG-CMAF-SUBS", Severity: report.Error,
		Desc: "Each fragment of an im1i subtitle track must contain a subs box", SpecRef: refCMAF + " 7.5.20",
	})
	RuleCMAFSwitchingSet = report.Register(report.Rule{
		ID: "SEG-CMAF-SWITCHINGSET", Severity: report.Error,
		Desc: "All tracks of a CMAF switching set must have the same media type", SpecRef: refCMAF + " 7.3.9.1",
	})
)

// isCMAF reports whether a track is declared as CMAF.
func isCMAF(rep *addressing.Rep, brands []string) bool {
	return rep.HasProfile(profileCMAF) || slices.Contains(brands, "cmfc") || slices.Contains(brands, "cmf2")
}

func cmafInitRules(rep *addressing.Rep, init *Init, info sampleEntryInfo, top []mp4.Box, loc report.Location) []report.Finding {
	if !init.CMAF {
		return nil
	}
	var f []report.Finding
	// CMAF-11, CMAF-12, CMAF-13
	switch {
	case info.fourCC == "avc1" || info.fourCC == "avc3" || info.fourCC == "hvc1" || info.fourCC == "hev1":
		if !info.video.hasConfig {
			f = append(f, RuleCMAFInit.New(loc, "%s sample entry without decoder configuration", info.fourCC))
		}
		if info.width == 0 || info.height == 0 {
			f = append(f, RuleCMAFInit.New(loc, "sample entry with size %dx%d", info.width, info.height))
		}
	case init.Handler == "soun":
		if info.sampleRate == 0 || info.channels == 0 {
			f = append(f, RuleCMAFInit.New(loc, "audio sample entry with sample rate %d and %d channels", info.sampleRate, info.channels))
		}
	}
	// CMAF-14 (upstream only checks hev1)
	if (info.fourCC == "hvc1" || info.fourCC == "hev1") && !info.colrNclx {
		f = append(f, RuleCMAFColr.New(loc, "%s sample entry without colr box of type nclx", info.fourCC))
	}
	// CMAF-16, CMAF-17
	f = append(f, checkTopLevel(top, loc)...)
	// CMAF-6, CMAF-7, CMAF-8
	switch {
	case rep.HasProfile("urn:mpeg:cmaf:representation_profile:cmfhd:2017") && init.Encrypted:
		f = append(f, RuleCMAFEncryption.New(loc, "cmfhd representation profile but the track is encrypted"))
	case rep.HasProfile("urn:mpeg:cmaf:representation_profile:cmfhdc:2017") && init.Encrypted && init.Scheme != "cenc":
		f = append(f, RuleCMAFEncryption.New(loc, "cmfhdc representation profile but scheme %q, not cenc", init.Scheme))
	case rep.HasProfile("urn:mpeg:cmaf:representation_profile:cmfhds:2017") && init.Encrypted && init.Scheme != "cbcs":
		f = append(f, RuleCMAFEncryption.New(loc, "cmfhds representation profile but scheme %q, not cbcs", init.Scheme))
	}
	return f
}

func checkTopLevel(boxes []mp4.Box, loc report.Location) []report.Finding {
	var f []report.Finding
	for _, b := range boxes {
		if t := b.Type(); t == "meta" || t == "udta" {
			f = append(f, RuleCMAFTopLevel.New(loc, "%s box at the top level", t))
		}
	}
	return f
}

func cmafMediaRules(rep *addressing.Rep, init *Init, mf *mp4.File, loc report.Location) []report.Finding {
	if !init.CMAF {
		return nil
	}
	var f []report.Finding
	f = append(f, checkTopLevel(mf.Children, loc)...)
	video := init.Handler == "vide"
	versions := map[byte]bool{}
	hasCTO := false
	fragments := 0
	for _, ms := range mf.Segments {
		// CMAF-19
		for _, sidx := range ms.Sidxs {
			for _, ref := range sidx.SidxRefs {
				if ref.ReferenceType != 0 {
					f = append(f, RuleCMAFSidx.New(loc, "sidx reference to another sidx box"))
					break
				}
			}
		}
		for fi, frag := range ms.Fragments {
			fragments++
			if frag.Moof == nil {
				continue
			}
			for _, traf := range frag.Moof.Trafs {
				for _, trun := range traf.Truns {
					versions[trun.Version] = true
					// CMAF-4
					if !trun.HasDataOffset() {
						f = append(f, RuleCMAFDataOffset.New(loc, "trun in fragment %d without data offset", fi))
					}
					for _, s := range trun.Samples {
						if s.CompositionTimeOffset != 0 {
							hasCTO = true
						}
					}
				}
				if traf.Tfhd != nil && !traf.Tfhd.DefaultBaseIfMoof() {
					f = append(f, RuleCMAFDataOffset.New(loc, "tfhd in fragment %d without default-base-is-moof", fi))
				}
				// CMAF-15
				if init.Encrypted && traf.Senc == nil && traf.UUIDSenc == nil {
					f = append(f, RuleCMAFSenc.New(loc, "encrypted fragment %d without senc box", fi))
				}
				// CMAF-21
				if strings.Contains(rep.Codecs, "im1i") && !slices.ContainsFunc(traf.Children, func(b mp4.Box) bool { return b.Type() == "subs" }) {
					f = append(f, RuleCMAFSubs.New(loc, "im1i fragment %d without subs box", fi))
				}
			}
		}
	}
	// CMAF-19: a subsegment of SegmentBase content must be one fragment.
	if rep.Kind == addressing.KindSegmentBase && fragments > 1 {
		f = append(f, RuleCMAFSidx.New(loc, "subsegment with %d fragments", fragments))
	}
	// CMAF-3
	if video {
		switch {
		case len(versions) > 1:
			f = append(f, RuleCMAFCompTime.New(loc, "mixed trun versions"))
		case versions[0] && hasCTO && !init.HasEditList:
			f = append(f, RuleCMAFCompTime.New(loc, "trun version 0 with composition offsets but no edit list"))
		case versions[1] && init.HasEditList:
			f = append(f, RuleCMAFCompTime.New(loc, "trun version 1 together with an edit list"))
		}
	}
	return f
}

// cmafAdaptationSetRules checks a CMAF switching set (CMAF-18).
func cmafAdaptationSetRules(reps []*addressing.Rep, inits []*Init) []report.Finding {
	handlers := map[string]bool{}
	cmaf := false
	for _, init := range inits {
		if init == nil {
			continue
		}
		cmaf = cmaf || init.CMAF
		handlers[init.Handler] = true
	}
	if !cmaf || len(handlers) < 2 {
		return nil
	}
	var hs []string
	for h := range handlers {
		hs = append(hs, h)
	}
	slices.Sort(hs)
	return []report.Finding{RuleCMAFSwitchingSet.New(report.Location{Path: reps[0].ASPath},
		"tracks with different handlers: %s", strings.Join(hs, ", "))}
}
