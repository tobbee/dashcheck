package dashcheck

import (
	"context"
	"fmt"
	"math"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/report"
)

const refTimingModel = "DASH-IF Timing Model"

var (
	RuleNotStarted = report.Register(report.Rule{
		ID: "LIVE-NOT-STARTED", Severity: report.Warning,
		Desc: "The presentation has not started yet, so no segments can be checked",
	})
	RulePublishFuture = report.Register(report.Rule{
		ID: "LIVE-PUBLISHTIME", Severity: report.Warning,
		Desc: "MPD@publishTime should not be in the future",
	})
	RuleNoSegments = report.Register(report.Rule{
		ID: "LIVE-NO-SEGMENTS", Severity: report.Error,
		Desc:    "A live Representation must have segments in the availability window",
		SpecRef: refTimingModel + " 13.3",
	})
	RuleStaleTimeline = report.Register(report.Rule{
		ID: "LIVE-STALE", Severity: report.Error,
		Desc:    "The MPD must describe segments up to the live edge",
		SpecRef: refTimingModel + " 9.2.2",
	})
	RuleEarly = report.Register(report.Rule{
		ID: "LIVE-EARLY", Severity: report.Warning,
		Desc:    "Segments should not be available before their availability start time",
		SpecRef: refTimingModel + " 13.3",
	})
)

// liveMPDChecks checks MPD-level timing against the wall clock.
func liveMPDChecks(md *m.MPD, reps []*addressing.Rep, now time.Time) []report.Finding {
	var f []report.Finding
	loc := report.Location{Path: "MPD"}
	if md.PublishTime != "" {
		if pt, err := addressing.ParseDateTime(string(md.PublishTime)); err == nil && pt.After(now.Add(time.Second)) {
			f = append(f, RulePublishFuture.New(loc, "@publishTime %s is %.1fs in the future", md.PublishTime, pt.Sub(now).Seconds()))
		}
	}
	if len(reps) > 0 && !reps[0].AST.IsZero() {
		start := reps[0].AST.Add(time.Duration(reps[0].PeriodStart * float64(time.Second)))
		if start.After(now) {
			f = append(f, RuleNotStarted.New(loc, "presentation starts at %s, %.1fs from now",
				start.UTC().Format(time.RFC3339), start.Sub(now).Seconds()))
		}
	}
	return f
}

// liveSegments returns the segments in the availability window at the
// current time, keeping LiveMargin from both edges.
func (rc *repChecker) liveSegments() ([]addressing.Segment, error) {
	rep := rc.rep
	now := rc.clk.Now()
	if rep.AST.IsZero() {
		return nil, fmt.Errorf("no availabilityStartTime")
	}
	pt := rep.PresentationTime(now)
	if pt < 0 {
		return nil, nil // the Period has not started; reported by liveMPDChecks
	}
	if rep.PeriodDur >= 0 && pt-rep.TSBD > rep.PeriodDur {
		return nil, nil // the Period has left the time-shift buffer
	}
	from := pt - rep.TSBD
	if math.IsInf(from, -1) || from < 0 {
		from = 0
	}
	to := pt + rep.ATO
	segs, err := rep.Segments(from, to)
	if err != nil {
		return nil, err
	}
	margin := rc.opts.LiveMargin
	var avail []addressing.Segment
	for _, s := range segs {
		if !rep.AvailabilityStart(s).After(now.Add(-margin)) && rep.AvailabilityEnd(s).After(now.Add(margin)) {
			avail = append(avail, s)
		}
	}
	if len(avail) == 0 {
		rc.add(RuleNoSegments.New(rep.Loc(), "no segment is in the availability window at %s",
			now.UTC().Format(time.RFC3339)))
		return nil, nil
	}
	if rep.Kind == addressing.KindTimelineTime || rep.Kind == addressing.KindTimelineNumber {
		rc.checkStale(avail[len(avail)-1], now)
	}
	return avail, nil
}

// checkStale reports if the SegmentTimeline ends well before the live edge,
// which happens when the MPD is not updated or comes from a stale cache.
func (rc *repChecker) checkStale(newest addressing.Segment, now time.Time) {
	rep := rc.rep
	segDur := float64(newest.Dur) / float64(rep.Timescale)
	lag := now.Sub(rep.AvailabilityStart(newest)).Seconds()
	// Allow one segment still being produced plus the margin.
	if lag > 2*segDur+rc.opts.LiveMargin.Seconds() {
		rc.add(RuleStaleTimeline.New(rep.Loc(), "newest segment in the SegmentTimeline became available %.1fs ago (segment duration %.1fs)", lag, segDur))
	}
}

// selectLive picks the newest MaxSegments segments.
func (rc *repChecker) selectLive(segs []addressing.Segment) []addressing.Segment {
	n := rc.opts.MaxSegments
	if n < 0 || len(segs) <= n {
		return segs
	}
	return segs[len(segs)-n:]
}

// probeEarly requests the first segment after the newest available one whose
// availability start is clearly in the future. It should not be available yet.
func (rc *repChecker) probeEarly(ctx context.Context, newest addressing.Segment) {
	rep := rc.rep
	if rep.ATO > 0 {
		return // low-latency segments are available in part before they are complete
	}
	segDur := float64(newest.Dur) / float64(rep.Timescale)
	end := rep.End(newest)
	next, err := rep.Segments(end, end+3*segDur)
	if err != nil {
		return
	}
	now := rc.clk.Now()
	for _, seg := range next {
		if seg.Time <= newest.Time {
			continue
		}
		until := rep.AvailabilityStart(seg).Sub(now)
		if until < rc.opts.LiveMargin {
			continue
		}
		resp, err := rc.opts.Fetcher.Fetch(ctx, fetch.Request{URL: seg.URL})
		if err == nil && resp.OK() {
			rc.add(RuleEarly.New(rep.SegLoc(seg), "segment returned status %d %.1fs before its availability start time",
				resp.Status, until.Seconds()))
		}
		return
	}
}
