package segcheck

import (
	"slices"
	"strings"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Cross-checks of the ContentProtection descriptors in the MPD with the
// protection of the track in the initialization segment.

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

// normalizeKID returns a KID in lower case without hyphens, so that the
// UUID string of the MPD compares with the tenc bytes in any letter case.
func normalizeKID(kid string) string {
	return strings.ToLower(strings.ReplaceAll(kid, "-", ""))
}
