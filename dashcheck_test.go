//go:build !js

package dashcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/report"
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
