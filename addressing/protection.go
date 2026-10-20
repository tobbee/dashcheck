package addressing

import (
	"cmp"

	m "github.com/Eyevinn/dash-mpd/mpd"
)

// ContentProtectionSources returns the ContentProtection elements with a
// @refId by that @refId: the "source" elements that others refer to with
// @ref (ISO/IEC 23009-1 5.8.4.1.3). The schema check reports duplicate
// @refId values; the first element wins.
func ContentProtectionSources(md *m.MPD) map[string]*m.ContentProtectionType {
	sources := map[string]*m.ContentProtectionType{}
	add := func(cps []*m.ContentProtectionType) {
		for _, cp := range cps {
			if _, dup := sources[cp.RefId]; cp.RefId != "" && !dup {
				sources[cp.RefId] = cp
			}
		}
	}
	add(md.ContentProtection)
	for _, p := range md.Periods {
		add(p.ContentProtections)
		for _, as := range p.AdaptationSets {
			add(as.ContentProtections)
			for _, rep := range as.Representations {
				add(rep.ContentProtections)
				for _, sr := range rep.SubRepresentations {
					add(sr.ContentProtections)
				}
			}
		}
	}
	return sources
}

// ResolveContentProtections returns cps with each referencing element (one
// with @ref) replaced by a copy that also has the attributes and elements it
// inherits from its source (ISO/IEC 23009-1 5.8.4.1.3). Elements without
// @ref, or whose source is missing, are returned as they are.
func ResolveContentProtections(cps []*m.ContentProtectionType, sources map[string]*m.ContentProtectionType) []*m.ContentProtectionType {
	resolved := make([]*m.ContentProtectionType, len(cps))
	for i, cp := range cps {
		resolved[i] = cp
		if src := sources[cp.Ref]; cp.Ref != "" && src != nil {
			resolved[i] = inherit(cp, src)
		}
	}
	return resolved
}

// inherit returns cp with the attributes and elements of src that cp does not have.
func inherit(cp, src *m.ContentProtectionType) *m.ContentProtectionType {
	r := *src
	r.RefId, r.Ref = "", cp.Ref
	r.SchemeIdUri = cmp.Or(cp.SchemeIdUri, src.SchemeIdUri)
	r.Value = cmp.Or(cp.Value, src.Value)
	r.Id = cmp.Or(cp.Id, src.Id)
	r.Robustness = cmp.Or(cp.Robustness, src.Robustness)
	r.DefaultKID = cmp.Or(cp.DefaultKID, src.DefaultKID)
	r.Pssh = cmp.Or(cp.Pssh, src.Pssh)
	r.MSPro = cmp.Or(cp.MSPro, src.MSPro)
	r.LaURL = cmp.Or(cp.LaURL, src.LaURL)
	r.AuthzURL = cmp.Or(cp.AuthzURL, src.AuthzURL)
	r.MarlinContentIds = cmp.Or(cp.MarlinContentIds, src.MarlinContentIds)
	r.UrlQueryInfo = cmp.Or(cp.UrlQueryInfo, src.UrlQueryInfo)
	r.ExtUrlQueryInfo = cmp.Or(cp.ExtUrlQueryInfo, src.ExtUrlQueryInfo)
	if len(cp.Certurls) > 0 {
		r.Certurls = cp.Certurls
	}
	if len(cp.SCTE214ContentIdentifiers) > 0 {
		r.SCTE214ContentIdentifiers = cp.SCTE214ContentIdentifiers
	}
	return &r
}
