//go:build !js

package dashcheck

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/report"
	"github.com/tobbee/dashcheck/spec"
)

const testDir = "testdata/testpic_2s"

// server serves testDir. mpd replaces Manifest.mpd if set, and override
// maps request paths to other files in testDir ("" means 404).
func server(t *testing.T, mpd string, override map[string]string) *httptest.Server {
	t.Helper()
	files := http.FileServer(http.Dir(testDir))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mpd != "" && r.URL.Path == "/Manifest.mpd" {
			_, _ = w.Write([]byte(mpd))
			return
		}
		if target, ok := override[r.URL.Path]; ok {
			if target == "" {
				http.NotFound(w, r)
				return
			}
			r.URL.Path = target
		}
		files.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func ruleSet(findings []report.Finding, minSev report.Severity) []string {
	var ids []string
	for _, f := range findings {
		if f.Severity >= minSev {
			ids = append(ids, f.Rule)
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

type fixedClock time.Time

func (c fixedClock) Now() time.Time { return time.Time(c) }

func TestStatic(t *testing.T) {
	cases := []struct {
		name     string
		override map[string]string
		want     []string
	}{
		{"clean", nil, nil},
		{"missing segment", map[string]string{"/V300/2.m4s": ""}, []string{"HTTP-SEGMENT"}},
		{"missing init", map[string]string{"/A48/init.mp4": ""}, []string{"HTTP-INIT"}},
		{"wrong segment served", map[string]string{"/V300/2.m4s": "/V300/4.m4s"}, []string{"SEG-CONTINUITY", "SEG-EPT"}},
		{"audio init for video", map[string]string{"/V300/init.mp4": "/A48/init.mp4"}, []string{"SEG-TRACKID", "XCHK-CODECS", "XCHK-HANDLER"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := server(t, "", c.override)
			rpt := Check(context.Background(), ts.URL+"/Manifest.mpd", Options{Fetcher: fetch.NewNative(5 * time.Second)})
			if got := ruleSet(rpt.Findings, report.Warning); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v\n%v", got, c.want, rpt.Findings)
			}
		})
	}
}

// TestURLParams checks that the segments of an AdaptationSet that needs the
// URL parameters of Annex I are not fetched, since dashcheck cannot add them.
func TestURLParams(t *testing.T) {
	data, err := os.ReadFile(testDir + "/Manifest.mpd")
	if err != nil {
		t.Fatal(err)
	}
	video := `maxFrameRate="60/2">`
	mpd := strings.Replace(string(data), video, video+`
         <EssentialProperty schemeIdUri="urn:mpeg:dash:urlparam:2014"/>`, 1)
	// The server needs the parameters, so every request without them fails.
	ts := server(t, mpd, map[string]string{"/V300/init.mp4": "", "/V300/1.m4s": "", "/V300/2.m4s": "", "/V300/3.m4s": "", "/V300/4.m4s": ""})
	rpt := Check(context.Background(), ts.URL+"/Manifest.mpd", Options{Fetcher: fetch.NewNative(5 * time.Second)})
	if got := ruleSet(rpt.Findings, report.Warning); len(got) != 0 {
		t.Errorf("got %v\n%v", got, rpt.Findings)
	}
	if !slices.ContainsFunc(rpt.Findings, func(f report.Finding) bool {
		return f.Rule == "ADDR-UNSUPPORTED" && strings.Contains(f.Location.Path, "V300")
	}) {
		t.Errorf("no ADDR-UNSUPPORTED for V300: %v", rpt.Findings)
	}
}

// TestAfterRemotePeriod checks that the segments of a Period are not fetched
// when its timing depends on a remote Period, which dashcheck does not resolve.
func TestAfterRemotePeriod(t *testing.T) {
	data, err := os.ReadFile(testDir + "/Manifest.mpd")
	if err != nil {
		t.Fatal(err)
	}
	mpd := strings.Replace(string(data), `<Period id="one" start="PT0S">`, `<Period id="zero" xmlns:xlink="http://www.w3.org/1999/xlink" xlink:href="https://example.com/zero.xml"/>
   <Period id="one">`, 1)
	override := map[string]string{}
	for _, rep := range []string{"A48", "V300"} {
		for _, f := range []string{"init.mp4", "1.m4s", "2.m4s", "3.m4s", "4.m4s"} {
			override["/"+rep+"/"+f] = ""
		}
	}
	ts := server(t, mpd, override)
	rpt := Check(context.Background(), ts.URL+"/Manifest.mpd", Options{Fetcher: fetch.NewNative(5 * time.Second)})
	if got := ruleSet(rpt.Findings, report.Warning); !slices.Equal(got, []string{"MPD-XLINK-START"}) {
		t.Errorf("got %v, want [MPD-XLINK-START]\n%v", got, rpt.Findings)
	}
	if rpt.Segments != 0 {
		t.Errorf("%d segments checked", rpt.Segments)
	}
}

func liveMPD(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(testDir + "/Manifest.mpd")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Replace(string(data), `type="static" mediaPresentationDuration="PT8S"`,
		`type="dynamic" availabilityStartTime="2026-01-01T00:00:00Z" publishTime="2026-01-01T00:00:00Z" timeShiftBufferDepth="PT30S" minimumUpdatePeriod="PT2S"`, 1)
}

func TestLive(t *testing.T) {
	ast := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		at       time.Duration // clock time after availabilityStartTime
		override map[string]string
		want     []string
	}{
		// At AST+9s, segments 1-4 (ending at 2-8 s) are available and 5 is not on the server.
		{"in sync", 9 * time.Second, nil, []string{"MPD-DYN-UTCTIMING"}},
		// At AST+7s, segment 4 becomes available at 8 s but the server already has it.
		{"server early", 7 * time.Second, nil, []string{"LIVE-EARLY", "MPD-DYN-UTCTIMING"}},
		// Segment 4 should be available at AST+9s but the server lags behind.
		{"server late", 9 * time.Second, map[string]string{"/V300/4.m4s": ""}, []string{"HTTP-SEGMENT", "MPD-DYN-UTCTIMING"}},
		{"not started", -5 * time.Second, nil, []string{"LIVE-NOT-STARTED", "LIVE-PUBLISHTIME", "MPD-DYN-UTCTIMING"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := server(t, liveMPD(t), c.override)
			opts := Options{
				Fetcher:    fetch.NewNative(5 * time.Second),
				Clock:      fixedClock(ast.Add(c.at)),
				MPDUpdates: -1,
			}
			rpt := Check(context.Background(), ts.URL+"/Manifest.mpd", opts)
			if got := ruleSet(rpt.Findings, report.Warning); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v\n%v", got, c.want, rpt.Findings)
			}
		})
	}
}

// timelinePeriod returns a Period with the video Representation V300,
// addressed with a SegmentTimeline.
func timelinePeriod(id, start string, pto, startNumber int, timeline string) string {
	return fmt.Sprintf(`<Period id="%s" start="%s">
      <AdaptationSet contentType="video" id="1" mimeType="video/mp4" segmentAlignment="true" startWithSAP="1">
         <SegmentTemplate timescale="90000" presentationTimeOffset="%d" startNumber="%d" initialization="$RepresentationID$/init.mp4" media="$RepresentationID$/$Number$.m4s">
            <SegmentTimeline>%s</SegmentTimeline>
         </SegmentTemplate>
         <Representation id="V300" codecs="avc1.64001e" bandwidth="300000" width="640" height="360" frameRate="30" sar="1:1"/>
      </AdaptationSet>
   </Period>`, id, start, pto, startNumber, timeline)
}

// timelineMPD returns a dynamic MPD with the given @timeShiftBufferDepth and Periods.
func timelineMPD(tsbd string, periods ...string) string {
	return `<?xml version="1.0" encoding="utf-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" profiles="urn:mpeg:dash:profile:isoff-live:2011" type="dynamic" availabilityStartTime="2026-01-01T00:00:00Z" publishTime="2026-01-01T00:00:00Z" timeShiftBufferDepth="` + tsbd + `" minimumUpdatePeriod="PT2S" minBufferTime="PT2S" maxSegmentDuration="PT2S">
   ` + strings.Join(periods, "\n   ") + `
</MPD>`
}

// liveFindings checks mpd at the time at after its availabilityStartTime
// and returns all findings and those of rule.
func liveFindings(t *testing.T, mpd string, at time.Duration, rule string) (all, found []report.Finding) {
	t.Helper()
	ts := server(t, mpd, nil)
	opts := Options{
		Fetcher:    fetch.NewNative(5 * time.Second),
		Clock:      fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(at)),
		MPDUpdates: -1,
	}
	rpt := Check(context.Background(), ts.URL+"/Manifest.mpd", opts)
	for _, f := range rpt.Findings {
		if f.Rule == rule {
			found = append(found, f)
		}
	}
	return rpt.Findings, found
}

// TestLiveStaleEndedPeriod checks that the SegmentTimeline of a Period that
// has ended is measured at the end of the Period, not at the live edge.
func TestLiveStaleEndedPeriod(t *testing.T) {
	cases := []struct {
		name  string
		mpd   string
		at    time.Duration
		stale bool
	}{
		// Period one (0-2 s) ended 7 s ago with its segment; Period two has segments 2 to 4 at 2-8 s.
		{"timeline up to the end of an ended Period", timelineMPD("PT30S",
			timelinePeriod("one", "PT0S", 0, 1, `<S t="0" d="180000"/>`),
			timelinePeriod("two", "PT2S", 180000, 2, `<S t="180000" d="180000" r="2"/>`)), 9 * time.Second, false},
		// Period one (0-8 s) ends 6 s after its only segment; segments 3 and 4 play at 8-12 s in Period two.
		{"timeline that stops before the end of the Period", timelineMPD("PT30S",
			timelinePeriod("one", "PT0S", 0, 1, `<S t="0" d="180000"/>`),
			timelinePeriod("two", "PT8S", 360000, 3, `<S t="360000" d="180000" r="1"/>`)), 13 * time.Second, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			all, stale := liveFindings(t, c.mpd, c.at, RuleStaleTimeline.ID)
			if got := len(stale) > 0; got != c.stale {
				t.Errorf("LIVE-STALE %v, want %v\n%v", got, c.stale, all)
			}
		})
	}
}

// TestLiveNoSegments checks that a Period whose first segment is not
// complete yet, or whose last segment leaves the time-shift buffer, is not
// reported as having no segments.
func TestLiveNoSegments(t *testing.T) {
	// Period one has segments 1 and 2 at 0-4 s, Period two segments 3 and 4 at 4-8 s.
	twoPeriods := func(tsbd string) string {
		return timelineMPD(tsbd,
			timelinePeriod("one", "PT0S", 0, 1, `<S t="0" d="180000" r="1"/>`),
			timelinePeriod("two", "PT4S", 360000, 3, `<S t="360000" d="180000" r="1"/>`))
	}
	cases := []struct {
		name string
		mpd  string
		at   time.Duration
		want bool
	}{
		// Segment 3, the first of Period two, becomes available at 6 s.
		{"new Period before its first segment is complete", twoPeriods("PT30S"), 5 * time.Second, false},
		{"first segment of a new Period within the margin", twoPeriods("PT30S"), 6500 * time.Millisecond, false},
		// Segment 2, the last of Period one, leaves the 4 s time-shift buffer at 8 s.
		{"Period leaving the time-shift buffer within the margin", twoPeriods("PT4S"), 7500 * time.Millisecond, false},
		// The only segment (0-2 s) left the 4 s time-shift buffer at 6 s.
		{"timeline that ends before the time-shift buffer", timelineMPD("PT4S",
			timelinePeriod("one", "PT0S", 0, 1, `<S t="0" d="180000"/>`)), 9 * time.Second, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			all, none := liveFindings(t, c.mpd, c.at, RuleNoSegments.ID)
			if got := len(none) > 0; got != c.want {
				t.Errorf("LIVE-NO-SEGMENTS %v, want %v\n%v", got, c.want, all)
			}
		})
	}
}

func TestLiveUpdate(t *testing.T) {
	ast := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	v1 := liveMPD(t)
	cases := []struct {
		name string
		v2   string
		want []string
	}{
		{"unchanged", v1, nil},
		{"changed without new publishTime", strings.Replace(v1, "<Period", "<!-- changed -->\n<Period", 1), []string{"MPD-UPDATE-PUBLISHTIME"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := http.FileServer(http.Dir(testDir))
			fetches := 0
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path != "/Manifest.mpd":
					files.ServeHTTP(w, r)
				case fetches == 0:
					fetches++
					_, _ = w.Write([]byte(v1))
				default:
					_, _ = w.Write([]byte(c.v2))
				}
			}))
			t.Cleanup(ts.Close)
			opts := Options{
				Fetcher:        fetch.NewNative(5 * time.Second),
				Clock:          fixedClock(ast.Add(9 * time.Second)),
				UpdateInterval: time.Millisecond,
			}
			rpt := Check(context.Background(), ts.URL+"/Manifest.mpd", opts)
			var got []string
			for _, f := range rpt.Findings {
				if strings.HasPrefix(f.Rule, "MPD-UPDATE-") {
					got = append(got, f.Rule)
				}
			}
			if !slices.Equal(got, c.want) || rpt.MPDUpdates != 1 {
				t.Errorf("got %v and %d updates, want %v and 1", got, rpt.MPDUpdates, c.want)
			}
		})
	}
}

func TestProfileChoice(t *testing.T) {
	ts := server(t, "", nil)
	cases := []struct {
		choice  string
		want    []string // report.Profiles
		chosenW bool     // whether MPD-PROFILE-CHOSEN is reported
	}{
		{"auto", []string{"DASH-IF IOP"}, false},
		{"auto,dvb", []string{"DASH-IF IOP", "DVB-DASH 2014"}, true},
		{"dvb", []string{"DVB-DASH 2014"}, true},
		{"none", nil, false},
	}
	for _, c := range cases {
		t.Run(c.choice, func(t *testing.T) {
			sel, err := spec.Parse(c.choice)
			if err != nil {
				t.Fatal(err)
			}
			rpt := Check(context.Background(), ts.URL+"/Manifest.mpd",
				Options{Fetcher: fetch.NewNative(5 * time.Second), MaxSegments: 1, Profiles: sel})
			if rpt.ProfileChoice != c.choice || !slices.Equal(rpt.Profiles, c.want) {
				t.Errorf("got choice %q, profiles %v; want %q, %v", rpt.ProfileChoice, rpt.Profiles, c.choice, c.want)
			}
			if got := slices.Contains(ruleSet(rpt.Findings, report.Warning), "MPD-PROFILE-CHOSEN"); got != c.chosenW {
				t.Errorf("MPD-PROFILE-CHOSEN %v, want %v\n%v", got, c.chosenW, rpt.Findings)
			}
		})
	}
}

func TestSequenceNumbersOption(t *testing.T) {
	// Segment 2 is served again as segment 3, so its sequence number does not increase.
	ts := server(t, "", map[string]string{"/V300/3.m4s": "/V300/2.m4s"})
	for _, on := range []bool{false, true} {
		rpt := Check(context.Background(), ts.URL+"/Manifest.mpd",
			Options{Fetcher: fetch.NewNative(5 * time.Second), SequenceNumbers: on})
		if got := slices.Contains(ruleSet(rpt.Findings, report.Warning), "SEG-SEQUENCE-NUMBER"); got != on {
			t.Errorf("option %v: SEG-SEQUENCE-NUMBER reported %v\n%v", on, got, rpt.Findings)
		}
	}
}

func TestHLSBitrateOnDemand(t *testing.T) {
	// The first 2 s segment of V300 has 26% less than the average bitrate,
	// which only shows when all four segments are checked.
	ts := server(t, "", nil)
	for _, c := range []struct {
		maxSegments int
		want        bool
	}{{4, true}, {2, false}} {
		rpt := Check(context.Background(), ts.URL+"/Manifest.mpd",
			Options{Fetcher: fetch.NewNative(5 * time.Second), HLSInterop: true, MaxSegments: c.maxSegments})
		var got []string
		for _, f := range rpt.Findings {
			if f.Rule == "SEG-HLS-BITRATE" {
				got = append(got, f.Location.RepID)
			}
		}
		if want := map[bool][]string{true: {"V300"}}[c.want]; !slices.Equal(got, want) {
			t.Errorf("-n %d: SEG-HLS-BITRATE for %v, want %v", c.maxSegments, got, want)
		}
	}
}

func TestSegmentSequences(t *testing.T) {
	// The four 2 s video segments served as two Segment Sequences of two
	// Partial Segments each.
	data, err := os.ReadFile(testDir + "/Manifest.mpd")
	if err != nil {
		t.Fatal(err)
	}
	mpd := strings.Replace(string(data), `maxFrameRate="60/2">`,
		`maxFrameRate="60/2"><EssentialProperty schemeIdUri="urn:mpeg:dash:ssr:2023"/>`, 1)
	mpd = strings.Replace(mpd,
		`duration="2" media="$RepresentationID$/$Number$.m4s"/>
         <Representation id="V300"`,
		`duration="4" k="2" media="$RepresentationID$/$Number$_$SubNumber$.m4s"/>
         <Representation id="V300"`, 1)
	// @maxSegmentDuration bounds the Segment Sequence, not its Partial Segments.
	mpd = strings.Replace(mpd, `maxSegmentDuration="PT2S"`, `maxSegmentDuration="PT4S"`, 1)
	if !strings.Contains(mpd, `k="2"`) || !strings.Contains(mpd, "ssr:2023") || !strings.Contains(mpd, "PT4S") {
		t.Fatal("test MPD not modified")
	}
	ts := server(t, mpd, map[string]string{
		"/V300/1_0.m4s": "/V300/1.m4s", "/V300/1_1.m4s": "/V300/2.m4s",
		"/V300/2_0.m4s": "/V300/3.m4s", "/V300/2_1.m4s": "/V300/4.m4s",
	})
	rpt := Check(context.Background(), ts.URL+"/Manifest.mpd", Options{Fetcher: fetch.NewNative(5 * time.Second), MaxSegments: -1})
	if got := ruleSet(rpt.Findings, report.Info); len(got) > 0 {
		t.Errorf("got %v\n%v", got, rpt.Findings)
	}
	if rpt.Segments != 8 {
		t.Errorf("%d segments checked, want 8", rpt.Segments)
	}
}
