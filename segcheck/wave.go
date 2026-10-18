package segcheck

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Checks ported from the CTAWave module of the DASH-IF Conformance tool,
// following CTA-5001-F (WAVE Content Specification, 2025 edition). They
// apply to MPDs with a urn:cta:wave: profile, as the WAVE test content uses.

const refWAVE = "CTA-5001-F"

var (
	RuleWAVEProfile = report.Register(report.Rule{
		ID: "SEG-WAVE-PROFILE", Severity: report.Error,
		Desc: "Each selection set of WAVE content needs a switching set whose tracks all conform to one WAVE media " +
			"profile (CTA-5001 tables 1 to 3)",
		SpecRef: refWAVE + " 4.1",
	})
	RuleWAVEScheme = report.Register(report.Rule{
		ID: "SEG-WAVE-SCHEME", Severity: report.Warning,
		Desc: "WAVE content should use only one Common Encryption scheme: CMAF presentation profiles allow one " +
			"per presentation, and the Baseline Splice Constraint Profile one per program",
		SpecRef: refWAVE + " 5, 7.2.2",
	})
	RuleWAVESplice = report.Register(report.Rule{
		ID: "SEG-WAVE-SPLICE", Severity: report.Info,
		Desc: "Sequential switching sets that do not meet the optional WAVE Baseline Splice Constraint Profile " +
			"(same media profile and sample entry type) or its recommendations (same picture aspect ratio and frame rate family)",
		SpecRef: refWAVE + " 7.2.2",
	})
)

// waveBrands are the media profile brands of CTA-5001-F tables 1 to 3, by
// handler. 'cfsd' conforms to 'cfhd' at a lower resolution.
var waveBrands = map[string][]string{
	"vide": {"cfhd", "cfsd", "chdf", "chh1", "cud1", "clg1", "chd1", "cdm1", "cdm4", "av01"},
	"soun": {"caac", "caaa", "camc", "ceac", "ca4s", "cmhs", "dts1", "dts2"},
	"subt": {"im1t", "im1i", "im2t", "im2i"},
	"text": {"im1t", "im1i", "im2t", "im2i"},
}

func isWAVE(rep *addressing.Rep) bool { return rep.HasProfilePrefix("urn:cta:wave:") }

// waveSet is one AdaptationSet (CMAF switching set) with its tracks.
type waveSet struct {
	path, key string // key groups the switching sets of a selection set
	period    int
	reps      []*addressing.Rep
	inits     []*Init
	unknown   bool // a track could not be checked
}

func (s *waveSet) handler() string {
	for _, init := range s.inits {
		if init != nil {
			return init.Handler
		}
	}
	return ""
}

// waveBrand returns a WAVE media profile brand that all tracks of the set
// conform to, or "".
func (s *waveSet) waveBrand() string {
	h := s.handler()
	for _, b := range waveBrands[h] {
		all := true
		for i, init := range s.inits {
			if init == nil || !conformsTo(b, s.reps[i], init) {
				all = false
				break
			}
		}
		if all {
			return b
		}
	}
	return ""
}

// conformsTo reports whether a track conforms to a WAVE media profile. The
// brand SHOULD be in the CMAF header (ISO/IEC 23000-19 10.1) but need not
// be, so the MPD @containerProfiles and the track itself are also used, as
// upstream does. The limits of signalled brands are checked by
// SEG-CMAF-MEDIAPROFILE.
func conformsTo(brand string, rep *addressing.Rep, init *Init) bool {
	if slices.Contains(init.Brands, brand) || slices.Contains(rep.ContainerProfiles, brand) {
		return true
	}
	v := init.Video
	if vp, ok := videoProfiles[brand]; ok {
		info := sampleEntryInfo{width: uint16(v.width), height: uint16(v.height), video: v}
		return v.hasConfig && vp.codec == v.codec && len(checkVideoProfile(brand, vp, rep, info, report.Location{})) == 0
	}
	switch brand {
	case "caac":
		a := init.aac
		return a != nil && a.sampleRate <= 48000 && a.channels <= 2 && slices.Contains([]int{2, 5, 29}, a.objectType)
	case "camc":
		a := init.aac
		return a != nil && a.sampleRate <= 48000 && a.channels <= 6 && slices.Contains([]int{2, 5, 29}, a.objectType)
	case "ceac":
		return init.SampleEntry == "ec-3" || init.SampleEntry == "ac-3"
	case "ca4s":
		return init.SampleEntry == "ac-4"
	case "cmhs":
		return init.SampleEntry == "mhm1" || init.SampleEntry == "mhm2"
	case "dts1":
		return slices.Contains([]string{"dtsc", "dtsh", "dtsl", "dtse"}, init.SampleEntry)
	case "dts2":
		return init.SampleEntry == "dtsx"
	case "im1t", "im1i", "im2t", "im2i":
		return strings.HasPrefix(rep.Codecs, "stpp.ttml."+brand)
	}
	return false
}

// waveRules runs the WAVE checks that need all Representations (WAVE-2 to
// WAVE-9 and the subtitle and video equivalents).
func waveRules(reps []*addressing.Rep, inits []*Init) []report.Finding {
	if len(reps) == 0 || !slices.ContainsFunc(reps, isWAVE) {
		return nil
	}
	var sets []*waveSet
	byPath := map[string]*waveSet{}
	for i, r := range reps {
		s := byPath[r.ASPath]
		if s == nil {
			s = &waveSet{path: r.ASPath, period: r.PeriodIndex}
			byPath[r.ASPath] = s
			sets = append(sets, s)
		}
		s.reps = append(s.reps, r)
		s.inits = append(s.inits, inits[i])
		s.unknown = s.unknown || inits[i] == nil
	}
	for _, s := range sets {
		s.key = fmt.Sprintf("Period[%d] %s %s", s.period, s.handler(), s.reps[0].Lang)
	}
	var f []report.Finding
	// WAVE-2, WAVE-15, WAVE-19: per selection set (switching sets of the
	// same media type and language in a Period).
	covered, unknown, seen := map[string]bool{}, map[string]bool{}, map[string]*waveSet{}
	var keys []string
	for _, s := range sets {
		if waveBrands[s.handler()] == nil {
			continue
		}
		if seen[s.key] == nil {
			seen[s.key] = s
			keys = append(keys, s.key)
		}
		unknown[s.key] = unknown[s.key] || s.unknown
		covered[s.key] = covered[s.key] || s.waveBrand() != ""
	}
	for _, k := range keys {
		if !covered[k] && !unknown[k] {
			s := seen[k]
			f = append(f, RuleWAVEProfile.New(report.Location{Path: s.path},
				"no %s switching set where all tracks conform to one of %s", mediaName(s.handler()), strings.Join(waveBrands[s.handler()], ", ")))
		}
	}
	// WAVE-5
	schemes := map[string]bool{}
	for _, init := range inits {
		if init != nil && init.Encrypted {
			schemes[init.Scheme] = true
		}
	}
	if len(schemes) > 1 {
		f = append(f, RuleWAVEScheme.New(report.Location{Path: "MPD"}, "schemes %s", strings.Join(slices.Sorted(maps.Keys(schemes)), ", ")))
	}
	// WAVE-4, WAVE-6 to WAVE-9, WAVE-17, WAVE-21: sequential switching sets
	for _, s := range sets {
		if prev := previousSet(sets, s); prev != nil && !s.unknown && !prev.unknown {
			if problems := spliceProblems(prev, s); len(problems) > 0 {
				f = append(f, RuleWAVESplice.New(report.Location{Path: s.path}, "after %s: %s", prev.path, strings.Join(problems, "; ")))
			}
		}
	}
	return f
}

// previousSet finds the sequential switching set in the previous Period: the
// AdaptationSet with the same @id, or else the one at the same position
// among the sets of the same media type.
func previousSet(sets []*waveSet, s *waveSet) *waveSet {
	if s.period == 0 {
		return nil
	}
	var prev []*waveSet
	for _, p := range sets {
		if p.period == s.period-1 && p.handler() == s.handler() {
			prev = append(prev, p)
		}
	}
	if id := s.reps[0].ASID; id != "" {
		for _, p := range prev {
			if p.reps[0].ASID == id {
				return p
			}
		}
	}
	pos := 0
	for _, o := range sets {
		if o == s {
			break
		}
		if o.period == s.period && o.handler() == s.handler() {
			pos++
		}
	}
	if pos < len(prev) {
		return prev[pos]
	}
	return nil
}

// spliceProblems compares two sequential switching sets with Table 5
// (required by the Baseline Splice Constraint Profile) and Table 6
// (recommended).
func spliceProblems(a, b *waveSet) []string {
	var p []string
	if ba, bb := a.waveBrand(), b.waveBrand(); ba != bb {
		p = append(p, fmt.Sprintf("media profile %q, then %q", ba, bb))
	}
	if ea, eb := sampleEntries(a), sampleEntries(b); ea != eb {
		p = append(p, fmt.Sprintf("sample entries %s, then %s", ea, eb))
	}
	if a.handler() != "vide" {
		return p
	}
	if pa, pb := pictureAspect(a.reps[0]), pictureAspect(b.reps[0]); pa > 0 && pb > 0 && math.Abs(pa-pb) > 0.01 {
		p = append(p, fmt.Sprintf("picture aspect ratio %.3f, then %.3f (recommended the same)", pa, pb))
	}
	fa, _ := parseRate(a.reps[0].FrameRate)
	fb, _ := parseRate(b.reps[0].FrameRate)
	if fa > 0 && fb > 0 && !sameRateFamily(fa, fb) {
		p = append(p, fmt.Sprintf("frame rate %s, then %s (recommended the same family)", a.reps[0].FrameRate, b.reps[0].FrameRate))
	}
	return p
}

func sampleEntries(s *waveSet) string {
	var e []string
	for _, init := range s.inits {
		if init != nil && !slices.Contains(e, init.SampleEntry) {
			e = append(e, init.SampleEntry)
		}
	}
	slices.Sort(e)
	return strings.Join(e, ",")
}

// pictureAspect is the display aspect ratio of a Representation, 0 if unknown.
func pictureAspect(r *addressing.Rep) float64 {
	if r.Width == 0 || r.Height == 0 {
		return 0
	}
	sar := 1.0
	if w, h, ok := strings.Cut(r.Sar, ":"); ok {
		sw, err1 := strconv.ParseFloat(w, 64)
		sh, err2 := strconv.ParseFloat(h, 64)
		if err1 == nil && err2 == nil && sh > 0 {
			sar = sw / sh
		}
	}
	return float64(r.Width) * sar / float64(r.Height)
}

// sameRateFamily reports whether one frame rate is an integer multiple of the other.
func sameRateFamily(a, b float64) bool {
	r := max(a, b) / min(a, b)
	return math.Abs(r-math.Round(r)) < 0.01
}
