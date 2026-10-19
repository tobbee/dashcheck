package addressing

import (
	"math"
	"os"
	"testing"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
)

func TestSubstitute(t *testing.T) {
	v := Values{RepresentationID: "V300", Number: 7, Time: 180000, Bandwidth: 300000}
	cases := []struct {
		tmpl, want string
		wantErr    bool
	}{
		{"$RepresentationID$/$Number$.m4s", "V300/7.m4s", false},
		{"$RepresentationID$/$Number%05d$.m4s", "V300/00007.m4s", false},
		{"seg-$Time$-$Bandwidth$.mp4", "seg-180000-300000.mp4", false},
		{"cost$$-$Number%3d$", "cost$-007", false},
		{"$Number%x$", "7", false},
		{"$RepresentationID%05d$", "", true},
		{"$Foo$", "", true},
		{"$Number", "", true},
	}
	for _, c := range cases {
		got, err := Substitute(c.tmpl, v)
		if (err != nil) != c.wantErr {
			t.Errorf("%q: err=%v, wantErr=%v", c.tmpl, err, c.wantErr)
			continue
		}
		if got != c.want {
			t.Errorf("%q: got %q, want %q", c.tmpl, got, c.want)
		}
	}
}

func mustReadMPD(t *testing.T, s string) *m.MPD {
	t.Helper()
	md, err := m.ReadFromString(s)
	if err != nil {
		t.Fatal(err)
	}
	return md
}

func TestNumberAddressing(t *testing.T) {
	data, err := os.ReadFile("../testdata/testpic_2s/Manifest.mpd")
	if err != nil {
		t.Fatal(err)
	}
	md := mustReadMPD(t, string(data))
	reps, findings := Reps(md, "https://example.com/a/Manifest.mpd")
	if len(findings) != 0 {
		t.Fatalf("unexpected findings: %v", findings)
	}
	if len(reps) != 2 {
		t.Fatalf("got %d reps", len(reps))
	}
	v := reps[1]
	if v.ID != "V300" || v.Kind != KindNumber || v.Codecs != "avc1.64001e" || v.MimeType != "video/mp4" {
		t.Errorf("bad rep %+v", v)
	}
	if v.InitURL != "https://example.com/a/V300/init.mp4" {
		t.Errorf("init URL %s", v.InitURL)
	}
	segs, err := v.Segments(0, v.PeriodDur)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 4 {
		t.Fatalf("got %d segments, want 4", len(segs))
	}
	last := segs[3]
	if last.Number != 4 || last.URL != "https://example.com/a/V300/4.m4s" || v.Start(last) != 6 || v.End(last) != 8 {
		t.Errorf("bad last segment %+v start=%v", last, v.Start(last))
	}
}

const timelineMPD = `<?xml version="1.0"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="dynamic" availabilityStartTime="1970-01-01T00:00:00Z"
     profiles="urn:mpeg:dash:profile:isoff-live:2011" minBufferTime="PT2S" timeShiftBufferDepth="PT10S">
  <BaseURL>https://cdn.example.com/live/</BaseURL>
  <Period id="p0" start="PT0S">
    <BaseURL>p0/</BaseURL>
    <AdaptationSet id="1" contentType="video" mimeType="video/mp4" codecs="avc1.64001e">
      <SegmentTemplate timescale="90000" presentationTimeOffset="900000" media="$RepresentationID$/$Time$.m4s" initialization="$RepresentationID$/init.mp4">
        <SegmentTimeline>
          <S t="900000" d="180000" r="2"/>
          <S d="90000"/>
          <S t="1530000" d="180000" r="-1"/>
        </SegmentTimeline>
      </SegmentTemplate>
      <Representation id="v1" bandwidth="1000000"/>
    </AdaptationSet>
  </Period>
</MPD>`

func TestTimelineAddressing(t *testing.T) {
	md := mustReadMPD(t, timelineMPD)
	reps, findings := Reps(md, "https://origin.example.com/x/live.mpd")
	if len(findings) != 0 {
		t.Fatalf("unexpected findings: %v", findings)
	}
	r := reps[0]
	if r.Kind != KindTimelineTime || r.Timescale != 90000 || r.PTO != 900000 || r.TSBD != 10 {
		t.Fatalf("bad rep %+v", r)
	}
	if r.InitURL != "https://cdn.example.com/live/p0/v1/init.mp4" {
		t.Errorf("init URL %s", r.InitURL)
	}
	// Open-ended Period: the negative @r needs a bounded interval.
	if _, err := r.Segments(0, math.Inf(1)); err == nil {
		t.Errorf("expected error for open-ended interval")
	}
	segs, err := r.Segments(0, 14)
	if err != nil {
		t.Fatal(err)
	}
	wantTimes := []uint64{900000, 1080000, 1260000, 1440000, 1530000, 1710000, 1890000, 2070000}
	if len(segs) != len(wantTimes) {
		t.Fatalf("got %d segments, want %d: %+v", len(segs), len(wantTimes), segs)
	}
	for i, s := range segs {
		if s.Time != wantTimes[i] {
			t.Errorf("segment %d: t=%d, want %d", i, s.Time, wantTimes[i])
		}
	}
	if segs[3].Dur != 90000 || r.Start(segs[3]) != 6 || r.End(segs[3]) != 7 {
		t.Errorf("bad short segment %+v", segs[3])
	}
	if segs[0].URL != "https://cdn.example.com/live/p0/v1/900000.m4s" {
		t.Errorf("URL %s", segs[0].URL)
	}
	// Window selection: segments overlapping [5, 9).
	segs, _ = r.Segments(5, 9)
	if len(segs) != 3 || segs[0].Time != 1260000 {
		t.Errorf("window: %+v", segs)
	}
	// Availability: the segment ending at 9 s becomes available at AST + 9 s.
	if got := r.AvailabilityStart(segs[2]); !got.Equal(time.Unix(9, 0)) {
		t.Errorf("availability start %v", got)
	}
}

func TestTemplateErrors(t *testing.T) {
	cases := map[string]string{
		"both ids":         `media="$Number$-$Time$.m4s" duration="2"`,
		"time no timeline": `media="$Time$.m4s" duration="2"`,
		"number no dur":    `media="$Number$.m4s"`,
		"no id":            `media="seg.m4s" duration="2"`,
	}
	for name, attrs := range cases {
		s := `<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static" mediaPresentationDuration="PT8S" minBufferTime="PT2S" profiles="x">
<Period><AdaptationSet><SegmentTemplate ` + attrs + `/><Representation id="r" bandwidth="1"/></AdaptationSet></Period></MPD>`
		md := mustReadMPD(t, s)
		_, findings := Reps(md, "https://example.com/m.mpd")
		if len(findings) != 1 || findings[0].Rule != "ADDR-TEMPLATE" {
			t.Errorf("%s: got %v", name, findings)
		}
	}
}

func TestTemplateInheritance(t *testing.T) {
	s := `<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static" mediaPresentationDuration="PT8S" minBufferTime="PT2S" profiles="x">
<Period><SegmentTemplate timescale="1000" duration="2000" startNumber="0"/>
<AdaptationSet><SegmentTemplate media="$RepresentationID$_$Number%03d$.m4s" initialization="$RepresentationID$.mp4"/>
<Representation id="a" bandwidth="1"/>
<Representation id="b" bandwidth="1"><SegmentTemplate startNumber="10"/></Representation>
</AdaptationSet></Period></MPD>`
	md := mustReadMPD(t, s)
	reps, findings := Reps(md, "https://example.com/dir/m.mpd")
	if len(findings) != 0 {
		t.Fatal(findings)
	}
	a, _ := reps[0].Segments(0, 8)
	b, _ := reps[1].Segments(0, 8)
	if len(a) != 4 || a[0].URL != "https://example.com/dir/a_000.m4s" || b[3].URL != "https://example.com/dir/b_013.m4s" {
		t.Errorf("a=%+v b=%+v", a, b)
	}
}

func TestUseOnlyProfiles(t *testing.T) {
	chunked := &Rep{LowLatency: true}
	chunked.UseOnlyProfiles()
	if chunked.LowLatency || !chunked.OnlyProfiles {
		t.Errorf("chunked without the profile: LowLatency %v, OnlyProfiles %v", chunked.LowLatency, chunked.OnlyProfiles)
	}
	ll := &Rep{LowLatency: true, Profiles: []string{ProfileLowLatency}}
	ll.UseOnlyProfiles()
	if !ll.LowLatency {
		t.Error("the low-latency profile no longer makes the Representation low latency")
	}
}
