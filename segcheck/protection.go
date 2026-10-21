package segcheck

import (
	"bytes"
	"cmp"
	"maps"
	"slices"
	"strings"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Cross-checks of the ContentProtection descriptors in the MPD with the
// protection of the track in the initialization segment, and the
// protection rules of DASH-IF IOP 7.7 and CMAF across an AdaptationSet.

const (
	refIOPProtection = "DASH-IF IOP 7.7.1"
	// systemCommon is the SystemID of the W3C Common PSSH box.
	systemCommon = "1077efec-c0b2-4d02-ace3-3c1e52e2fb4b"
)

var (
	RuleCPMissing = report.Register(report.Rule{
		ID: "XCHK-CP-MISSING", Severity: report.Error,
		Desc:    "Encrypted tracks need ContentProtection in the MPD, and Common Encryption the mp4protection descriptor",
		SpecRef: "ISO/IEC 23009-1 5.8.4.1.1, ISO/IEC 23001-7 11.3.2",
	})
	RuleCPScheme = report.Register(report.Rule{
		ID: "XCHK-CP-SCHEME", Severity: report.Error,
		Desc:    "The mp4protection @value must be the scheme type in the schm box of the track",
		SpecRef: "ISO/IEC 23009-1 5.8.5.2.2",
	})
	RuleCPKID = report.Register(report.Rule{
		ID: "XCHK-CP-KID", Severity: report.Error,
		Desc:    "cenc:default_KID must list the default_KID in the tenc box of the track",
		SpecRef: "ISO/IEC 23001-7 11.3.2",
	})
	RuleASKID = report.Register(report.Rule{
		ID: "SEG-AS-KID", Severity: report.Error,
		Desc:    "All Representations of an IOP or CMAF AdaptationSet must be encrypted with the same tenc default_KID",
		SpecRef: "DASH-IF IOP 7.7.1, 7.7.5, ISO/IEC 23000-19 8.2.3.1",
	})
	RuleIOPPsshInit = report.Register(report.Rule{
		ID: "SEG-IOP-PSSH-INIT", Severity: report.Warning,
		Desc:    "IOP initialization segments should have no pssh boxes, but cenc:pssh in the MPD instead",
		SpecRef: refIOPProtection,
	})
	RuleIOPPssh = report.Register(report.Rule{
		ID: "SEG-IOP-PSSH", Severity: report.Error,
		Desc:    "The initialization segments of an IOP AdaptationSet must have the same pssh boxes for each SystemID, and a W3C Common pssh box must list only the default_KID",
		SpecRef: refIOPProtection,
	})
)

func protectionInitRules(rep *addressing.Rep, init *Init, loc report.Location) []report.Finding {
	var f []report.Finding
	var mp4p []*m.ContentProtectionType
	for _, cp := range rep.ContentProtections {
		if cp.SchemeIdUri == addressing.SchemeMP4Protection {
			mp4p = append(mp4p, cp)
		}
	}
	// LL-24
	switch {
	case init.Encrypted && len(rep.ContentProtections) == 0:
		f = append(f, RuleCPMissing.New(loc, "encrypted track (scheme %q) without ContentProtection in the MPD", init.Scheme))
	case init.Encrypted && addressing.CENCSchemes[init.Scheme] && len(mp4p) == 0:
		f = append(f, RuleCPMissing.New(loc, "%s-encrypted track without the %s descriptor", init.Scheme, addressing.SchemeMP4Protection))
	}
	for _, cp := range mp4p {
		switch {
		case cp.Value == "": // MPD-CP-VALUE
		case !init.Encrypted:
			f = append(f, RuleCPScheme.New(loc, "mp4protection @value %q, but the track is not encrypted", cp.Value))
		case init.Scheme != "" && addressing.MP4ProtectionScheme(cp.Value) != init.Scheme:
			f = append(f, RuleCPScheme.New(loc, "mp4protection @value %q, but the schm scheme type is %q", cp.Value, init.Scheme))
		}
	}
	if isIOP(rep) && init.Encrypted && len(init.pssh) > 0 {
		var ids []string
		for _, p := range init.pssh {
			ids = append(ids, p.SystemID.String())
		}
		f = append(f, RuleIOPPsshInit.New(loc, "pssh boxes for SystemIDs %s", strings.Join(ids, ", ")))
		for _, p := range init.pssh {
			if p.SystemID.String() == systemCommon && (len(p.KIDs) != 1 || normalizeKID(p.KIDs[0].String()) != normalizeKID(init.DefaultKID)) {
				f = append(f, RuleIOPPssh.New(loc, "the W3C Common pssh box lists the KIDs %v, not only the tenc default_KID %s", p.KIDs, init.DefaultKID))
			}
		}
	}
	if init.Encrypted && init.DefaultKID != "" {
		kid := normalizeKID(init.DefaultKID)
		for _, cp := range rep.ContentProtections {
			if cp.DefaultKID == "" {
				continue
			}
			if !slices.ContainsFunc(strings.Fields(cp.DefaultKID), func(k string) bool { return normalizeKID(k) == kid }) {
				f = append(f, RuleCPKID.New(loc, "cenc:default_KID %q of the %s descriptor does not list the tenc default_KID %s",
					cp.DefaultKID, cp.SchemeIdUri, init.DefaultKID))
			}
		}
	}
	return f
}

func protectionAdaptationSetRules(reps []*addressing.Rep, inits []*Init) []report.Finding {
	iop, cmaf := isIOP(reps[0]), false
	for _, init := range inits {
		cmaf = cmaf || (init != nil && init.CMAF)
	}
	var f []report.Finding
	loc := report.Location{Path: reps[0].ASPath}
	// SEG-DVB-SWITCHABLE checks the default KIDs of DVB AdaptationSets.
	if (iop || cmaf) && !isDVB(reps[0]) {
		kids := map[string]bool{}
		var desc []string
		encrypted := false
		for i, init := range inits {
			if init == nil {
				continue
			}
			kid := "clear"
			if init.Encrypted {
				encrypted = true
				kid = cmp.Or(init.DefaultKID, "no tenc")
			}
			kids[normalizeKID(kid)] = true
			desc = append(desc, reps[i].ID+": "+kid)
		}
		if encrypted && len(kids) > 1 {
			f = append(f, RuleASKID.New(loc, "the Representations have different default KIDs (%s)", strings.Join(desc, ", ")))
		}
	}
	if iop {
		f = append(f, iopPsshRules(reps, inits, loc)...)
	}
	return f
}

// iopPsshRules checks that, when the initialization segments of an
// AdaptationSet have pssh boxes, each has the same ones for each SystemID, so
// that the license of any Representation allows switching (DASH-IF IOP 7.7.1).
func iopPsshRules(reps []*addressing.Rep, inits []*Init, loc report.Location) []report.Finding {
	var ids []string
	var boxes []map[string]string // per Representation, the encoded pssh boxes by SystemID
	systems := map[string]bool{}
	for i, init := range inits {
		if init == nil || !init.Encrypted {
			continue
		}
		byID := map[string]string{}
		for _, p := range init.pssh {
			var buf bytes.Buffer
			if err := p.Encode(&buf); err == nil {
				byID[p.SystemID.String()] += buf.String()
				systems[p.SystemID.String()] = true
			}
		}
		ids = append(ids, reps[i].ID)
		boxes = append(boxes, byID)
	}
	var f []report.Finding
	for _, system := range slices.Sorted(maps.Keys(systems)) {
		versions := map[string]string{} // encoded boxes to a version letter
		var desc []string
		missing := false
		for i, byID := range boxes {
			b, ok := byID[system]
			if !ok {
				missing = true
				desc = append(desc, ids[i]+": none")
				continue
			}
			if _, seen := versions[b]; !seen {
				versions[b] = string(rune('A' + len(versions)))
			}
			desc = append(desc, ids[i]+": "+versions[b])
		}
		if missing || len(versions) > 1 {
			f = append(f, RuleIOPPssh.New(loc, "the pssh boxes for SystemID %s differ between the initialization segments (%s)", system, strings.Join(desc, ", ")))
		}
	}
	return f
}

// normalizeKID returns a KID in lower case without hyphens, so that the
// UUID string of the MPD compares with the tenc bytes in any letter case.
func normalizeKID(kid string) string {
	return strings.ToLower(strings.ReplaceAll(kid, "-", ""))
}
