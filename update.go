package dashcheck

import (
	"context"
	"net/url"
	"strings"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/clock"
	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/mpdcheck"
	"github.com/tobbee/dashcheck/report"
)

const (
	minUpdateInterval = 2 * time.Second
	maxUpdateInterval = 10 * time.Second
)

// checkUpdates fetches a dynamic MPD again opts.MPDUpdates times and checks
// each version against the one before.
func checkUpdates(ctx context.Context, rpt *report.Report, mpdURL string, md *m.MPD, resp *fetch.Response, clk clock.Clock, opts *Options) {
	if opts.MPDUpdates < 0 {
		return
	}
	interval := opts.UpdateInterval
	if interval <= 0 {
		interval = updateInterval(md)
	}
	target := updateURL(md, mpdURL, resp.URL)
	prev := mpdcheck.Version{MPD: md, Data: resp.Body, Time: clk.Now().Add(-time.Since(resp.End))}
	last := resp.End
	for i := range opts.MPDUpdates {
		wait := max(time.Until(last.Add(interval)), 0)
		opts.progress("updates", i, opts.MPDUpdates, "fetching the MPD again in %.1fs to check the update", wait.Seconds())
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		r, err := fetch.Get(ctx, opts.Fetcher, target)
		if err != nil {
			rpt.Findings = append(rpt.Findings, RuleHTTPMPD.New(report.Location{URL: target}, "MPD update: %v", err))
			return
		}
		next, err := m.ReadFromString(string(r.Body))
		if err != nil {
			rpt.Findings = append(rpt.Findings, mpdcheck.RuleParse.New(report.Location{URL: r.URL}, "MPD update: %v", err))
			return
		}
		cur := mpdcheck.Version{MPD: next, Data: r.Body, Time: clk.Now().Add(-time.Since(r.End))}
		rpt.Findings = append(rpt.Findings, mpdcheck.CheckUpdate(prev, cur, r.URL)...)
		rpt.MPDUpdates++
		if next.GetType() != "dynamic" {
			return
		}
		prev, last = cur, r.End
	}
}

// updateInterval is the time between MPD fetches: about one segment, so
// that SegmentTimeline MPDs have changed.
func updateInterval(md *m.MPD) time.Duration {
	d := minUpdateInterval
	switch {
	case md.MaxSegmentDuration != nil:
		d = time.Duration(*md.MaxSegmentDuration)
	case md.MinimumUpdatePeriod != nil:
		d = time.Duration(*md.MinimumUpdatePeriod)
	}
	return min(max(d, minUpdateInterval), maxUpdateInterval)
}

// updateURL is where updates of the MPD are fetched: the first MPD.Location,
// or else the original URL.
func updateURL(md *m.MPD, mpdURL, finalURL string) string {
	if len(md.Location) == 0 {
		return mpdURL
	}
	loc := strings.TrimSpace(md.Location[0].Value)
	base, err1 := url.Parse(finalURL)
	ref, err2 := url.Parse(loc)
	if loc == "" || err1 != nil || err2 != nil {
		return mpdURL
	}
	return base.ResolveReference(ref).String()
}
