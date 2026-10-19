// Command dashcheck checks an MPEG-DASH stream: the MPD, its segments and,
// for live streams, the timing against a synchronized clock.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/tobbee/dashcheck"
	"github.com/tobbee/dashcheck/clock"
	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/report"
)

const usage = `Usage: dashcheck [options] <MPD URL or file>

dashcheck checks an MPEG-DASH MPD against the DASH XML schema and a set of
semantic rules, fetches the initialization segment and some media segments of
every Representation, and checks them against the MPD. For live (dynamic) MPDs
the clock is synchronized to the MPD's UTCTiming and the newest available
segments are checked against the availability window.

The exit code is 1 if any error was found.

Options:
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dashcheck", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
	}
	nSegs := fs.Int("n", dashcheck.DefaultMaxSegments, "media segments to check per Representation")
	all := fs.Bool("all", false, "check all segments (static MPDs)")
	hlsInterop := fs.Bool("hls-interop", false, "also check the DASH-HLS interoperability rules of CTA-5005-B")
	updates := fs.Int("updates", dashcheck.DefaultMPDUpdates, "times a dynamic MPD is fetched again to check its updates (0 for none)")
	jsonOut := fs.Bool("json", false, "write the report as JSON")
	minSev := fs.String("level", "info", "lowest severity to show: info, warning or error")
	reference := fs.String("reference", "", "reference clock URL (http-iso), or \"none\" (default "+`"https://time.akamai.com/?iso&ms"`+")")
	timeout := fs.Duration("timeout", 20*time.Second, "timeout per HTTP request")
	verbose := fs.Bool("v", false, "show progress on stderr")
	listRules := fs.Bool("rules", false, "list all rules and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *listRules {
		printRules(stdout)
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	var minSeverity report.Severity
	if err := minSeverity.UnmarshalJSON([]byte(`"` + *minSev + `"`)); err != nil {
		fmt.Fprintf(stderr, "bad -level: %v\n", err)
		return 2
	}
	mpdURL, err := toURL(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	opts := dashcheck.Options{
		Fetcher:     fetch.NewNative(*timeout),
		MaxSegments: *nSegs,
		Reference:   *reference,
	}
	if *all {
		opts.MaxSegments = -1
	}
	opts.MPDUpdates = *updates
	opts.HLSInterop = *hlsInterop
	if *updates <= 0 {
		opts.MPDUpdates = -1
	}
	if *verbose {
		opts.Progress = func(p dashcheck.Progress) {
			fmt.Fprintf(stderr, "[%s %d/%d] %s\n", p.Phase, p.Done, p.Total, p.Message)
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	rpt := dashcheck.Check(ctx, mpdURL, opts)
	var shown []report.Finding
	for _, f := range rpt.Findings {
		if f.Severity >= minSeverity {
			shown = append(shown, f)
		}
	}
	if *jsonOut {
		out := *rpt
		out.Findings = shown
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			fmt.Fprintln(stderr, err)
		}
	} else {
		printReport(stdout, rpt, shown)
	}
	if rpt.HasErrors() {
		return 1
	}
	return 0
}

// toURL turns a local path into a file URL; URLs are returned as is.
func toURL(arg string) (string, error) {
	if strings.Contains(arg, "://") {
		return arg, nil
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", err
	}
	return "file://" + filepath.ToSlash(abs), nil
}

func printReport(w io.Writer, rpt *report.Report, shown []report.Finding) {
	fmt.Fprintf(w, "MPD:      %s\n", rpt.MPDURL)
	if rpt.Type != "" {
		fmt.Fprintf(w, "Type:     %s\n", rpt.Type)
	}
	switch rpt.ClockSource {
	case "":
	case clock.LocalSource:
		fmt.Fprintf(w, "Clock:    local clock (no time source could be reached)\n")
	default:
		// ClockOffset is the source minus this computer's clock.
		offset := time.Duration(rpt.ClockOffset * float64(time.Second))
		fmt.Fprintf(w, "Clock:    synced to %s; this computer's clock is %s it\n", rpt.ClockSource, clock.Describe(-offset))
	}
	fmt.Fprintf(w, "Segments: %d checked in %.1fs\n", rpt.Segments, rpt.Duration)
	if rpt.MPDUpdates > 0 {
		times := "time"
		if rpt.MPDUpdates > 1 {
			times = "times"
		}
		fmt.Fprintf(w, "Updates:  MPD fetched %d more %s and compared with the version before\n", rpt.MPDUpdates, times)
	}
	fmt.Fprintln(w)
	for _, f := range shown {
		fmt.Fprintln(w, f)
	}
	c := rpt.Counts()
	if len(shown) > 0 {
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "%d errors, %d warnings, %d info\n", c[report.Error], c[report.Warning], c[report.Info])
}

func printRules(w io.Writer) {
	group := ""
	for _, r := range report.Rules() {
		if r.Group != group {
			if group != "" {
				fmt.Fprintln(w)
			}
			group = r.Group
			fmt.Fprintf(w, "%s\n", group)
		}
		ref := ""
		if r.SpecRef != "" {
			ref = " (" + r.SpecRef + ")"
		}
		fmt.Fprintf(w, "%-22s %-7s %s%s\n", r.ID, r.Severity, r.Desc, ref)
	}
}
