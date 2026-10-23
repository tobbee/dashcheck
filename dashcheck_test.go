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

// TestLiveStaleEndedPeriod checks that the SegmentTimeline of a Period that
// has ended is measured at the end of the Period, not at the live edge.
func TestLiveStaleEndedPeriod(t *testing.T) {
	ast := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	period := func(id, start string, pto, startNumber int, timeline string) string {
		return fmt.Sprintf(`<Period id="%s" start="%s">
      <AdaptationSet contentType="video" id="1" mimeType="video/mp4" segmentAlignment="true" startWithSAP="1">
         <SegmentTemplate timescale="90000" presentationTimeOffset="%d" startNumber="%d" initialization="$RepresentationID$/init.mp4" media="$RepresentationID$/$Number$.m4s">
            <SegmentTimeline>%s</SegmentTimeline>
         </SegmentTemplate>
         <Representation id="V300" codecs="avc1.64001e" bandwidth="300000" width="640" height="360" frameRate="30" sar="1:1"/>
      </AdaptationSet>
   </Period>`, id, start, pto, startNumber, timeline)
	}
	mpd := func(periods ...string) string {
		return `<?xml version="1.0" encoding="utf-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" profiles="urn:mpeg:dash:profile:isoff-live:2011" type="dynamic" availabilityStartTime="2026-01-01T00:00:00Z" publishTime="2026-01-01T00:00:00Z" timeShiftBufferDepth="PT30S" minimumUpdatePeriod="PT2S" minBufferTime="PT2S" maxSegmentDuration="PT2S">
   ` + strings.Join(periods, "\n   ") + `
</MPD>`
	}
	cases := []struct {
		name  string
		mpd   string
		at    time.Duration
		stale bool
	}{
		// Period one (0-2 s) ended 7 s ago with its segment; Period two has segments 2 to 4 at 2-8 s.
		{"timeline up to the end of an ended Period", mpd(
			period("one", "PT0S", 0, 1, `<S t="0" d="180000"/>`),
			period("two", "PT2S", 180000, 2, `<S t="180000" d="180000" r="2"/>`)), 9 * time.Second, false},
		// Period one (0-8 s) ends 6 s after its only segment; segments 3 and 4 play at 8-12 s in Period two.
		{"timeline that stops before the end of the Period", mpd(
			period("one", "PT0S", 0, 1, `<S t="0" d="180000"/>`),
			period("two", "PT8S", 360000, 3, `<S t="360000" d="180000" r="1"/>`)), 13 * time.Second, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := server(t, c.mpd, nil)
			opts := Options{
				Fetcher:    fetch.NewNative(5 * time.Second),
				Clock:      fixedClock(ast.Add(c.at)),
				MPDUpdates: -1,
			}
			rpt := Check(context.Background(), ts.URL+"/Manifest.mpd", opts)
			var stale []report.Finding
			for _, f := range rpt.Findings {
				if f.Rule == RuleStaleTimeline.ID {
					stale = append(stale, f)
				}
			}
			if got := len(stale) > 0; got != c.stale {
				t.Errorf("LIVE-STALE %v, want %v\n%v", got, c.stale, rpt.Findings)
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
	if !strings.Contains(mpd, `k="2"`) || !strings.Contains(mpd, "ssr:2023") {
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
