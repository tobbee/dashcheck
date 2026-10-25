//go:build !js

package dashcheck

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/report"
)

const onDemandMPD = `<?xml version="1.0" encoding="utf-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" profiles="urn:mpeg:dash:profile:isoff-on-demand:2011" minBufferTime="PT2S" type="static" mediaPresentationDuration="PT6S">
  <Period id="one" start="PT0S">
    <AdaptationSet contentType="video" mimeType="video/mp4" subsegmentAlignment="true" subsegmentStartsWithSAP="1" par="16:9" maxWidth="640" maxHeight="360">
      <Representation id="V300" codecs="avc1.64001e" bandwidth="300000" width="640" height="360" frameRate="30" sar="1:1">
        <BaseURL>track.mp4</BaseURL>
        <SegmentBase indexRange="%s"><Initialization range="%s"/></SegmentBase>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`

// trackFile builds a track file from the V300 init segment and media
// segments, with a sidx box that skips the first skip segments with
// first_offset and indexes the next indexed ones.
func trackFile(t *testing.T, segs []string, skip int, indexed int) (file []byte, mpd string) {
	t.Helper()
	read := func(name string) []byte {
		data, err := os.ReadFile(testDir + "/V300/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	init := read("init.mp4")
	var media [][]byte
	for _, s := range segs {
		media = append(media, read(s))
	}
	sidx := &mp4.SidxBox{ReferenceID: 1, Timescale: 90000}
	for i, m := range media {
		switch {
		case i < skip:
			sidx.FirstOffset += uint64(len(m))
		case i < skip+indexed:
			sidx.SidxRefs = append(sidx.SidxRefs, mp4.SidxRef{ReferencedSize: uint32(len(m)), SubSegmentDuration: 180000, StartsWithSAP: 1, SAPType: 1})
		}
	}
	var buf bytes.Buffer
	buf.Write(init)
	if err := sidx.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	for _, m := range media {
		buf.Write(m)
	}
	indexRange := fmt.Sprintf("%d-%d", len(init), uint64(len(init))+sidx.Size()-1)
	return buf.Bytes(), fmt.Sprintf(onDemandMPD, indexRange, fmt.Sprintf("0-%d", len(init)-1))
}

func TestSidxSpan(t *testing.T) {
	segs := []string{"1.m4s", "2.m4s", "3.m4s"}
	cases := []struct {
		name          string
		skip, indexed int
		want          string // part of the SEG-SIDX-SPAN message, if any
	}{
		{"all indexed", 0, 3, ""},
		{"last fragment not indexed", 0, 2, "after the last of 2 subsegments"},
		{"first fragment skipped with first_offset", 1, 2, "before the first of 2 subsegments"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file, mpd := trackFile(t, segs, c.skip, c.indexed)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/Manifest.mpd" {
					_, _ = w.Write([]byte(mpd))
					return
				}
				http.ServeContent(w, r, "track.mp4", time.Time{}, bytes.NewReader(file))
			}))
			defer ts.Close()
			rpt := Check(context.Background(), ts.URL+"/Manifest.mpd", Options{Fetcher: fetch.NewNative(5 * time.Second), MaxSegments: -1})
			var got []string
			for _, f := range rpt.Findings {
				if f.Rule == "SEG-SIDX-SPAN" {
					got = append(got, f.Message)
				}
			}
			switch {
			case c.want == "" && len(got) > 0:
				t.Errorf("got %q", got)
			case c.want != "" && (len(got) != 1 || !strings.Contains(got[0], c.want)):
				t.Errorf("got %q, want one message with %q\n%v", got, c.want, rpt.Findings)
			}
		})
	}
}

// serveTrack serves the MPD and the track file, and records the Range header
// of each request for the track file. With ignoreRange, it answers range
// requests with the whole file.
func serveTrack(t *testing.T, mpd string, file []byte, ignoreRange bool) (url string, ranges *[]string) {
	t.Helper()
	var mu sync.Mutex
	ranges = &[]string{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Manifest.mpd" {
			_, _ = w.Write([]byte(mpd))
			return
		}
		mu.Lock()
		*ranges = append(*ranges, r.Header.Get("Range"))
		mu.Unlock()
		if ignoreRange {
			_, _ = w.Write(file)
			return
		}
		http.ServeContent(w, r, "track.mp4", time.Time{}, bytes.NewReader(file))
	}))
	t.Cleanup(ts.Close)
	return ts.URL + "/Manifest.mpd", ranges
}

func TestSelfInitializing(t *testing.T) {
	file, mpd := trackFile(t, []string{"1.m4s", "2.m4s", "3.m4s"}, 0, 3)
	mpd = regexp.MustCompile(`<Initialization [^>]*/>`).ReplaceAllString(mpd, "")
	url, ranges := serveTrack(t, mpd, file, false)
	rpt := Check(context.Background(), url, Options{Fetcher: fetch.NewNative(5 * time.Second), MaxSegments: -1})
	for _, f := range rpt.Findings {
		if f.Severity == report.Error {
			t.Errorf("unexpected error %s: %s", f.Rule, f.Message)
		}
	}
	init, err := os.ReadFile(testDir + "/V300/init.mp4")
	if err != nil {
		t.Fatal(err)
	}
	// The Initialization Segment is what precedes @indexRange, not the whole file.
	if want := fmt.Sprintf("bytes=0-%d", len(init)-1); !slices.Contains(*ranges, want) || slices.Contains(*ranges, "") {
		t.Errorf("requested ranges %q, want %q and no request for the whole file", *ranges, want)
	}
}
