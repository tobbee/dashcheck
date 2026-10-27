// Package clock provides a wall clock synchronized to the UTCTiming
// sources signaled in an MPD, the reference that players use.
package clock

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/report"
)

// Clock returns the current time.
type Clock interface {
	Now() time.Time
}

// System is the local clock.
type System struct{}

func (System) Now() time.Time { return time.Now() }

// Offset is the local clock corrected by a measured offset.
type Offset struct {
	Offset time.Duration // reference minus local
	Source string
}

func (o *Offset) Now() time.Time { return time.Now().Add(o.Offset) }

// DefaultReference is used when the MPD has no usable UTCTiming.
// It sends CORS headers, so it works from browsers.
const DefaultReference = "https://time.akamai.com/?iso&ms"

// MaxSkew is the largest acceptable difference between a UTCTiming source and the reference.
const MaxSkew = 500 * time.Millisecond

const (
	SchemeHTTPXSDate = "urn:mpeg:dash:utc:http-xsdate:2014"
	SchemeHTTPISO    = "urn:mpeg:dash:utc:http-iso:2014"
	SchemeHTTPHead   = "urn:mpeg:dash:utc:http-head:2014"
	SchemeDirect     = "urn:mpeg:dash:utc:direct:2014"
)

var (
	RuleSource = report.Register(report.Rule{
		ID: "CLOCK-SOURCE", Severity: report.Error,
		Desc:    "Each UTCTiming source must respond with a parseable time",
		SpecRef: "ISO/IEC 23009-1 (UTCTiming)",
	})
	RuleSkew = report.Register(report.Rule{
		ID: "CLOCK-SKEW", Severity: report.Warning,
		Desc: fmt.Sprintf("A UTCTiming source should be within %v of the reference clock", MaxSkew),
	})
	RuleUnsupported = report.Register(report.Rule{
		ID: "CLOCK-UNSUPPORTED", Severity: report.Info,
		Desc: "UTCTiming schemes that the checker cannot measure are skipped",
	})
	RuleLocal = report.Register(report.Rule{
		ID: "CLOCK-LOCAL", Severity: report.Info,
		Desc: "The offset between the local clock and the reference clock",
	})
)

// Measurement is the result of one clock measurement.
type Measurement struct {
	Scheme string
	Value  string
	Offset time.Duration // source minus local
	RTT    time.Duration
	// Resolution of the source's time, 1 s for a Date header or a time
	// without fractional seconds. Offset assumes the middle of it.
	Resolution time.Duration
}

// Source names the time source: its URL, or for the direct scheme, whose
// value is a time that differs in each MPD, the scheme.
func (meas *Measurement) Source() string {
	if meas.Scheme == SchemeDirect {
		return SchemeDirect + " (the time in the MPD)"
	}
	return meas.Value
}

// Measure queries one UTCTiming source. mpdResp is the response that
// delivered the MPD and is used for the direct scheme.
func Measure(ctx context.Context, f fetch.Fetcher, scheme, value string, mpdResp *fetch.Response) (*Measurement, error) {
	meas := &Measurement{Scheme: scheme, Value: value}
	value = strings.TrimSpace(value)
	if first, _, ok := strings.Cut(value, " "); ok {
		value = first // a whitespace-separated list of URLs
	}
	switch scheme {
	case SchemeDirect:
		t, err := addressing.ParseDateTime(value)
		if err != nil {
			return nil, fmt.Errorf("direct value %q: %v", value, err)
		}
		if mpdResp == nil {
			return nil, fmt.Errorf("direct scheme needs the MPD response time")
		}
		// The ABNF of ISO/IEC 23009-1 5.8.5.7 has whole seconds.
		if !strings.Contains(value, ".") {
			meas.Resolution = time.Second
		}
		meas.Offset = t.Add(meas.Resolution / 2).Sub(midpoint(mpdResp))
		meas.RTT = mpdResp.End.Sub(mpdResp.Start)
		return meas, nil
	case SchemeHTTPXSDate, SchemeHTTPISO:
		resp, err := fetch.Get(ctx, f, value)
		if err != nil {
			return nil, err
		}
		s := strings.TrimSpace(string(resp.Body))
		t, err := addressing.ParseDateTime(s)
		if err != nil {
			return nil, fmt.Errorf("response %q from %s: %v", truncate(s, 40), value, err)
		}
		// Some servers, such as https://time.akamai.com/?iso, send whole seconds.
		if !strings.Contains(s, ".") {
			meas.Resolution = time.Second
		}
		meas.Offset = t.Add(meas.Resolution / 2).Sub(midpoint(resp))
		meas.RTT = resp.End.Sub(resp.Start)
		return meas, nil
	case SchemeHTTPHead:
		resp, err := f.Fetch(ctx, fetch.Request{URL: value, Head: true})
		if err != nil {
			return nil, err
		}
		date := resp.Header["date"]
		if date == "" {
			return nil, fmt.Errorf("no Date header from %s (not sent, or not exposed by CORS)", value)
		}
		t, err := time.Parse(time.RFC1123, date)
		if err != nil {
			return nil, fmt.Errorf("bad Date header %q from %s: %v", date, value, err)
		}
		meas.Resolution = time.Second
		meas.Offset = t.Add(meas.Resolution / 2).Sub(midpoint(resp))
		meas.RTT = resp.End.Sub(resp.Start)
		return meas, nil
	default:
		return nil, errUnsupported
	}
}

var errUnsupported = fmt.Errorf("unsupported scheme")

func midpoint(r *fetch.Response) time.Time {
	return r.Start.Add(r.End.Sub(r.Start) / 2)
}

func host(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		return u.Host
	}
	return rawURL
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Sync measures the MPD's UTCTiming sources and the reference, and returns
// a clock following the first working MPD source. If none works it follows
// the reference, and if that fails too, the local clock.
func Sync(ctx context.Context, f fetch.Fetcher, timings []*m.DescriptorType, reference string, mpdResp *fetch.Response) (Clock, string, []report.Finding) {
	var findings []report.Finding
	var chosen *Measurement
	var measured []*Measurement
	var locs []report.Location // of measured
	for i, u := range timings {
		loc := report.Location{Path: fmt.Sprintf("MPD/UTCTiming[%d]", i)}
		meas, err := Measure(ctx, f, string(u.SchemeIdUri), u.Value, mpdResp)
		if err == errUnsupported {
			findings = append(findings, RuleUnsupported.New(loc, "cannot measure %s", u.SchemeIdUri))
			continue
		}
		if err != nil {
			findings = append(findings, RuleSource.New(loc, "%s: %v", u.SchemeIdUri, err))
			continue
		}
		measured = append(measured, meas)
		locs = append(locs, loc)
		if chosen == nil {
			chosen = meas
		}
	}
	var ref *Measurement
	if reference != "" {
		// Reuse the measurement if the MPD points at the reference, so that
		// the same server is not reported with two slightly different offsets.
		for _, meas := range measured {
			if (meas.Scheme == SchemeHTTPISO || meas.Scheme == SchemeHTTPXSDate) && strings.TrimSpace(meas.Value) == reference {
				ref = meas
				break
			}
		}
		if ref == nil {
			var err error
			ref, err = Measure(ctx, f, SchemeHTTPISO, reference, nil)
			if err != nil {
				findings = append(findings, RuleSource.New(report.Location{URL: reference}, "reference clock: %v", err))
				ref = nil
			}
		}
	}
	if ref != nil {
		findings = append(findings, RuleLocal.New(report.Location{URL: reference},
			"this computer's clock is %s the reference clock %s (round trip %v)",
			Describe(-ref.Offset), host(reference), ref.RTT.Round(time.Millisecond)))
		for i, meas := range measured {
			// Both offsets are relative to the local clock, so their difference is source minus reference.
			skew := meas.Offset - ref.Offset
			tol := MaxSkew + (meas.RTT+ref.RTT)/2 + meas.Resolution/2
			if math.Abs(float64(skew)) > float64(tol) {
				loc := report.Location{URL: meas.Value}
				if meas.Scheme == SchemeDirect {
					loc = locs[i]
				}
				findings = append(findings, RuleSkew.New(loc,
					"UTCTiming source %s is %s the reference clock %s", meas.Source(), Describe(skew), host(reference)))
			}
		}
	}
	switch {
	case chosen != nil:
		return &Offset{Offset: chosen.Offset, Source: chosen.Source()}, chosen.Source(), findings
	case ref != nil:
		return &Offset{Offset: ref.Offset, Source: ref.Value}, ref.Value, findings
	default:
		return System{}, LocalSource, findings
	}
}

// LocalSource is the clock source name when no time source could be reached.
const LocalSource = "local clock"

// Describe describes a clock relative to another, given their difference d
// (this minus other): "45ms behind", "12ms ahead of" or "in sync with".
func Describe(d time.Duration) string {
	d = d.Round(time.Millisecond)
	switch {
	case d > 0:
		return fmt.Sprintf("%v ahead of", d)
	case d < 0:
		return fmt.Sprintf("%v behind", -d)
	default:
		return "in sync with"
	}
}
