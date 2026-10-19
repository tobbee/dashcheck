package segcheck

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

const dir = "../testdata/testpic_2s"

func reps(t *testing.T) []*addressing.Rep {
	t.Helper()
	md, err := m.ReadFromFile(filepath.Join(dir, "Manifest.mpd"))
	if err != nil {
		t.Fatal(err)
	}
	r, findings := addressing.Reps(md, "file://"+dir+"/Manifest.mpd")
	if len(findings) > 0 {
		t.Fatal(findings)
	}
	return r
}

func read(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func ruleIDs(f []report.Finding) []string {
	var ids []string
	for _, x := range f {
		ids = append(ids, x.Rule)
	}
	return ids
}

func TestCleanContent(t *testing.T) {
	for _, rep := range reps(t) {
		init, f := CheckInit(read(t, rep.ID+"/init.mp4"), rep, rep.Loc())
		if len(f) > 0 {
			t.Errorf("%s init: %v", rep.ID, f)
		}
		if init == nil {
			t.Fatalf("%s: no init", rep.ID)
		}
		segs, err := rep.Segments(0, rep.PeriodDur)
		if err != nil {
			t.Fatal(err)
		}
		var prev *Media
		for i, seg := range segs {
			data := read(t, rep.ID+"/"+filepath.Base(seg.URL))
			media, f := CheckMedia(data, init, rep, seg, i == len(segs)-1)
			f = append(f, CheckContinuity(prev, media, init, rep, seg)...)
			if len(f) > 0 {
				t.Errorf("%s seg %d: %v", rep.ID, seg.Number, f)
			}
			prev = media
		}
	}
}

func TestInitCrossChecks(t *testing.T) {
	video := reps(t)[1]
	cases := []struct {
		name   string
		modify func(r *addressing.Rep)
		want   []string
	}{
		{"wrong profile in codecs", func(r *addressing.Rep) { r.Codecs = "avc1.4d401e" }, []string{"XCHK-CODECS-DETAIL"}},
		{"wrong codec", func(r *addressing.Rep) { r.Codecs = "hvc1.1.6.L93.B0" }, []string{"XCHK-CODECS"}},
		{"wrong size", func(r *addressing.Rep) { r.Width = 1280 }, []string{"XCHK-SIZE"}},
		{"wrong handler", func(r *addressing.Rep) { r.ContentType = "audio" }, []string{"XCHK-HANDLER"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := *video
			c.modify(&r)
			_, f := CheckInit(read(t, "V300/init.mp4"), &r, r.Loc())
			if got := ruleIDs(f); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v: %v", got, c.want, f)
			}
		})
	}
}

func TestMediaTimingErrors(t *testing.T) {
	video := reps(t)[1]
	init, _ := CheckInit(read(t, "V300/init.mp4"), video, video.Loc())
	segs, _ := video.Segments(0, video.PeriodDur)
	seg2 := read(t, "V300/2.m4s")

	// Segment 2 served for segment 4 is 4 s off, more than 50% of the 2 s duration.
	_, f := CheckMedia(seg2, init, video, segs[3], false)
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-EPT"}) {
		t.Errorf("wrong segment: got %v", f)
	}

	// The same content checked as a SegmentTimeline segment must match exactly.
	tl := *video
	tl.Kind = addressing.KindTimelineNumber
	seg := segs[1]
	seg.Dur++ // one tick longer than the content
	_, f = CheckMedia(seg2, init, &tl, seg, false)
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-DURATION"}) {
		t.Errorf("timeline duration: got %v", f)
	}

	// Skipping segment 3 gives a gap between 2 and 4.
	m2, _ := CheckMedia(seg2, init, video, segs[1], false)
	m4, _ := CheckMedia(read(t, "V300/4.m4s"), init, video, segs[3], false)
	f = CheckContinuity(m2, m4, init, video, segs[3])
	if got := ruleIDs(f); !slices.Equal(got, []string{"SEG-CONTINUITY"}) {
		t.Errorf("continuity: got %v", f)
	}
}

func TestIsSync(t *testing.T) {
	cases := []struct {
		flags uint32
		want  bool
	}{
		{0x02000000, true},  // depends on no other sample
		{0x00000000, true},  // dependency unknown, not marked non-sync
		{0x00010000, false}, // marked non-sync
		{0x01010000, false}, // depends on others, non-sync
	}
	for _, c := range cases {
		if got := isSync(c.flags); got != c.want {
			t.Errorf("isSync(%08x) = %v, want %v", c.flags, got, c.want)
		}
	}
}

func TestImplicitAAC(t *testing.T) {
	info := sampleEntryInfo{fourCC: "mp4a", codec: "mp4a.40.2", derived: true}
	for _, c := range []struct {
		mpd  string
		want bool
	}{{"mp4a.40.2", true}, {"mp4a.40.5", true}, {"mp4a.40.29", true}, {"mp4a.40.42", false}} {
		reason, _ := compareCodecs(c.mpd, info)
		if (reason == "") != c.want {
			t.Errorf("%s: reason %q", c.mpd, reason)
		}
	}
}

// TestFragmentContinuity joins media segments into one segment with several
// fragments, which must follow each other in decode time.
func TestFragmentContinuity(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		want  []string // messages of SEG-CONTINUITY-FRAGMENT, by prefix
	}{
		{"consecutive", []string{"1.m4s", "2.m4s", "3.m4s"}, nil},
		{"gap", []string{"1.m4s", "3.m4s"}, []string{"gap of 180000 ticks (2.000s)"}},
		{"overlap", []string{"2.m4s", "1.m4s"}, []string{"overlap of -360000 ticks (-4.000s)"}},
		{"gap, then consecutive", []string{"1.m4s", "3.m4s", "4.m4s"}, []string{"gap of 180000 ticks (2.000s)"}},
	}
	for _, rep := range reps(t) {
		if rep.ID != "V300" {
			continue
		}
		init, _ := CheckInit(read(t, "V300/init.mp4"), rep, rep.Loc())
		segs, _ := rep.Segments(0, rep.PeriodDur)
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				var data []byte
				for _, name := range c.files {
					data = append(data, read(t, "V300/"+name)...)
				}
				_, f := CheckMedia(data, init, rep, segs[0], false)
				var got []string
				for _, x := range f {
					if x.Rule == "SEG-CONTINUITY-FRAGMENT" {
						got = append(got, x.Message)
					}
				}
				if len(got) != len(c.want) {
					t.Fatalf("got %q, want %q", got, c.want)
				}
				for i := range got {
					if !strings.HasPrefix(got[i], c.want[i]) {
						t.Errorf("got %q, want prefix %q", got[i], c.want[i])
					}
				}
			})
		}
	}
}

func TestSequenceNumbers(t *testing.T) {
	rep := reps(t)[1]
	segs, _ := rep.Segments(0, rep.PeriodDur)
	check := func(opt bool, files ...string) []string {
		init, _ := CheckInit(read(t, "V300/init.mp4"), rep, rep.Loc())
		init.SequenceNumbers = opt
		var data []byte
		for _, name := range files {
			data = append(data, read(t, "V300/"+name)...)
		}
		_, f := CheckMedia(data, init, rep, segs[0], false)
		var got []string
		for _, x := range f {
			if x.Rule == "SEG-SEQUENCE-NUMBER" {
				got = append(got, x.Message)
			}
		}
		return got
	}
	if got := check(true, "1.m4s"); got != nil {
		t.Errorf("one chunk: %q", got)
	}
	if got := check(false, "1.m4s", "2.m4s"); got != nil {
		t.Errorf("not asked for: %q", got)
	}
	if got := check(true, "1.m4s", "2.m4s"); len(got) != 1 || !strings.Contains(got[0], "different sequence numbers") {
		t.Errorf("chunks with different numbers: %q", got)
	}

	init := &Init{SequenceNumbers: true, Timescale: 1}
	cases := []struct {
		prev, cur uint32
		ok        bool
	}{
		{5, 6, true},
		{5, 9, true},
		{5, 5, false},
		{6, 5, false},
		{math.MaxUint32, 0, true},
		{0, math.MaxUint32, false},
	}
	for _, c := range cases {
		f := sequenceNumberContinuity(&Media{SequenceNumber: c.prev}, &Media{SequenceNumber: c.cur}, init, rep, segs[1])
		if (len(f) == 0) != c.ok {
			t.Errorf("%d then %d: %v", c.prev, c.cur, f)
		}
	}
}
