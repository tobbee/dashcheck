package mpdcheck

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// ContentProtection rules of ISO/IEC 23009-1 5.8.4.1 and 5.8.5.2, and of the
// cenc namespace of ISO/IEC 23001-7 11.

const (
	schemeCADescriptor = "urn:mpeg:dash:13818:1:CA_descriptor:2011"
	schemeClearKey     = "urn:uuid:e2719d58-a985-b3c9-781a-b030af78d30e"
)

var (
	RuleContentProtectionValue = report.Register(report.Rule{
		ID: "MPD-CP-VALUE", Severity: report.Error,
		Desc:    "ContentProtection @value must be a 4CC with an optional version for mp4protection and 4 lower-case hex digits for CA_descriptor",
		SpecRef: "ISO/IEC 23009-1 5.8.5.2",
	})
	RuleCPDefaultKID = report.Register(report.Rule{
		ID: "MPD-CP-DEFAULTKID", Severity: report.Warning,
		Desc:    "The mp4protection descriptor of Common Encryption content should have cenc:default_KID",
		SpecRef: "ISO/IEC 23009-1 5.8.5.2.2, ISO/IEC 23001-7 11.3.2",
	})
	RuleCPCenc = report.Register(report.Rule{
		ID: "MPD-CP-CENC", Severity: report.Error,
		Desc:    "cenc:default_KID must be a list of UUIDs, and cenc:pssh a base64 pssh box with the SystemID of its ContentProtection",
		SpecRef: "ISO/IEC 23001-7 11.2",
	})
	RuleCPRobustness = report.Register(report.Rule{
		ID: "MPD-CP-ROBUSTNESS", Severity: report.Error,
		Desc:    "The mp4protection and CA_descriptor ContentProtection must not have @robustness",
		SpecRef: "ISO/IEC 23009-1 5.8.5.2.2, 5.8.5.2.3",
	})
	RuleCPClearKey = report.Register(report.Rule{
		ID: "MPD-CP-CLEARKEY", Severity: report.Error,
		Desc:    "The Clear Key ContentProtection must have @value ClearKey1.0 and no Clear Key initialization data (cenc:pssh)",
		SpecRef: "DASH-IF IOP 7.6.2.4",
	})
	RuleCPRef = report.Register(report.Rule{
		ID: "MPD-CP-REF", Severity: report.Error,
		Desc:    "ContentProtection on the MPD or a Period needs @refId, @ref excludes @refId, and the source of a @ref must not be in a later Period",
		SpecRef: "ISO/IEC 23009-1 5.3.1.2, 5.3.2.2, 5.8.4.1.3, 5.8.4.1.4",
	})
)

var (
	hex4Re = regexp.MustCompile(`^[0-9a-f]{4}$`)
	// A 4CC of four characters or percent-encoded bytes (IETF RFC 8141), and
	// an optional version of up to 8 hex digits (ISO/IEC 23009-1 5.8.5.2.2).
	// The ABNF there allows only letters, but ISO/IEC 23001-7 defines cbc1.
	mp4ProtectionValueRe = regexp.MustCompile(`^(?:[A-Za-z0-9._~!$&'()*+,;=@-]|%[0-9A-Fa-f]{2}){4}(?::[0-9A-Fa-f]{1,8})?$`)
	kidRe                = regexp.MustCompile(`^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$`)
)

// SCH-55, SCH-56. cps are the ContentProtection elements of an
// AdaptationSet or Representation with @ref resolved. kidRequired is set
// where MPD-IOP-CP reports a missing cenc:default_KID, which then replaces
// the MPD-CP-DEFAULTKID warning. The Clear Key rules apply to every MPD, as
// they define the use of the DASH-IF Clear Key SystemID.
func checkContentProtectionValues(cps []*m.ContentProtectionType, path string, kidRequired bool) []report.Finding {
	var f []report.Finding
	for i, cp := range cps {
		loc := report.Location{Path: fmt.Sprintf("%s/ContentProtection[%d]", path, i)}
		switch cp.SchemeIdUri {
		case addressing.SchemeMP4Protection:
			if !mp4ProtectionValueRe.MatchString(cp.Value) {
				f = append(f, RuleContentProtectionValue.New(loc, "mp4protection @value %q is not a 4CC like cenc or cbcs, with an optional :version", cp.Value))
			}
			if scheme := addressing.MP4ProtectionScheme(cp.Value); addressing.CENCSchemes[scheme] && cp.DefaultKID == "" && !kidRequired {
				f = append(f, RuleCPDefaultKID.New(loc, "%s descriptor without cenc:default_KID", scheme))
			}
		case schemeCADescriptor:
			if !hex4Re.MatchString(cp.Value) {
				f = append(f, RuleContentProtectionValue.New(loc, "CA_descriptor @value %q is not 4 lower-case hex digits", cp.Value))
			}
		}
		if strings.EqualFold(string(cp.SchemeIdUri), schemeClearKey) {
			if cp.Value != "ClearKey1.0" {
				f = append(f, RuleCPClearKey.New(loc, "Clear Key @value %q, not ClearKey1.0", cp.Value))
			}
			if cp.Pssh != nil {
				f = append(f, RuleCPClearKey.New(loc, "Clear Key initialization data in cenc:pssh"))
			}
		}
	}
	return f
}

// cpElem is a ContentProtection element as written in the MPD.
type cpElem struct {
	cp     *m.ContentProtectionType
	path   string
	period int  // index of the Period, -1 on the MPD
	top    bool // on the MPD or a Period, where it is only a source for @ref
}

func contentProtectionElems(md *m.MPD) []cpElem {
	var elems []cpElem
	add := func(cps []*m.ContentProtectionType, path string, period int, top bool) {
		for i, cp := range cps {
			elems = append(elems, cpElem{cp, fmt.Sprintf("%s/ContentProtection[%d]", path, i), period, top})
		}
	}
	add(md.ContentProtection, "MPD", -1, true)
	for pi, p := range md.Periods {
		pPath := fmt.Sprintf("Period[%d]", pi)
		add(p.ContentProtections, pPath, pi, true)
		for ai, as := range p.AdaptationSets {
			asPath := fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)
			add(as.ContentProtections, asPath, pi, false)
			for _, rep := range as.Representations {
				rPath := fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id)
				add(rep.ContentProtections, rPath, pi, false)
				for si, sr := range rep.SubRepresentations {
					add(sr.ContentProtections, fmt.Sprintf("%s/SubRepresentation[%d]", rPath, si), pi, false)
				}
			}
		}
	}
	return elems
}

// protectionRules checks each ContentProtection element as written; the
// rules on inherited values are in checkContentProtectionValues.
func protectionRules(md *m.MPD) []report.Finding {
	var f []report.Finding
	elems := contentProtectionElems(md)
	sourcePeriod := map[string]int{}
	for _, e := range elems {
		if _, dup := sourcePeriod[e.cp.RefId]; e.cp.RefId != "" && !dup {
			sourcePeriod[e.cp.RefId] = e.period
		}
	}
	for _, e := range elems {
		cp, loc := e.cp, report.Location{Path: e.path}
		for kid := range strings.FieldsSeq(cp.DefaultKID) {
			if !kidRe.MatchString(kid) {
				f = append(f, RuleCPCenc.New(loc, "cenc:default_KID %q is not a UUID", kid))
			}
		}
		if cp.Pssh != nil {
			if msg := checkPssh(cp); msg != "" {
				f = append(f, RuleCPCenc.New(loc, "%s", msg))
			}
		}
		if cp.Robustness != "" && (cp.SchemeIdUri == addressing.SchemeMP4Protection || cp.SchemeIdUri == schemeCADescriptor) {
			f = append(f, RuleCPRobustness.New(loc, "@robustness %q on the %s descriptor", cp.Robustness, cp.SchemeIdUri))
		}
		switch {
		case e.top && cp.RefId == "":
			f = append(f, RuleCPRef.New(loc, "ContentProtection on the %s without @refId", strings.SplitN(e.path, "/", 2)[0]))
		case cp.Ref != "" && cp.RefId != "":
			f = append(f, RuleCPRef.New(loc, "both @ref %q and @refId %q", cp.Ref, cp.RefId))
		}
		if p, ok := sourcePeriod[cp.Ref]; cp.Ref != "" && ok && p > e.period {
			f = append(f, RuleCPRef.New(loc, "@ref %q refers to a ContentProtection in the later Period[%d]", cp.Ref, p))
		}
	}
	return f
}

// checkPssh checks that cenc:pssh is the base64 of a whole pssh box with the
// SystemID of the urn:uuid @schemeIdUri (ISO/IEC 23001-7 11.2). It returns
// what is wrong, or "".
func checkPssh(cp *m.ContentProtectionType) string {
	data, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(cp.Pssh.Value), ""))
	if err != nil {
		return fmt.Sprintf("cenc:pssh is not base64: %v", err)
	}
	box, err := mp4.DecodeBox(0, bytes.NewReader(data))
	if err != nil {
		return fmt.Sprintf("cenc:pssh is not an ISOBMFF box: %v", err)
	}
	pssh, ok := box.(*mp4.PsshBox)
	if !ok {
		return fmt.Sprintf("cenc:pssh holds a %q box, not pssh", box.Type())
	}
	if n := len(data) - int(pssh.Size()); n != 0 {
		return fmt.Sprintf("cenc:pssh has %d bytes after the pssh box", n)
	}
	uuid, ok := strings.CutPrefix(strings.ToLower(string(cp.SchemeIdUri)), "urn:uuid:")
	if !ok {
		return fmt.Sprintf("cenc:pssh in a %s descriptor, which names no DRM system", cp.SchemeIdUri)
	}
	if id := pssh.SystemID.String(); id != uuid {
		return fmt.Sprintf("cenc:pssh has SystemID %s, not the %s of @schemeIdUri", id, uuid)
	}
	return ""
}
