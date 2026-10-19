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
	"sync"
	"time"

	"github.com/tobbee/dashcheck"
	"github.com/tobbee/dashcheck/clock"
	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/internal"
	"github.com/tobbee/dashcheck/report"
	"github.com/tobbee/dashcheck/spec"
)

const usage = `Usage: dashcheck [options] <MPD URL or file>
       dashcheck -short [options] <MPD URL or file>...

dashcheck checks an MPEG-DASH MPD against the DASH XML schema and a set of
semantic rules, fetches the initialization segment and some media segments of
every Representation, and checks them against the MPD. For live (dynamic) MPDs
the clock is synchronized to the MPD's UTCTiming and the newest available
segments are checked against the availability window.

The profile-specific rules (DASH-IF IOP, CMAF, low latency, DVB-DASH, HbbTV,
CTA WAVE) follow the MPD's @profiles. -profiles checks other profiles: a list
of auto, iop, cmaf, low-latency, dvb, dvb2017, hbbtv and wave, or none. With
auto in the list, the chosen profiles are added to those the MPD signals;
without it, only the chosen profiles are checked.

-short prints a line per MPD instead of the report: FAIL (errors), WARN
(warnings) or OK, followed by the findings at -level if it is given. It takes
several MPDs, such as files/*.mpd, and checks -parallel of them at a time.
With -json, it writes an array of reports, with the findings at -level.

The exit code is 1 if any error was found, and 2 for bad options.

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
	profiles := fs.String("profiles", "auto", "profiles to check, such as dvb,hbbtv or auto,dvb (see above)")
	hlsInterop := fs.Bool("hls-interop", false, "also check the DASH-HLS interoperability rules of CTA-5005-B")
	updates := fs.Int("updates", dashcheck.DefaultMPDUpdates, "times a dynamic MPD is fetched again to check its updates (0 for none)")
	short := fs.Bool("short", false, "print a FAIL, WARN or OK line per MPD instead of the report; allows several MPDs")
	parallel := fs.Int("parallel", 4, "MPDs checked at the same time with -short")
	jsonOut := fs.Bool("json", false, "write the report as JSON")
	minSev := fs.String("level", "info", "lowest severity to show: info, warning or error (none with -short)")
	reference := fs.String("reference", "", "reference clock URL (http-iso), or \"none\" (default "+`"https://time.akamai.com/?iso&ms"`+")")
	timeout := fs.Duration("timeout", 20*time.Second, "timeout per HTTP request")
	verbose := fs.Bool("v", false, "show progress on stderr")
	listRules := fs.Bool("rules", false, "list all rules and exit")
	version := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version {
		fmt.Fprintf(stdout, "dashcheck %s\n", internal.Version())
		return 0
	}
	if *listRules {
		printRules(stdout)
		return 0
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return 2
	}
	sel, err := spec.Parse(*profiles)
	if err != nil {
		fmt.Fprintf(stderr, "bad -profiles: %v\n", err)
		return 2
	}
	var mpds []target
	for _, arg := range fs.Args() {
		u, err := toURL(arg)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		mpds = append(mpds, target{arg, u})
	}
	if len(mpds) > 1 && !*short {
		fmt.Fprintln(stderr, "several MPDs need -short")
		return 2
	}
	levelSet := false
	fs.Visit(func(f *flag.Flag) { levelSet = levelSet || f.Name == "level" })
	var minSeverity report.Severity
	if err := minSeverity.UnmarshalJSON([]byte(`"` + *minSev + `"`)); err != nil {
		fmt.Fprintf(stderr, "bad -level: %v\n", err)
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
	opts.Profiles = sel
	if *updates <= 0 {
		opts.MPDUpdates = -1
	}
	var mu sync.Mutex // the checks of several MPDs report progress at the same time
	progress := func(prefix string) func(dashcheck.Progress) {
		if !*verbose {
			return nil
		}
		return func(p dashcheck.Progress) {
			mu.Lock()
			defer mu.Unlock()
			fmt.Fprintf(stderr, "[%s%s %d/%d] %s\n", prefix, p.Phase, p.Done, p.Total, p.Message)
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	shown := func(rpt *report.Report) []report.Finding {
		var out []report.Finding
		if *short && !levelSet {
			return nil
		}
		for _, f := range rpt.Findings {
			if f.Severity >= minSeverity {
				out = append(out, f)
			}
		}
		return out
	}
	if !*short {
		opts.Progress = progress("")
		rpt := dashcheck.Check(ctx, mpds[0].url, opts)
		if *jsonOut {
			writeJSON(stdout, stderr, newJSONReport(rpt, shown(rpt)))
		} else {
			printReport(stdout, rpt, shown(rpt))
		}
		if rpt.HasErrors() {
			return 1
		}
		return 0
	}

	var reports []jsonReport
	var tally [3]int // MPDs with errors, with warnings only, and without either
	checkAll(ctx, mpds, opts, *parallel, progress, func(mpd target, rpt *report.Report) {
		c := rpt.Counts()
		switch {
		case c[report.Error] > 0:
			tally[0]++
		case c[report.Warning] > 0:
			tally[1]++
		default:
			tally[2]++
		}
		if *jsonOut {
			reports = append(reports, newJSONReport(rpt, shown(rpt)))
		} else {
			printLine(stdout, mpd.name, rpt, shown(rpt))
		}
	})
	switch {
	case *jsonOut:
		writeJSON(stdout, stderr, reports)
	case len(mpds) > 1:
		fmt.Fprintf(stdout, "\n%d MPDs: %d with errors, %d with warnings, %d OK\n", len(mpds), tally[0], tally[1], tally[2])
	}
	if tally[0] > 0 {
		return 1
	}
	return 0
}

// A target is an MPD to check: its URL, and the argument that named it, a URL or a path.
type target struct{ name, url string }

// checkAll checks the MPDs, up to parallel at a time, and calls done with
// each report in the order of the MPDs, as soon as it and those before it
// are done. progress gives the progress function for a prefix.
func checkAll(ctx context.Context, mpds []target, opts dashcheck.Options, parallel int,
	progress func(prefix string) func(dashcheck.Progress), done func(target, *report.Report)) {
	reports := make([]*report.Report, len(mpds))
	finished := make([]chan struct{}, len(mpds))
	for i := range finished {
		finished[i] = make(chan struct{})
	}
	sem := make(chan struct{}, max(parallel, 1))
	go func() {
		for i, mpd := range mpds {
			sem <- struct{}{}
			o := opts
			o.Progress = progress(fmt.Sprintf("mpd %d/%d ", i+1, len(mpds)))
			go func() {
				defer func() {
					<-sem
					close(finished[i])
				}()
				reports[i] = dashcheck.Check(ctx, mpd.url, o)
			}()
		}
	}()
	for i, mpd := range mpds {
		<-finished[i]
		done(mpd, reports[i])
	}
}

// printLine prints the -short result of an MPD: FAIL, WARN or OK, its name
// and the counts, followed by the findings shown.
func printLine(w io.Writer, name string, rpt *report.Report, shown []report.Finding) {
	c := rpt.Counts()
	status := "OK  "
	switch {
	case c[report.Error] > 0:
		status = "FAIL"
	case c[report.Warning] > 0:
		status = "WARN"
	}
	fmt.Fprintf(w, "%s  %s  (%d errors, %d warnings, %d info)\n", status, name, c[report.Error], c[report.Warning], c[report.Info])
	for _, f := range shown {
		fmt.Fprintf(w, "      %s\n", f)
	}
}

// jsonReport is a report as -json writes it: the findings at -level, and
// the counts of all findings.
type jsonReport struct {
	*report.Report
	Counts struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
		Info     int `json:"info"`
	} `json:"counts"`
}

func newJSONReport(rpt *report.Report, shown []report.Finding) jsonReport {
	out := *rpt
	out.Findings = shown
	if out.Findings == nil {
		out.Findings = []report.Finding{} // [] rather than null
	}
	jr := jsonReport{Report: &out}
	c := rpt.Counts()
	jr.Counts.Errors, jr.Counts.Warnings, jr.Counts.Info = c[report.Error], c[report.Warning], c[report.Info]
	return jr
}

func writeJSON(stdout, stderr io.Writer, v any) {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(stderr, err)
	}
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
		fmt.Fprintf(w, "Profiles: %s\n", describeProfiles(rpt))
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

// describeProfiles names the profiles whose rules were checked, and how they were chosen.
func describeProfiles(rpt *report.Report) string {
	names := "no profile-specific rules"
	if len(rpt.Profiles) > 0 {
		names = strings.Join(rpt.Profiles, ", ")
	}
	switch rpt.ProfileChoice {
	case "", "auto":
		return names + " (from @profiles)"
	default:
		return names + " (-profiles " + rpt.ProfileChoice + ")"
	}
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
