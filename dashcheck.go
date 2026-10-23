// Package dashcheck checks MPEG-DASH streams: the MPD, its segments, and for
// dynamic MPDs the timing against a synchronized wall clock.
//
// All network access goes through a fetch.Fetcher and all time through a
// clock.Clock, so the same code runs natively and in the browser (wasm).
package dashcheck

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/clock"
	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/mpdcheck"
	"github.com/tobbee/dashcheck/report"
	"github.com/tobbee/dashcheck/segcheck"
	"github.com/tobbee/dashcheck/spec"
)

// Options control a check.
type Options struct {
	Fetcher fetch.Fetcher
	// Clock overrides clock synchronization. If nil, dynamic MPDs use a clock
	// synced to their UTCTiming sources, and static MPDs the local clock.
	Clock clock.Clock
	// Reference is the reference clock URL (http-iso). Empty means
	// clock.DefaultReference; "none" disables it.
	Reference string
	// MaxSegments is the number of media segments checked per Representation.
	// Zero means DefaultMaxSegments; negative means all (static MPDs only).
	MaxSegments int
	// Concurrency is the number of Representations checked in parallel.
	Concurrency int
	// LiveMargin keeps live checks away from the edges of the availability
	// window, to allow for clock inaccuracy. Zero means DefaultLiveMargin.
	LiveMargin time.Duration
	// MPDUpdates is the number of times a dynamic MPD is fetched again to
	// check its updates. Zero means DefaultMPDUpdates; negative disables it.
	MPDUpdates int
	// Profiles chooses the profiles whose rules apply. The zero value is the
	// profiles that the MPD signals in @profiles.
	Profiles spec.Selection
	// HLSInterop adds the checks of the DASH-HLS Interoperability
	// Specification CTA-5005-B, which MPDs do not signal.
	HLSInterop bool
	// SequenceNumbers adds the check of a sequence number convention: one
	// mfhd sequence_number per segment, increasing (SEG-SEQUENCE-NUMBER).
	SequenceNumbers bool
	// UpdateInterval is the time between MPD fetches. Zero means about one
	// segment duration, between 2 and 10 seconds.
	UpdateInterval time.Duration
	// Progress, if set, receives progress updates. It may be called from several goroutines.
	Progress func(Progress)
}

// Progress describes how far a check has come.
type Progress struct {
	Phase   string `json:"phase"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Message string `json:"message,omitempty"`
}

const (
	DefaultMaxSegments = 4
	DefaultConcurrency = 4
	DefaultLiveMargin  = time.Second
	DefaultMPDUpdates  = 1
)

var (
	RuleHTTPMPD = report.Register(report.Rule{
		ID: "HTTP-MPD", Severity: report.Error,
		Desc: "The MPD must be fetchable",
	})
	RuleHTTPInit = report.Register(report.Rule{
		ID: "HTTP-INIT", Severity: report.Error,
		Desc: "The initialization segment must be fetchable",
	})
	RuleHTTPSegment = report.Register(report.Rule{
		ID: "HTTP-SEGMENT", Severity: report.Error,
		Desc: "Media segments must be fetchable while they are in the availability window",
	})
	RuleUnsupported = report.Register(report.Rule{
		ID: "SEG-UNSUPPORTED", Severity: report.Info,
		Desc: "Segments in formats other than ISOBMFF are not checked",
	})
	RuleSegmentSequences = report.Register(report.Rule{
		ID: "ADDR-UNSUPPORTED", Severity: report.Info,
		Desc: "SegmentTimeline duration patterns (S@p, S@ssp) and URL parameters (Annex I) are not supported yet, so their segments are not checked",
	})
	RuleSegmentList = report.Register(report.Rule{
		ID: "ADDR-SEGMENTS", Severity: report.Error,
		Desc: "The segment list of a Representation must be derivable",
	})
)

func (o *Options) defaults() {
	if o.MaxSegments == 0 {
		o.MaxSegments = DefaultMaxSegments
	}
	if o.Concurrency <= 0 {
		o.Concurrency = DefaultConcurrency
	}
	if o.LiveMargin == 0 {
		o.LiveMargin = DefaultLiveMargin
	}
	if o.MPDUpdates == 0 {
		o.MPDUpdates = DefaultMPDUpdates
	}
	if o.Reference == "" {
		o.Reference = clock.DefaultReference
	}
	if o.Reference == "none" {
		o.Reference = ""
	}
}

func (o *Options) progress(phase string, done, total int, format string, args ...any) {
	if o.Progress != nil {
		o.Progress(Progress{Phase: phase, Done: done, Total: total, Message: fmt.Sprintf(format, args...)})
	}
}

// Check fetches and checks the MPD at mpdURL and its segments.
func Check(ctx context.Context, mpdURL string, opts Options) *report.Report {
	opts.defaults()
	rpt := &report.Report{MPDURL: mpdURL, Started: time.Now()}
	defer func() { rpt.Duration = time.Since(rpt.Started).Seconds() }()

	opts.progress("mpd", 0, 1, "fetching %s", mpdURL)
	resp, err := fetch.Get(ctx, opts.Fetcher, mpdURL)
	if err != nil {
		rpt.Findings = append(rpt.Findings, RuleHTTPMPD.New(report.Location{URL: mpdURL}, "%v", err))
		return rpt
	}
	if md, clk := checkMPD(ctx, rpt, resp.Body, resp, &opts); md != nil && rpt.Type == "dynamic" {
		checkUpdates(ctx, rpt, mpdURL, md, resp, clk, &opts)
	}
	return rpt
}

// CheckData checks an MPD given as data, e.g. a local file opened in a browser.
// Segments are fetched relative to mpdURL.
func CheckData(ctx context.Context, data []byte, mpdURL string, opts Options) *report.Report {
	opts.defaults()
	rpt := &report.Report{MPDURL: mpdURL, Started: time.Now()}
	defer func() { rpt.Duration = time.Since(rpt.Started).Seconds() }()
	now := time.Now()
	resp := &fetch.Response{URL: mpdURL, Status: 200, Body: data, Start: now, End: now}
	checkMPD(ctx, rpt, data, resp, &opts)
	return rpt
}

// checkMPD checks an MPD and its segments, and returns the parsed MPD and
// the clock used.
func checkMPD(ctx context.Context, rpt *report.Report, data []byte, resp *fetch.Response, opts *Options) (*m.MPD, clock.Clock) {
	opts.progress("mpd", 0, 1, "checking MPD")
	md, findings := mpdcheck.CheckWith(data, resp.URL, opts.Profiles)
	rpt.Findings = append(rpt.Findings, findings...)
	if md == nil {
		return nil, nil
	}
	rpt.Type = md.GetType()
	rpt.ProfileChoice = opts.Profiles.String()
	for _, p := range spec.Signalled(md) {
		rpt.Profiles = append(rpt.Profiles, p.Title)
	}
	dynamic := rpt.Type == "dynamic"

	clk := opts.Clock
	switch {
	case clk != nil:
	case dynamic:
		opts.progress("clock", 0, 1, "synchronizing clock")
		var source string
		var cf []report.Finding
		clk, source, cf = clock.Sync(ctx, opts.Fetcher, md.UTCTimings, opts.Reference, resp)
		rpt.Findings = append(rpt.Findings, cf...)
		rpt.ClockSource = source
		if o, ok := clk.(*clock.Offset); ok {
			rpt.ClockOffset = o.Offset.Seconds()
		}
	default:
		clk = clock.System{}
	}

	// Relative URLs resolve against the MPD URL after redirects.
	reps, af := addressing.Reps(md, resp.URL)
	rpt.Findings = append(rpt.Findings, af...)
	if opts.Profiles.Only {
		for _, r := range reps {
			r.UseOnlyProfiles()
		}
	}
	now := clk.Now()
	if dynamic {
		rpt.Findings = append(rpt.Findings, liveMPDChecks(md, reps, now)...)
	}

	results := make([][]report.Finding, len(reps))
	counts := make([]int, len(reps))
	inits := make([]*segcheck.Init, len(reps))
	firstEPTs := make([]float64, len(reps))
	var wg sync.WaitGroup
	sem := make(chan struct{}, opts.Concurrency)
	var mu sync.Mutex
	done := 0
	for i, rep := range reps {
		if rep.Kind == addressing.KindUnknown {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rc := &repChecker{rep: rep, opts: opts, clk: clk, dynamic: dynamic, firstEPT: math.NaN()}
			results[i], counts[i] = rc.run(ctx)
			inits[i], firstEPTs[i] = rc.init, rc.firstEPT
			mu.Lock()
			done++
			opts.progress("segments", done, len(reps), "checked %s", rep.ID)
			mu.Unlock()
		}()
	}
	wg.Wait()
	for i := range reps {
		rpt.Findings = append(rpt.Findings, results[i]...)
		rpt.Segments += counts[i]
	}
	// Checks across the Representations of each AdaptationSet.
	var asReps []*addressing.Rep
	var asInits []*segcheck.Init
	for i, rep := range reps {
		if len(asReps) > 0 && asReps[0].ASPath != rep.ASPath {
			rpt.Findings = append(rpt.Findings, segcheck.CheckAdaptationSet(asReps, asInits)...)
			asReps, asInits = nil, nil
		}
		asReps = append(asReps, rep)
		asInits = append(asInits, inits[i])
	}
	rpt.Findings = append(rpt.Findings, segcheck.CheckAdaptationSet(asReps, asInits)...)
	rpt.Findings = append(rpt.Findings, segcheck.CheckPresentation(reps, inits, firstEPTs)...)
	return md, clk
}

// repChecker checks the segments of one Representation.
type repChecker struct {
	rep      *addressing.Rep
	opts     *Options
	clk      clock.Clock
	dynamic  bool
	findings []report.Finding
	checked  int
	init     *segcheck.Init
	sidx     *mp4.SidxBox // index of a SegmentBase Representation
	firstEPT float64      // presentation time of the first segment of the Period, NaN if not checked
	nSidx    int          // number of sidx boxes in its index range
	// Byte positions in the file of a SegmentBase Representation: between the
	// sidx box and the first subsegment, and after the last subsegment.
	gapStart, gapEnd, indexEnd uint64
}

func (rc *repChecker) add(f ...report.Finding) { rc.findings = append(rc.findings, f...) }

func (rc *repChecker) run(ctx context.Context) ([]report.Finding, int) {
	rep := rc.rep
	if !isISOBMFF(rep.MimeType) {
		rc.add(RuleUnsupported.New(rep.Loc(), "segments with @mimeType %q are not checked", rep.MimeType))
		return rc.findings, rc.checked
	}
	// A client that does not support an EssentialProperty ignores its element
	// (ISO/IEC 23009-1 5.8.4.8), so segments without the parameters may fail.
	if i := slices.IndexFunc(rep.Essentials, isURLParams); i >= 0 {
		rc.add(RuleSegmentSequences.New(rep.Loc(), "URL parameters (%s, ISO/IEC 23009-1 Annex I) are not supported, so the segments are not checked",
			rep.Essentials[i].SchemeIdUri))
		return rc.findings, rc.checked
	}
	var segs []addressing.Segment
	var last addressing.Segment // last segment of the Period, for duration tolerance
	var err error
	if rep.Kind == addressing.KindSegmentBase {
		segs, err = rc.sidxSegments(ctx)
		if len(segs) > 0 {
			last = segs[len(segs)-1]
		}
	} else if rc.dynamic {
		segs, err = rc.liveSegments()
	} else {
		segs, err = rep.Segments(0, rep.PeriodDur)
		if len(segs) > 0 {
			last = segs[len(segs)-1]
		}
	}
	if err != nil {
		if errors.Is(err, addressing.ErrDurationPatterns) {
			rc.add(RuleSegmentSequences.New(rep.Loc(), "%v", err))
		} else {
			rc.add(RuleSegmentList.New(rep.Loc(), "%v", err))
		}
		return rc.findings, rc.checked
	}
	init := rc.checkInit(ctx)
	if init == nil {
		return rc.findings, rc.checked
	}
	rc.init = init
	var periodFirst addressing.Segment
	if !rc.dynamic && len(segs) > 0 {
		periodFirst = segs[0]
	}
	nAll := len(segs)
	if rep.Kind == addressing.KindSegmentBase {
		segs = rc.selectStatic(segs)
	} else if rc.dynamic {
		segs = rc.selectLive(segs)
	} else {
		segs = rc.selectStatic(segs)
	}
	var prevSeg addressing.Segment
	var prev, first *segcheck.Media
	// Segment sizes for SEG-HLS-BITRATE, while every segment is checked.
	// A CMAF fragment is a segment here, but a whole Segment Sequence with Partial Segments.
	hlsBitrate := rc.opts.HLSInterop && rep.Kind != addressing.KindSegmentBase && !rep.SegmentSequences && init.Timescale > 0
	var frags []segcheck.Fragment
	for i, seg := range segs {
		media := rc.checkSegment(ctx, init, seg, seg == last)
		if rc.opts.HLSInterop && seg == last && !rc.dynamic {
			rc.add(segcheck.CheckHLSInteropSplice(rep, init, media, seg)...)
		}
		if media == nil {
			hlsBitrate = false
		} else {
			frags = append(frags, segcheck.Fragment{Size: float64(media.Size), Dur: float64(media.Duration) / float64(init.Timescale)})
		}
		if i == 0 {
			first = media
			if media != nil && seg == periodFirst && rep.Timescale > 0 {
				rc.firstEPT = media.EPT - float64(rep.PTO)/float64(rep.Timescale)
			}
		}
		if prev != nil && media != nil && consecutive(prevSeg, seg) {
			rc.add(segcheck.CheckContinuity(prev, media, init, rep, seg)...)
		}
		prev, prevSeg = media, seg
	}
	if hlsBitrate && len(segs) > 0 {
		// On demand all segments of the Period must be checked; live, those
		// from the start of the Period or of the newest hour.
		start := segs[0]
		fromStart := len(segs) == nAll
		if rc.dynamic {
			fromStart = rep.Start(start) < float64(start.Dur)/float64(rep.Timescale)
		}
		rc.add(segcheck.CheckHLSBitrate(rep, frags, fromStart)...)
	}
	if rc.sidx != nil {
		// The index range precedes the media by construction of the sidx offsets.
		rc.add(segcheck.CheckSelfInitSidx(rep, init, rc.sidx, rc.nSidx, true, first)...)
		rc.add(segcheck.CheckDVBSidx(rep, rc.sidx, rc.nSidx)...)
		rc.checkSidxSpan(ctx)
		if rc.opts.HLSInterop {
			rc.add(segcheck.CheckHLSInteropSidx(rep, rc.sidx, rc.nSidx)...)
		}
	}
	if rc.dynamic && len(segs) > 0 {
		rc.probeEarly(ctx, segs[len(segs)-1])
	}
	return report.Collapse(rc.findings), rc.checked
}

// isISOBMFF reports whether the media type is ISOBMFF based (mp4).
// isURLParams reports whether a descriptor asks for the URL parameters of
// ISO/IEC 23009-1 Annex I, the baseline (I.2) or the extended (I.3) scheme.
func isURLParams(d *m.DescriptorType) bool {
	return d.SchemeIdUri == "urn:mpeg:dash:urlparam:2014" || d.SchemeIdUri == "urn:mpeg:dash:urlparam:2016"
}

func isISOBMFF(mimeType string) bool {
	t, _, _ := strings.Cut(mimeType, ";")
	switch strings.TrimSpace(t) {
	case "video/mp4", "audio/mp4", "application/mp4", "":
		return true
	}
	return false
}

func consecutive(a, b addressing.Segment) bool {
	if a.Time+a.Dur == b.Time {
		return true
	}
	// Partial Segments of a Segment Sequence can share a number.
	return !a.Partial && a.Number != 0 && a.Number+1 == b.Number
}

// selectStatic picks the first MaxSegments-1 segments and the last one,
// so that both continuity at the start and the end of the Period are covered.
func (rc *repChecker) selectStatic(segs []addressing.Segment) []addressing.Segment {
	n := rc.opts.MaxSegments
	if n < 0 || len(segs) <= n {
		return segs
	}
	out := append([]addressing.Segment{}, segs[:n-1]...)
	return append(out, segs[len(segs)-1])
}

func (rc *repChecker) checkInit(ctx context.Context) *segcheck.Init {
	rep := rc.rep
	loc := rep.Loc()
	loc.URL = rep.InitURL
	if rep.InitURL == "" {
		rc.add(RuleHTTPInit.New(loc, "no initialization segment signaled"))
		return nil
	}
	resp, err := rc.opts.Fetcher.Fetch(ctx, fetch.Request{URL: rep.InitURL, Range: rep.InitRange})
	if err != nil {
		rc.add(RuleHTTPInit.New(loc, "%v", err))
		return nil
	}
	if !resp.OK() {
		rc.add(RuleHTTPInit.New(loc, "status %d", resp.Status))
		return nil
	}
	init, f := segcheck.CheckInit(resp.Body, rep, loc)
	rc.add(f...)
	if init != nil && rc.opts.SequenceNumbers {
		init.SequenceNumbers = true
	}
	if init != nil && rc.opts.HLSInterop {
		init.HLSInterop = true
		rc.add(segcheck.CheckHLSInteropInit(rep, init, loc)...)
	}
	return init
}

func (rc *repChecker) checkSegment(ctx context.Context, init *segcheck.Init, seg addressing.Segment, last bool) *segcheck.Media {
	rep := rc.rep
	resp, err := rc.opts.Fetcher.Fetch(ctx, fetch.Request{URL: seg.URL, Range: seg.Range})
	if err != nil {
		rc.add(RuleHTTPSegment.New(rep.SegLoc(seg), "%v", err))
		return nil
	}
	if !resp.OK() {
		msg := fmt.Sprintf("status %d", resp.Status)
		if rc.dynamic {
			avail := rep.AvailabilityStart(seg)
			msg += fmt.Sprintf("; available since %s (%.1fs before the request)",
				avail.UTC().Format(time.RFC3339Nano), rc.clk.Now().Sub(avail).Seconds())
		}
		rc.add(RuleHTTPSegment.New(rep.SegLoc(seg), "%s", msg))
		return nil
	}
	rc.checked++
	media, f := segcheck.CheckMedia(resp.Body, init, rep, seg, last)
	rc.add(f...)
	return media
}
