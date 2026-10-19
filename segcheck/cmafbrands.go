package segcheck

import (
	"slices"
	"strings"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Rules that follow the structural CMAF brands a track claims (ISO/IEC
// 23000-19 7.2): the minor version with a CMAF major brand, and the 'cmf2'
// constraints of clause 7.7, which among other things make each CMAF chunk
// decodable without the defaults in the CMAF header's TrackExtendsBox.

var (
	RuleCMAFMinorVersion = report.Register(report.Rule{
		ID: "SEG-CMAF-MINORVERSION", Severity: report.Error,
		Desc:    "With a structural CMAF brand (cmfc or cmf2) as major_brand, ftyp and styp must have minor_version 0",
		SpecRef: refCMAF + " 7.2",
	})
	RuleCMF2Elst = report.Register(report.Rule{
		ID: "SEG-CMF2-ELST", Severity: report.Error,
		Desc: "A video CMAF track with the cmf2 brand must not have an edit list", SpecRef: refCMAF + " 7.7.2",
	})
	RuleCMF2Trun = report.Register(report.Rule{
		ID: "SEG-CMF2-TRUN", Severity: report.Error,
		Desc:    "A video CMAF track with the cmf2 brand must use trun version 1, except in a CMAF track file",
		SpecRef: refCMAF + " 7.7.3",
	})
	RuleCMF2Defaults = report.Register(report.Rule{
		ID: "SEG-CMF2-DEFAULTS", Severity: report.Error,
		Desc:    "Each chunk of a CMAF track with the cmf2 brand must set sample durations, sizes and flags in tfhd or trun, not only in trex",
		SpecRef: refCMAF + " 7.7.3",
	})
)

// structuralBrands are the structural CMAF brands of ISO/IEC 23000-19 table 1.
var structuralBrands = []string{"cmfc", "cmf2"}

// checkMinorVersion checks the minor_version of an ftyp or styp box.
func checkMinorVersion(box, major string, minor uint32, loc report.Location) []report.Finding {
	if !slices.Contains(structuralBrands, major) || minor == 0 {
		return nil
	}
	return []report.Finding{RuleCMAFMinorVersion.New(loc, "%s major_brand %s with minor_version %d", box, major, minor)}
}

func cmafBrandInitRules(init *Init, ftyp *mp4.FtypBox, loc report.Location) []report.Finding {
	if !init.CMAF || ftyp == nil {
		return nil
	}
	f := checkMinorVersion("ftyp", ftyp.MajorBrand(), ftyp.MinorVersion(), loc)
	if slices.Contains(init.Brands, "cmf2") && init.Handler == "vide" && init.HasEditList {
		f = append(f, RuleCMF2Elst.New(loc, "video track with the cmf2 brand has an edit list"))
	}
	return f
}

func cmafBrandMediaRules(rep *addressing.Rep, init *Init, mf *mp4.File, loc report.Location) []report.Finding {
	if !init.CMAF {
		return nil
	}
	var f []report.Finding
	cmf2 := slices.Contains(init.Brands, "cmf2")
	for _, ms := range mf.Segments {
		if ms.Styp != nil {
			f = append(f, checkMinorVersion("styp", ms.Styp.MajorBrand(), ms.Styp.MinorVersion(), loc)...)
			cmf2 = cmf2 || ms.Styp.MajorBrand() == "cmf2" || slices.Contains(ms.Styp.CompatibleBrands(), "cmf2")
		}
	}
	if !cmf2 {
		return f
	}
	// A CMAF track file (one file with the header and all fragments) may use
	// trun version 0; in DASH that is the on-demand SegmentBase addressing.
	checkVersion := init.Handler == "vide" && rep.Kind != addressing.KindSegmentBase
	var badVersion, noDefaults int
	var firstVersion uint8
	var missing []string
	for _, ms := range mf.Segments {
		for _, frag := range ms.Fragments {
			if frag.Moof == nil || frag.Moof.Traf == nil || frag.Moof.Traf.Tfhd == nil {
				continue
			}
			traf := frag.Moof.Traf
			for _, trun := range traf.Truns {
				if checkVersion && trun.Version != 1 {
					if badVersion == 0 {
						firstVersion = trun.Version
					}
					badVersion++
				}
				if m := missingDefaults(traf.Tfhd, trun); len(m) > 0 {
					noDefaults++
					for _, x := range m {
						if !slices.Contains(missing, x) {
							missing = append(missing, x)
						}
					}
				}
			}
		}
	}
	if badVersion > 0 {
		f = append(f, RuleCMF2Trun.New(loc, "trun version %d in %d track runs of a video track with the cmf2 brand", firstVersion, badVersion))
	}
	if noDefaults > 0 {
		f = append(f, RuleCMF2Defaults.New(loc, "%d track runs leave the sample %s to trex", noDefaults, strings.Join(missing, ", ")))
	}
	return f
}

// missingDefaults returns the sample fields that neither the tfhd defaults
// nor the trun of a track run set, so that they come from trex: "duration",
// "size" and "flags". first_sample_flags covers the flags of a single-sample run.
func missingDefaults(tfhd *mp4.TfhdBox, trun *mp4.TrunBox) []string {
	var m []string
	if !tfhd.HasDefaultSampleDuration() && !trun.HasSampleDuration() {
		m = append(m, "duration")
	}
	if !tfhd.HasDefaultSampleSize() && !trun.HasSampleSize() {
		m = append(m, "size")
	}
	singleWithFlags := trun.HasFirstSampleFlags() && trun.SampleCount() == 1
	if !tfhd.HasDefaultSampleFlags() && !trun.HasSampleFlags() && !singleWithFlags {
		m = append(m, "flags")
	}
	return m
}
