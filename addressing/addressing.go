// Package addressing turns an MPD into a list of Representations with their
// segment addresses and times. It is pure computation without I/O, so the same
// code serves static and dynamic MPDs: a dynamic MPD is just a static one seen
// through the availability window at a given wall-clock time.
package addressing

import (
	"fmt"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/report"
)

// Kind is the addressing scheme of a Representation.
type Kind int

const (
	KindUnknown        Kind = iota
	KindNumber              // SegmentTemplate with @duration and $Number$
	KindTimelineTime        // SegmentTemplate with SegmentTimeline and $Time$
	KindTimelineNumber      // SegmentTemplate with SegmentTimeline and $Number$
	KindSegmentBase         // single file with sidx (on-demand profile)
	KindSegmentList         // explicit SegmentList
)

func (k Kind) String() string {
	switch k {
	case KindNumber:
		return "SegmentTemplate/$Number$"
	case KindTimelineTime:
		return "SegmentTimeline/$Time$"
	case KindTimelineNumber:
		return "SegmentTimeline/$Number$"
	case KindSegmentBase:
		return "SegmentBase"
	case KindSegmentList:
		return "SegmentList"
	default:
		return "unknown"
	}
}

// Segment is one media segment, or one Partial Segment of a Segment Sequence
// (ISO/IEC 23009-1 5.3.9.7).
type Segment struct {
	// Number is the segment number (number-based and timeline addressing).
	// With $SubNumber$ it is the number of the Segment Sequence.
	Number uint64
	// SubNumber is the position of a Partial Segment in its Segment Sequence,
	// from 0. It is 0 for other segments.
	SubNumber uint64
	Partial   bool   // a Partial Segment of a Segment Sequence
	Time      uint64 // earliest presentation time in Timescale ticks, including presentationTimeOffset
	Dur       uint64 // duration in Timescale ticks
	URL       string
	Range     string // byte range "first-last" for SegmentBase and SegmentList@mediaRange
	sub       bool   // addressed with $SubNumber$
}

// ID returns a short segment identifier for locations.
func (s Segment) ID(k Kind) string {
	switch {
	case k == KindTimelineTime:
		return fmt.Sprintf("t=%d", s.Time)
	case k == KindSegmentBase:
		return fmt.Sprintf("bytes=%s", s.Range)
	case s.sub:
		return fmt.Sprintf("%d.%d", s.Number, s.SubNumber)
	}
	return fmt.Sprintf("%d", s.Number)
}

// Rep is a Representation with all inherited attributes resolved.
type Rep struct {
	ID          string
	Path        string // element path, e.g. Period[0]/AdaptationSet[1]/Representation[V300]
	PeriodID    string
	PeriodIndex int
	ASPath      string // element path of the AdaptationSet
	ASID        string // AdaptationSet@id, "" if absent
	ASCodecs    string // @codecs on the AdaptationSet itself

	Profiles           []string // effective @profiles
	BitstreamSwitching bool     // effective @bitstreamSwitching
	// OnlyProfiles is set by UseOnlyProfiles. CMAF brands in the init
	// segment then do not turn on the CMAF rules; only the CMAF profile does.
	OnlyProfiles bool

	ContentType       string
	Lang              string              // AdaptationSet@lang
	ContainerProfiles []string            // effective @containerProfiles (CMAF media profile brands)
	Roles             []*m.DescriptorType // Role elements of the AdaptationSet
	MimeType          string
	Codecs            string
	Bandwidth         uint32
	Width, Height     uint32
	// MaxWidth and MaxHeight are AdaptationSet@maxWidth and @maxHeight, 0 if absent.
	MaxWidth, MaxHeight uint32
	FrameRate           string
	Sar                 string
	AudioSamplingRate   string
	StartWithSAP        uint32

	Kind Kind
	// SegmentSequences is set when the segments are Partial Segments in
	// Segment Sequences (@k greater than 1, S@ssp, or $SubNumber$).
	SegmentSequences bool
	// SequenceProperties are the effective SegmentSequenceProperties
	// elements, from the Representation or else the AdaptationSet.
	SequenceProperties []*m.SegmentSequencePropertiesType
	BaseURL            string // absolute base URL for this Representation
	InitURL            string
	InitRange          string
	IndexRange         string // SegmentBase@indexRange
	Timescale          uint32
	PTO                uint64 // presentationTimeOffset in Timescale ticks

	Dynamic     bool
	AST         time.Time // MPD@availabilityStartTime (dynamic only)
	PeriodStart float64   // seconds from the start of the presentation
	PeriodDur   float64   // seconds, or -1 if open-ended
	LastPeriod  bool      // no Period follows
	TSBD        float64   // timeShiftBufferDepth in seconds, +Inf if absent
	ATO         float64   // availabilityTimeOffset in seconds
	ATC         *bool     // effective availabilityTimeComplete, nil if absent
	Media       string    // effective SegmentTemplate@media

	LowLatency          bool                       // the MPD uses low-latency profile or chunked delivery
	TargetLatency       float64                    // ServiceDescription target latency in seconds, 0 if absent
	ContentProtections  []*m.ContentProtectionType // ContentProtection of the AdaptationSet and Representation, with @ref resolved
	AudioChannelConfigs []*m.DescriptorType        // effective AudioChannelConfiguration elements
	Essentials          []*m.DescriptorType        // EssentialProperty elements of the AdaptationSet and Representation
	Supplementals       []*m.DescriptorType        // SupplementalProperty elements of the AdaptationSet and Representation
	SegmentProfiles     string                     // effective @segmentProfiles

	tmpl  *m.SegmentTemplateType
	list  *m.SegmentListType
	repID string
}

// Loc returns a report location for this Representation.
func (r *Rep) Loc() report.Location {
	return report.Location{RepID: r.ID, Path: r.Path}
}

// SegLoc returns a report location for one segment.
func (r *Rep) SegLoc(s Segment) report.Location {
	return report.Location{RepID: r.ID, Path: r.Path, Segment: s.ID(r.Kind), URL: s.URL}
}

// Start returns the segment start in seconds relative to the Period start.
func (r *Rep) Start(s Segment) float64 {
	return (float64(s.Time) - float64(r.PTO)) / float64(r.Timescale)
}

// End returns the segment end in seconds relative to the Period start.
func (r *Rep) End(s Segment) float64 {
	return r.Start(s) + float64(s.Dur)/float64(r.Timescale)
}

// AvailabilityStart returns the wall-clock time when the segment becomes available.
// Only meaningful for dynamic MPDs.
func (r *Rep) AvailabilityStart(s Segment) time.Time {
	return r.AST.Add(seconds(r.PeriodStart + r.End(s) - r.ATO))
}

// AvailabilityEnd returns the wall-clock time when the segment leaves the
// time-shift buffer. Only meaningful for dynamic MPDs.
func (r *Rep) AvailabilityEnd(s Segment) time.Time {
	if math.IsInf(r.TSBD, 1) {
		return time.Unix(1<<40, 0)
	}
	return r.AST.Add(seconds(r.PeriodStart + r.End(s) + r.TSBD))
}

// PresentationTime returns the presentation time relative to the Period
// start corresponding to the wall-clock time now. Only meaningful for dynamic MPDs.
func (r *Rep) PresentationTime(now time.Time) float64 {
	return now.Sub(r.AST).Seconds() - r.PeriodStart
}

func seconds(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}

var (
	RuleBaseURL = report.Register(report.Rule{
		ID: "ADDR-BASEURL", Severity: report.Error,
		Desc: "BaseURL elements must be valid URLs",
	})
	RuleSingleSegment = report.Register(report.Rule{
		ID: "ADDR-SINGLE", Severity: report.Info,
		Desc: "Representations that consist of a single segment given only by BaseURL are not checked yet",
	})
	RuleNoAddressing = report.Register(report.Rule{
		ID: "ADDR-NONE", Severity: report.Error,
		Desc:    "Each Representation needs SegmentTemplate, SegmentList or SegmentBase addressing",
		SpecRef: "ISO/IEC 23009-1 5.3.9",
	})
	RuleTemplate = report.Register(report.Rule{
		ID: "ADDR-TEMPLATE", Severity: report.Error,
		Desc:    "SegmentTemplate and SegmentList must have a consistent set of identifiers and timing attributes",
		SpecRef: "ISO/IEC 23009-1 5.3.9.4",
	})
	RulePeriodTiming = report.Register(report.Rule{
		ID: "ADDR-PERIOD", Severity: report.Error,
		Desc:    "Period start and duration must be derivable",
		SpecRef: "ISO/IEC 23009-1 5.3.2",
	})
)

// Reps resolves all Representations in the MPD.
func Reps(md *m.MPD, mpdURL string) ([]*Rep, []report.Finding) {
	var reps []*Rep
	var findings []report.Finding

	base, err := url.Parse(mpdURL)
	if err != nil {
		findings = append(findings, RuleBaseURL.New(report.Location{URL: mpdURL}, "bad MPD URL: %v", err))
		return nil, findings
	}
	base, f := resolveBase(base, md.BaseURL, "MPD")
	findings = append(findings, f...)

	dynamic := md.GetType() == "dynamic"
	var ast time.Time
	if dynamic && md.AvailabilityStartTime != "" {
		ast, err = parseDateTime(string(md.AvailabilityStartTime))
		if err != nil {
			findings = append(findings, RulePeriodTiming.New(report.Location{Path: "MPD"},
				"bad availabilityStartTime %q: %v", md.AvailabilityStartTime, err))
		}
	}
	tsbd := math.Inf(1)
	if md.TimeShiftBufferDepth != nil {
		tsbd = time.Duration(*md.TimeShiftBufferDepth).Seconds()
	}

	starts, durs, f := periodTimes(md)
	findings = append(findings, f...)
	cpSources := ContentProtectionSources(md)

	for pi, p := range md.Periods {
		pPath := fmt.Sprintf("Period[%d]", pi)
		pBase, f := resolveBase(base, p.BaseURLs, pPath)
		findings = append(findings, f...)
		for ai, as := range p.AdaptationSets {
			asPath := fmt.Sprintf("%s/AdaptationSet[%d]", pPath, ai)
			asBase, f := resolveBase(pBase, as.BaseURLs, asPath)
			findings = append(findings, f...)
			for _, rep := range as.Representations {
				rPath := fmt.Sprintf("%s/Representation[%s]", asPath, rep.Id)
				rBase, f := resolveBase(asBase, rep.BaseURLs, rPath)
				findings = append(findings, f...)
				r := &Rep{
					ID:                  rep.Id,
					Path:                rPath,
					PeriodID:            p.Id,
					PeriodIndex:         pi,
					ASPath:              asPath,
					ASID:                optUint(as.Id),
					Lang:                as.Lang,
					ContainerProfiles:   containerProfiles(rep.ContainerProfiles, as.ContainerProfiles),
					Roles:               as.Roles,
					ASCodecs:            as.Codecs,
					TargetLatency:       targetLatency(md),
					ContentProtections:  ResolveContentProtections(slices.Concat(as.ContentProtections, rep.ContentProtections), cpSources),
					AudioChannelConfigs: firstElements(rep.AudioChannelConfigurations, as.AudioChannelConfigurations),
					Essentials:          slices.Concat(as.EssentialProperties, rep.EssentialProperties),
					Supplementals:       slices.Concat(as.SupplementalProperties, rep.SupplementalProperties),
					SequenceProperties:  firstElements(rep.SegmentSequenceProperties, as.SegmentSequenceProperties),
					SegmentProfiles:     first(listOf4CC(rep.SegmentProfiles), listOf4CC(as.SegmentProfiles)),
					Profiles:            splitProfiles(first(string(rep.Profiles), string(as.Profiles), string(md.Profiles))),
					BitstreamSwitching:  boolAttr(as.BitstreamSwitching, p.BitstreamSwitching),
					ContentType:         string(as.ContentType),
					MimeType:            first(rep.MimeType, as.MimeType),
					Codecs:              first(rep.Codecs, as.Codecs),
					Bandwidth:           rep.Bandwidth,
					Width:               firstU(rep.Width, as.Width),
					Height:              firstU(rep.Height, as.Height),
					MaxWidth:            as.MaxWidth,
					MaxHeight:           as.MaxHeight,
					FrameRate:           first(string(rep.FrameRate), string(as.FrameRate)),
					Sar:                 first(string(rep.Sar), string(as.Sar)),
					AudioSamplingRate:   firstVec(rep.AudioSamplingRate, as.AudioSamplingRate),
					StartWithSAP:        firstU(rep.StartWithSAP, as.StartWithSAP),
					BaseURL:             rBase.String(),
					Dynamic:             dynamic,
					AST:                 ast,
					PeriodStart:         starts[pi],
					PeriodDur:           durs[pi],
					LastPeriod:          pi == len(md.Periods)-1,
					TSBD:                tsbd,
					ATO:                 baseURLATO(md.BaseURL, p.BaseURLs, as.BaseURLs, rep.BaseURLs),
					repID:               rep.Id,
				}
				findings = append(findings, r.setAddressing(p, as, rep)...)
				reps = append(reps, r)
			}
		}
	}
	lowLatency := dynamic && slices.Contains(splitProfiles(string(md.Profiles)), ProfileLowLatency)
	for _, r := range reps {
		lowLatency = lowLatency || (dynamic && (r.ATO > 0 || (r.ATC != nil && !*r.ATC)))
	}
	for _, r := range reps {
		r.LowLatency = lowLatency
	}
	return reps, findings
}

// ProfileLowLatency is the DASH-IF low-latency profile.
const ProfileLowLatency = "http://www.dashif.org/guidelines/low-latency-live-v5"

func targetLatency(md *m.MPD) float64 {
	for _, sd := range md.ServiceDescription {
		for _, l := range sd.Latencies {
			if l.Target != nil {
				return float64(*l.Target) / 1000
			}
		}
	}
	return 0
}

// firstElements returns the first non-empty list of elements, lowest level first.
func firstElements[T any](levels ...[]T) []T {
	for _, l := range levels {
		if len(l) > 0 {
			return l
		}
	}
	return nil
}

func listOf4CC(v *m.ListOf4CCType) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func (r *Rep) setAddressing(p *m.Period, as *m.AdaptationSetType, rep *m.RepresentationType) []report.Finding {
	loc := r.Loc()
	switch {
	case p.SegmentTemplate != nil || as.SegmentTemplate != nil || rep.SegmentTemplate != nil:
		t := mergeTemplates(p.SegmentTemplate, as.SegmentTemplate, rep.SegmentTemplate)
		r.tmpl = t
		r.Timescale = 1
		if t.Timescale != nil {
			r.Timescale = *t.Timescale
		}
		if t.PresentationTimeOffset != nil {
			r.PTO = *t.PresentationTimeOffset
		}
		r.ATO += float64(t.AvailabilityTimeOffset)
		r.ATC = t.AvailabilityTimeComplete
		r.Media = t.Media
		hasTime := containsID(t.Media, "Time")
		hasNumber := containsID(t.Media, "Number")
		hasSub := containsID(t.Media, "SubNumber")
		r.SegmentSequences = hasSub || (t.K != nil && *t.K > 1) || timelineSequences(t.SegmentTimeline)
		switch {
		case t.Media == "":
			return []report.Finding{RuleTemplate.New(loc, "SegmentTemplate has no @media")}
		case hasTime && hasNumber:
			return []report.Finding{RuleTemplate.New(loc, "@media %q has both $Time$ and $Number$", t.Media)}
		case t.SegmentTimeline != nil && t.Duration != nil:
			return []report.Finding{RuleTemplate.New(loc, "SegmentTemplate has both @duration and SegmentTimeline")}
		case t.SegmentTimeline != nil && hasTime:
			r.Kind = KindTimelineTime
		case t.SegmentTimeline != nil && hasNumber:
			r.Kind = KindTimelineNumber
		case t.SegmentTimeline != nil:
			return []report.Finding{RuleTemplate.New(loc, "@media %q needs $Time$ or $Number$ with SegmentTimeline", t.Media)}
		case hasTime:
			return []report.Finding{RuleTemplate.New(loc, "$Time$ in @media %q requires a SegmentTimeline", t.Media)}
		case hasNumber && t.Duration != nil:
			if *t.Duration == 0 {
				return []report.Finding{RuleTemplate.New(loc, "SegmentTemplate@duration is 0")}
			}
			r.Kind = KindNumber
		case hasNumber:
			return []report.Finding{RuleTemplate.New(loc, "$Number$ without @duration or SegmentTimeline")}
		default:
			return []report.Finding{RuleTemplate.New(loc, "@media %q has neither $Number$ nor $Time$", t.Media)}
		}
		// Partial Segments are addressed with $Number$ alone, or with
		// $SubNumber$ (ISO/IEC 23009-1 5.3.9.7.1).
		if r.SegmentSequences && hasTime && !hasSub {
			r.Kind = KindUnknown
			return []report.Finding{RuleTemplate.New(loc, "Segment Sequences with $Time$ in @media %q need $SubNumber$", t.Media)}
		}
		if t.Initialization != "" {
			init, err := r.expand(t.Initialization, 0, 0, 0)
			if err != nil {
				return []report.Finding{RuleTemplate.New(loc, "@initialization: %v", err)}
			}
			r.InitURL = init
		} else if t.SegmentBaseType.Initialization != nil {
			r.InitURL = r.resolve(string(t.SegmentBaseType.Initialization.SourceURL))
			r.InitRange = t.SegmentBaseType.Initialization.Range
		}
		return nil
	case p.SegmentList != nil || as.SegmentList != nil || rep.SegmentList != nil:
		l := firstList(rep.SegmentList, as.SegmentList, p.SegmentList)
		r.list = l
		r.Kind = KindSegmentList
		r.SegmentSequences = (l.K != nil && *l.K > 1) || timelineSequences(l.SegmentTimeline)
		r.Timescale = 1
		if l.Timescale != nil {
			r.Timescale = *l.Timescale
		}
		if l.PresentationTimeOffset != nil {
			r.PTO = *l.PresentationTimeOffset
		}
		if l.Initialization != nil {
			r.InitURL = r.resolve(string(l.Initialization.SourceURL))
			r.InitRange = l.Initialization.Range
		}
		if l.Duration != nil && l.SegmentTimeline != nil {
			return []report.Finding{RuleTemplate.New(loc, "SegmentList has both @duration and SegmentTimeline")}
		}
		if l.Duration == nil && l.SegmentTimeline == nil && len(l.SegmentURL) > 1 {
			return []report.Finding{RuleTemplate.New(loc, "SegmentList with several segments needs @duration or SegmentTimeline")}
		}
		return nil
	case p.SegmentBase != nil || as.SegmentBase != nil || rep.SegmentBase != nil:
		b := firstBase(rep.SegmentBase, as.SegmentBase, p.SegmentBase)
		r.Kind = KindSegmentBase
		r.Timescale = 1
		if b.Timescale != nil {
			r.Timescale = *b.Timescale
		}
		if b.PresentationTimeOffset != nil {
			r.PTO = *b.PresentationTimeOffset
		}
		r.InitURL = r.BaseURL
		if b.Initialization != nil {
			if b.Initialization.SourceURL != "" {
				r.InitURL = r.resolve(string(b.Initialization.SourceURL))
			}
			r.InitRange = b.Initialization.Range
		}
		r.IndexRange = b.IndexRange
		if r.IndexRange == "" {
			return []report.Finding{RuleTemplate.New(loc, "SegmentBase without @indexRange is not supported")}
		}
		return nil
	case len(rep.BaseURLs)+len(as.BaseURLs)+len(p.BaseURLs) > 0:
		return []report.Finding{RuleSingleSegment.New(loc, "single segment at %s", r.BaseURL)}
	default:
		return []report.Finding{RuleNoAddressing.New(loc, "no SegmentTemplate, SegmentList or SegmentBase")}
	}
}

// Segments returns the segments that overlap the interval [from, to) in
// seconds relative to the Period start. to may be +Inf for a bounded Period. It is not used for SegmentBase,
// whose segments come from the sidx box.
func (r *Rep) Segments(from, to float64) ([]Segment, error) {
	if r.durationPatterns() {
		return nil, ErrDurationPatterns
	}
	switch r.Kind {
	case KindNumber:
		return r.numberSegments(from, to)
	case KindTimelineTime, KindTimelineNumber:
		return r.timelineSegments(r.tmpl.SegmentTimeline, r.startNumber(), from, to)
	case KindSegmentList:
		return r.listSegments(from, to)
	default:
		return nil, fmt.Errorf("segment list not available for %s addressing", r.Kind)
	}
}

// MaxSegments limits the number of segments enumerated in one call.
const MaxSegments = 1_000_000

func (r *Rep) startNumber() uint64 {
	if r.tmpl != nil && r.tmpl.StartNumber != nil {
		return uint64(*r.tmpl.StartNumber)
	}
	if r.list != nil && r.list.StartNumber != nil {
		return uint64(*r.list.StartNumber)
	}
	return 1
}

func (r *Rep) numberSegments(from, to float64) ([]Segment, error) {
	ts := float64(r.Timescale)
	dur := uint64(*r.tmpl.Duration) // of a segment or a Segment Sequence
	k := partials(r.tmpl.K)
	sub := containsID(r.tmpl.Media, "SubNumber")
	if r.PeriodDur >= 0 && to > r.PeriodDur {
		to = r.PeriodDur
	}
	if math.IsInf(to, 1) {
		return nil, fmt.Errorf("open-ended interval for number-based addressing")
	}
	if from < 0 {
		from = 0
	}
	qFirst := uint64(math.Floor(from * ts / float64(dur)))
	qEnd := uint64(math.Ceil(to * ts / float64(dur))) // exclusive
	if (qEnd-min(qEnd, qFirst))*k > MaxSegments {
		return nil, fmt.Errorf("more than %d segments", MaxSegments)
	}
	start := r.startNumber()
	var segs []Segment
	for q := qFirst; q < qEnd; q++ {
		for j := range k {
			// With $SubNumber$, $Number$ counts Segment Sequences, otherwise
			// Partial Segments (ISO/IEC 23009-1 5.3.9.7).
			nr := start + q*k + j
			if sub {
				nr = start + q
			}
			if r.afterEnd(nr, j, sub) {
				return segs, nil
			}
			ps, pe := q*dur+j*dur/k, q*dur+(j+1)*dur/k
			if float64(ps) >= to*ts {
				return segs, nil
			}
			if float64(pe) <= from*ts {
				continue
			}
			s := r.segment(nr, j, sub, r.PTO+ps, pe-ps)
			u, err := r.expand(r.tmpl.Media, s.Number, s.SubNumber, s.Time)
			if err != nil {
				return nil, err
			}
			s.URL = u
			segs = append(segs, s)
		}
	}
	return segs, nil
}

// partials returns the number of Partial Segments in a Segment Sequence
// from @k, 1 if absent.
func partials(k *uint64) uint64 {
	if k != nil && *k > 1 {
		return *k
	}
	return 1
}

// afterEnd reports whether segment number nr, with sub-number j when sub is
// set, comes after @endNumber and @endSubNumber.
func (r *Rep) afterEnd(nr, j uint64, sub bool) bool {
	t := r.tmpl
	if t == nil || t.EndNumber == nil {
		return false
	}
	end := uint64(*t.EndNumber)
	return nr > end || (sub && nr == end && t.EndSubNumber != nil && j > *t.EndSubNumber)
}

// segment returns a segment, as a Partial Segment if the Representation uses
// Segment Sequences.
func (r *Rep) segment(nr, j uint64, sub bool, t, dur uint64) Segment {
	s := Segment{Number: nr, Time: t, Dur: dur}
	if r.SegmentSequences {
		s.Partial, s.SubNumber, s.sub = true, j, sub
	}
	return s
}

func (r *Rep) timelineSegments(stl *m.SegmentTimelineType, startNr uint64, from, to float64) ([]Segment, error) {
	ts := float64(r.Timescale)
	endTicks := uint64(math.MaxUint64)
	if r.PeriodDur >= 0 {
		endTicks = r.PTO + uint64(math.Round(r.PeriodDur*ts))
	}
	fromTicks := float64(r.PTO) + from*ts
	toTicks := float64(r.PTO) + to*ts
	var segs []Segment
	var t uint64
	nr := startNr
	sub := r.tmpl != nil && containsID(r.tmpl.Media, "SubNumber")
	for i, s := range stl.S {
		if s.T != nil {
			t = *s.T
		}
		if s.D == 0 {
			return nil, fmt.Errorf("SegmentTimeline S[%d] has d=0", i)
		}
		repeat := uint64(s.R) + 1
		if s.R < 0 {
			// Repeat until the next S@t, the Period end, or the end of the interval.
			limit := endTicks
			if i+1 < len(stl.S) && stl.S[i+1].T != nil {
				limit = *stl.S[i+1].T
			}
			if limit == math.MaxUint64 {
				if math.IsInf(to, 1) {
					return nil, fmt.Errorf("SegmentTimeline S[%d] has negative @r in an open-ended Period", i)
				}
				limit = uint64(math.Ceil(toTicks))
			}
			if limit <= t {
				return nil, fmt.Errorf("SegmentTimeline S[%d] has negative @r but no room until %d", i, limit)
			}
			repeat = (limit - t + s.D - 1) / s.D
		}
		// With @k, each S describes Segment Sequences of k Partial Segments.
		k := partials(s.K)
		for range repeat {
			for j := range k {
				ps, pe := t+j*s.D/k, t+(j+1)*s.D/k
				if float64(ps) >= toTicks {
					return segs, nil
				}
				if float64(pe) <= fromTicks {
					continue
				}
				seg := r.segment(nr+j, j, sub, ps, pe-ps)
				if sub {
					seg.Number = nr
				}
				if r.list == nil {
					// With $SubNumber$, $Time$ is the start of the Segment Sequence.
					ut := ps
					if sub {
						ut = t
					}
					u, err := r.expand(r.tmpl.Media, seg.Number, seg.SubNumber, ut)
					if err != nil {
						return nil, err
					}
					seg.URL = u
				}
				segs = append(segs, seg)
				if len(segs) > MaxSegments {
					return nil, fmt.Errorf("more than %d segments", MaxSegments)
				}
			}
			t += s.D
			if sub {
				nr++
			} else {
				nr += k
			}
		}
	}
	return segs, nil
}

func (r *Rep) listSegments(from, to float64) ([]Segment, error) {
	l := r.list
	var segs []Segment
	if l.SegmentTimeline != nil {
		var err error
		segs, err = r.timelineSegments(l.SegmentTimeline, r.startNumber(), math.Inf(-1), math.Inf(1))
		if err != nil {
			return nil, err
		}
	} else {
		dur := uint64(0)
		if l.Duration != nil {
			dur = uint64(*l.Duration)
		}
		// With @k, each SegmentURL is a Partial Segment and @duration is that
		// of a Segment Sequence.
		k := partials(l.K)
		for i := range uint64(len(l.SegmentURL)) {
			q, j := i/k, i%k
			ps, pe := q*dur+j*dur/k, q*dur+(j+1)*dur/k
			segs = append(segs, r.segment(r.startNumber()+i, j, false, r.PTO+ps, pe-ps))
		}
	}
	var out []Segment
	for i, s := range segs {
		if i >= len(l.SegmentURL) {
			break
		}
		su := l.SegmentURL[i]
		s.URL = r.BaseURL
		if su.Media != "" {
			s.URL = r.resolve(string(su.Media))
		}
		s.Range = string(su.MediaRange)
		if r.End(s) > from && r.Start(s) < to {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *Rep) resolve(ref string) string {
	base, err := url.Parse(r.BaseURL)
	if err != nil {
		return ref
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return base.ResolveReference(u).String()
}

func (r *Rep) expand(tmpl string, number, sub, t uint64) (string, error) {
	s, err := Substitute(tmpl, Values{RepresentationID: r.repID, Number: number, SubNumber: sub, Time: t, Bandwidth: r.Bandwidth})
	if err != nil {
		return "", err
	}
	return r.resolve(s), nil
}

func resolveBase(base *url.URL, bases []*m.BaseURLType, path string) (*url.URL, []report.Finding) {
	if len(bases) == 0 {
		return base, nil
	}
	ref, err := url.Parse(string(bases[0].Value))
	if err != nil {
		return base, []report.Finding{RuleBaseURL.New(report.Location{Path: path + "/BaseURL"}, "bad BaseURL %q: %v", bases[0].Value, err)}
	}
	return base.ResolveReference(ref), nil
}

func baseURLATO(levels ...[]*m.BaseURLType) float64 {
	ato := 0.0
	for _, l := range levels {
		if len(l) > 0 && l[0].AvailabilityTimeOffset != 0 {
			ato = float64(l[0].AvailabilityTimeOffset)
		}
	}
	return ato
}

// periodTimes returns start and duration in seconds for each Period.
// Duration is -1 for an open-ended last Period.
func periodTimes(md *m.MPD) (starts, durs []float64, findings []report.Finding) {
	n := len(md.Periods)
	starts = make([]float64, n)
	durs = make([]float64, n)
	for i, p := range md.Periods {
		switch {
		case p.Start != nil:
			starts[i] = time.Duration(*p.Start).Seconds()
		case i == 0:
			starts[i] = 0
		case md.Periods[i-1].Duration != nil:
			starts[i] = starts[i-1] + time.Duration(*md.Periods[i-1].Duration).Seconds()
		default:
			findings = append(findings, RulePeriodTiming.New(report.Location{Path: fmt.Sprintf("Period[%d]", i)},
				"no @start and previous Period has no @duration"))
			starts[i] = starts[i-1]
		}
	}
	for i, p := range md.Periods {
		switch {
		case p.Duration != nil:
			durs[i] = time.Duration(*p.Duration).Seconds()
		case i+1 < n:
			durs[i] = starts[i+1] - starts[i]
		case md.MediaPresentationDuration != nil:
			durs[i] = time.Duration(*md.MediaPresentationDuration).Seconds() - starts[i]
		default:
			durs[i] = -1 // open-ended; mpdcheck reports static MPDs without a known end
		}
	}
	return starts, durs, findings
}

func parseDateTime(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05.999999999Z0700"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("not an xs:dateTime")
}

// ParseDateTime parses an xs:dateTime as used in MPDs. A missing time zone means UTC.
func ParseDateTime(s string) (time.Time, error) { return parseDateTime(s) }

// HasProfile reports whether the Representation's effective @profiles include profile.
func (r *Rep) HasProfile(profile string) bool {
	for _, p := range r.Profiles {
		if p == profile {
			return true
		}
	}
	return false
}

// UseOnlyProfiles makes only the profiles in Profiles turn on the
// profile-specific rules, for when the user chose the profiles to check
// (spec.Selection.Only). Chunking then does not make the Representation low
// latency, and CMAF brands do not make it CMAF.
func (r *Rep) UseOnlyProfiles() {
	r.OnlyProfiles = true
	r.LowLatency = r.LowLatency && r.HasProfile(ProfileLowLatency)
}

// HasProfilePrefix reports whether any effective profile starts with prefix.
func (r *Rep) HasProfilePrefix(prefix string) bool {
	for _, p := range r.Profiles {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

func splitProfiles(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// boolAttr returns the first set value of an inherited boolean attribute.
func boolAttr(levels ...*bool) bool {
	for _, b := range levels {
		if b != nil {
			return *b
		}
	}
	return false
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstU(vals ...uint32) uint32 {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}

func firstVec(vals ...*m.UIntVectorType) string {
	for _, v := range vals {
		if v != nil && *v != "" {
			return string(*v)
		}
	}
	return ""
}

func firstList(vals ...*m.SegmentListType) *m.SegmentListType {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

func firstBase(vals ...*m.SegmentBaseType) *m.SegmentBaseType {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

// mergeTemplates applies SegmentTemplate inheritance: attributes on a lower
// level override those on a higher level.
func mergeTemplates(levels ...*m.SegmentTemplateType) *m.SegmentTemplateType {
	out := &m.SegmentTemplateType{}
	for _, t := range levels {
		if t == nil {
			continue
		}
		if t.Media != "" {
			out.Media = t.Media
		}
		if t.Index != "" {
			out.Index = t.Index
		}
		if t.Initialization != "" {
			out.Initialization = t.Initialization
		}
		if t.Duration != nil {
			out.Duration = t.Duration
		}
		if t.StartNumber != nil {
			out.StartNumber = t.StartNumber
		}
		if t.EndNumber != nil {
			out.EndNumber = t.EndNumber
		}
		if t.EndSubNumber != nil {
			out.EndSubNumber = t.EndSubNumber
		}
		if t.K != nil {
			out.K = t.K
		}
		if t.SegmentTimeline != nil {
			out.SegmentTimeline = t.SegmentTimeline
		}
		if t.Timescale != nil {
			out.Timescale = t.Timescale
		}
		if t.PresentationTimeOffset != nil {
			out.PresentationTimeOffset = t.PresentationTimeOffset
		}
		if t.AvailabilityTimeOffset != 0 {
			out.AvailabilityTimeOffset = t.AvailabilityTimeOffset
		}
		if t.AvailabilityTimeComplete != nil {
			out.AvailabilityTimeComplete = t.AvailabilityTimeComplete
		}
		if t.SegmentBaseType.Initialization != nil {
			out.SegmentBaseType.Initialization = t.SegmentBaseType.Initialization
		}
	}
	return out
}

// optUint formats an optional unsigned attribute, "" if absent.
func optUint(v *uint32) string {
	if v == nil {
		return ""
	}
	return strconv.FormatUint(uint64(*v), 10)
}

// containerProfiles returns the Representation's @containerProfiles, or else
// the AdaptationSet's.
func containerProfiles(rep, as *m.ListOf4CCType) []string {
	for _, l := range []*m.ListOf4CCType{rep, as} {
		if l != nil {
			return strings.Fields(string(*l))
		}
	}
	return nil
}

// timelineSequences reports whether a SegmentTimeline has Segment Sequences,
// with S@k or a Segment Sequence duration pattern (S@ssp).
func timelineSequences(stl *m.SegmentTimelineType) bool {
	return stl != nil && slices.ContainsFunc(stl.S, func(s *m.S) bool { return (s.K != nil && *s.K > 1) || s.Ssp != nil })
}

// durationPatterns reports whether the SegmentTimeline uses duration
// patterns (S@p or S@ssp), which are not supported.
func (r *Rep) durationPatterns() bool {
	var stl *m.SegmentTimelineType
	switch {
	case r.tmpl != nil:
		stl = r.tmpl.SegmentTimeline
	case r.list != nil:
		stl = r.list.SegmentTimeline
	}
	return stl != nil && slices.ContainsFunc(stl.S, func(s *m.S) bool { return s.P != nil || s.Ssp != nil })
}
