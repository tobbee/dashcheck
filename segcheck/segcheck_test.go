package segcheck

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/Eyevinn/mp4ff/mp4"
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

// TestHEAACSampleRate checks that @audioSamplingRate may give the sampling
// rate after SBR for HE-AAC (DASH-IF IOP v4.3 6.3.2), which is twice the
// rate of the AAC core with implicit or backward-compatible signaling.
func TestHEAACSampleRate(t *testing.T) {
	audio := reps(t)[0] // AAC-LC at 48 kHz
	cases := []struct {
		codecs, rate string
		want         []string
	}{
		{"mp4a.40.2", "48000", nil},
		{"mp4a.40.2", "96000", []string{"XCHK-SAMPLERATE"}},
		{"mp4a.40.5", "96000", nil},
		{"mp4a.40.29", "96000", nil},
		{"mp4a.40.5", "48000", nil},
		{"mp4a.40.5", "44100", []string{"XCHK-SAMPLERATE"}},
	}
	for _, c := range cases {
		t.Run(c.codecs+" "+c.rate, func(t *testing.T) {
			r := *audio
			r.Codecs, r.AudioSamplingRate = c.codecs, c.rate
			_, f := CheckInit(read(t, "A48/init.mp4"), &r, r.Loc())
			if got := ruleIDs(f); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", f, c.want)
			}
		})
	}
	// Explicit hierarchical signaling gives the rate after SBR.
	explicit := sampleEntryInfo{sampleRate: 24000, aac: &audioConfig{objectType: 5, sampleRate: 22050, extRate: 44100}}
	if got := sbrRate("mp4a.40.5", explicit); got != 44100 {
		t.Errorf("explicit signaling: got %d Hz", got)
	}
}

// TestUndecodedSampleEntry checks that a sample entry that mp4ff does not
// decode, such as DTS, is reported as not supported instead of as one
// without sample rate and channels.
func TestUndecodedSampleEntry(t *testing.T) {
	audio := *reps(t)[0]
	audio.Codecs = "dtsc"
	audio.Profiles = []string{profileCMAF} // for the CMAF rules on sample rate and channels
	data := bytes.Replace(read(t, "A48/init.mp4"), []byte("mp4a"), []byte("dtsc"), 1)
	init, f := CheckInit(data, &audio, audio.Loc())
	if got := ruleIDs(f); !slices.Equal(got, []string{"INIT-UNSUPPORTED"}) || !init.CMAF {
		t.Errorf("got %v and CMAF %v, want [INIT-UNSUPPORTED] for CMAF", f, init.CMAF)
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

// TestTimelinePeriodEnd checks that the last segment of a run with a
// negative S@r may end at the Period end, before its MPD duration.
func TestTimelinePeriodEnd(t *testing.T) {
	video := reps(t)[1]
	init, _ := CheckInit(read(t, "V300/init.mp4"), video, video.Loc())
	segs, _ := video.Segments(0, video.PeriodDur)
	// A finer timescale than that of the MPD (1), to round less.
	tl := *video
	tl.Kind = addressing.KindTimelineNumber
	tl.Timescale *= 1000
	tl.PTO *= 1000
	seg := segs[1] // 2 s of content
	seg.Time *= 1000
	seg.Dur *= 1500
	start := float64(seg.Time-tl.PTO) / float64(tl.Timescale)
	cases := []struct {
		name      string
		periodEnd bool
		toEnd     float64 // from the segment start to the Period end
		want      []string
	}{
		{"ends at the Period end", true, 2, nil},
		{"ends with the media after the Period end", true, 1.5, nil},
		{"ends before the Period end", true, 2.5, []string{"SEG-DURATION"}},
		{"not the last of a negative S@r", false, 2, []string{"SEG-DURATION"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			seg.PeriodEnd = c.periodEnd
			tl.PeriodDur = start + c.toEnd
			_, f := CheckMedia(read(t, "V300/2.m4s"), init, &tl, seg, true)
			f = slices.DeleteFunc(f, func(x report.Finding) bool { return x.Rule != "SEG-DURATION" })
			if got := ruleIDs(f); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", f, c.want)
			}
		})
	}
}

// TestEditListCut checks that the media before the media time of an edit
// list, such as AAC priming, is left out of the earliest presentation time
// and the duration of the first segment.
func TestEditListCut(t *testing.T) {
	audio := reps(t)[0]
	init, _ := CheckInit(read(t, "A48/init.mp4"), audio, audio.Loc())
	segs, _ := audio.Segments(0, audio.PeriodDur)
	seg1, seg2 := read(t, "A48/1.m4s"), read(t, "A48/2.m4s")
	m1, _ := CheckMedia(seg1, init, audio, segs[0], false)
	m2, _ := CheckMedia(seg2, init, audio, segs[1], false)
	if m1.DecodeTime != 0 || m1.Cut != 0 {
		t.Fatalf("segment 1 without edit list: tfdt %d, cut %d", m1.DecodeTime, m1.Cut)
	}
	const priming = 1024
	elst := *init
	elst.HasEditList, elst.EditOffset = true, priming
	tl := *audio
	tl.Kind, tl.Timescale = addressing.KindTimelineNumber, elst.Timescale
	at := func(nr int, time, dur uint64) addressing.Segment {
		s := segs[nr-1]
		s.Time, s.Dur = time, dur
		return s
	}
	cases := []struct {
		name string
		data []byte
		seg  addressing.Segment
		cut  uint64
		want []string
	}{
		{"first segment from 0 with the presented duration", seg1, at(1, 0, m1.Duration-priming), priming, nil},
		{"first segment with the priming in its duration", seg1, at(1, 0, m1.Duration), priming, []string{"SEG-DURATION"}},
		{"second segment at tfdt minus the priming", seg2, at(2, m2.DecodeTime-priming, m2.Duration), 0, nil},
		{"second segment at tfdt", seg2, at(2, m2.DecodeTime, m2.Duration), 0, []string{"SEG-EPT"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			media, f := CheckMedia(c.data, &elst, &tl, c.seg, false)
			if got := ruleIDs(f); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", f, c.want)
			}
			if media.Cut != c.cut {
				t.Errorf("cut %d, want %d", media.Cut, c.cut)
			}
		})
	}
}

// nonSyncStart returns a media segment with its first sample marked as a
// non-sync sample.
func nonSyncStart(t *testing.T, data []byte) []byte {
	t.Helper()
	mf, err := mp4.DecodeFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	trun := mf.Segments[0].Fragments[0].Moof.Traf.Trun
	if flags, ok := trun.FirstSampleFlags(); ok {
		trun.SetFirstSampleFlags(flags | mp4.NonSyncSampleFlags)
	} else if trun.HasSampleFlags() {
		trun.Samples[0].Flags |= mp4.NonSyncSampleFlags
	} else {
		t.Fatal("no sample flags in trun")
	}
	var buf bytes.Buffer
	if err := mf.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestPartialSegments checks the rules that apply differently to Partial
// Segments of a Segment Sequence: @startWithSAP only to the first, and only
// the start of a sequence in a SegmentTimeline is exact.
func TestPartialSegments(t *testing.T) {
	video := reps(t)[1]
	init, _ := CheckInit(read(t, "V300/init.mp4"), video, video.Loc())
	segs, _ := video.Segments(0, video.PeriodDur)
	seg2 := read(t, "V300/2.m4s")
	partial := func(sub uint64) addressing.Segment {
		s := segs[1]
		s.Partial, s.SubNumber = true, sub
		return s
	}
	noSync := nonSyncStart(t, seg2)
	tl := *video
	tl.Kind = addressing.KindTimelineNumber
	ssr := func(sapType, cadence uint32) *addressing.Rep {
		r := *video
		r.SequenceProperties = []*m.SegmentSequencePropertiesType{{SapType: sapType, Cadence: cadence}}
		return &r
	}
	shift := func(s addressing.Segment, time, dur uint64) addressing.Segment {
		s.Time += time
		s.Dur += dur
		return s
	}
	cases := []struct {
		name string
		rep  *addressing.Rep
		data []byte
		seg  addressing.Segment
		want []string
	}{
		{"segment without sync start", video, noSync, segs[1], []string{"SEG-SAP"}},
		{"first Partial Segment without sync start", video, noSync, partial(0), []string{"SEG-SAP"}},
		{"later Partial Segment without sync start", video, noSync, partial(1), nil},
		{"Partial Segment with sapType 1 without sync start", ssr(1, 1), noSync, partial(1), []string{"SEG-SSR-SAP"}},
		{"Partial Segment outside the cadence", ssr(1, 2), noSync, partial(1), nil},
		{"Partial Segment at the cadence", ssr(1, 2), noSync, partial(2), []string{"SEG-SSR-SAP"}},
		{"Partial Segment with sapType 3", ssr(3, 1), noSync, partial(1), nil},
		{"first Partial Segment reported by SEG-SAP", ssr(1, 1), noSync, partial(0), []string{"SEG-SAP"}},
		{"timeline segment one tick long", &tl, seg2, shift(segs[1], 0, 1), []string{"SEG-DURATION"}},
		{"timeline Partial Segment one tick long", &tl, seg2, shift(partial(1), 0, 1), nil},
		{"timeline sequence start one tick late", &tl, seg2, shift(partial(0), 1, 0), []string{"SEG-EPT"}},
		{"timeline Partial Segment one tick late", &tl, seg2, shift(partial(1), 1, 0), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, f := CheckMedia(c.data, init, c.rep, c.seg, false)
			if got := ruleIDs(f); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", f, c.want)
			}
		})
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

// withoutTfdt returns media segments with the tfdt boxes of the given
// fragments, counted over all segments, removed, and without styp boxes if
// noStyp is set.
func withoutTfdt(t *testing.T, data []byte, noStyp bool, frags ...int) []byte {
	return withoutBox(t, data, "tfdt", noStyp, frags...)
}

// withoutBox is withoutTfdt for the tfdt or trun boxes.
func withoutBox(t *testing.T, data []byte, box string, noStyp bool, frags ...int) []byte {
	t.Helper()
	mf, err := mp4.DecodeFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	i := 0
	for _, ms := range mf.Segments {
		if noStyp {
			ms.Styp = nil
		}
		for _, frag := range ms.Fragments {
			if slices.Contains(frags, i) {
				traf := frag.Moof.Traf
				switch box {
				case "tfdt":
					traf.Tfdt = nil
				case "trun":
					traf.Trun, traf.Truns = nil, nil
				}
				traf.Children = slices.DeleteFunc(traf.Children, func(b mp4.Box) bool { return b.Type() == box })
			}
			i++
		}
	}
	var buf bytes.Buffer
	if err := mf.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestNoTfdt checks that a tfdt box is required in CMAF tracks, in segments
// with the msdh brand, and in the first fragment of a segment with
// SegmentTimeline and $Time$ but no sidx, and that a fragment without one
// continues the previous fragment, while the start of a segment without one
// is not checked.
func TestNoTfdt(t *testing.T) {
	video := reps(t)[1]
	init, _ := CheckInit(read(t, "V300/init.mp4"), video, video.Loc())
	segs, _ := video.Segments(0, video.PeriodDur)
	seg1, seg2, seg3, seg4 := read(t, "V300/1.m4s"), read(t, "V300/2.m4s"), read(t, "V300/3.m4s"), read(t, "V300/4.m4s")
	cat := func(parts ...[]byte) []byte { return bytes.Join(parts, nil) }
	var sidx bytes.Buffer
	if err := (&mp4.SidxBox{ReferenceID: 1, Timescale: 90000, SidxRefs: []mp4.SidxRef{{}}}).Encode(&sidx); err != nil {
		t.Fatal(err)
	}
	cmaf := *init
	cmaf.CMAF = true
	hls := *init
	hls.HLSInterop = true
	elst := *init
	elst.HasEditList = true
	chosen := *video
	chosen.OnlyProfiles = true
	tl := *video
	tl.Kind = addressing.KindTimelineNumber
	tt := *video
	tt.Kind = addressing.KindTimelineTime
	longer := segs[1]
	longer.Dur++
	part := segs[1]
	part.Partial, part.SubNumber = true, 1
	cases := []struct {
		name string
		init *Init
		rep  *addressing.Rep
		data []byte
		seg  addressing.Segment
		want []string
	}{
		{"msdh brand", init, video, withoutTfdt(t, seg2, false, 0), segs[1], []string{"SEG-TFDT"}},
		{"no styp", init, video, withoutTfdt(t, seg2, true, 0), segs[1], []string{"SEG-START-UNKNOWN"}},
		{"msdh brand with chosen profiles", init, &chosen, withoutTfdt(t, seg2, false, 0), segs[1], []string{"SEG-START-UNKNOWN"}},
		{"CMAF track", &cmaf, video, withoutTfdt(t, seg2, true, 0), segs[1], []string{"SEG-TFDT"}},
		{"CTA-5005 content", &hls, video, withoutTfdt(t, seg2, true, 0), segs[1], []string{"SEG-TFDT"}},
		{"timeline with $Time$", init, &tt, withoutTfdt(t, seg2, true, 0), segs[1], []string{"SEG-TFDT"}},
		{"timeline with $Time$ and sidx", init, &tt, cat(sidx.Bytes(), withoutTfdt(t, seg2, true, 0)), segs[1], []string{"SEG-START-UNKNOWN"}},
		{"timeline with $Time$, second fragment", init, &tt, withoutTfdt(t, cat(seg1, seg2), true, 1), segs[0], nil},
		{"timeline with $Time$, later Partial Segment", init, &tt, withoutTfdt(t, seg2, true, 0), part, []string{"SEG-START-UNKNOWN"}},
		{"start not checked", init, video, withoutTfdt(t, seg2, true, 0), segs[3], []string{"SEG-START-UNKNOWN"}},
		{"timeline duration checked", init, &tl, withoutTfdt(t, seg2, true, 0), longer, []string{"SEG-START-UNKNOWN", "SEG-DURATION"}},
		{"timeline duration with edit list", &elst, &tl, withoutTfdt(t, seg2, true, 0), longer, []string{"SEG-START-UNKNOWN"}},
		{"second fragment continues the first", init, video, withoutTfdt(t, cat(seg1, seg3), false, 1), segs[0], []string{"SEG-TFDT"}},
		{"third fragment after an inferred one", init, video, withoutTfdt(t, cat(seg1, seg2, seg3), false, 1), segs[0], []string{"SEG-TFDT"}},
		{"later tfdt after a relative start", init, video, withoutTfdt(t, cat(seg1, seg3), true, 0), segs[0], []string{"SEG-START-UNKNOWN"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, f := CheckMedia(c.data, c.init, c.rep, c.seg, false)
			// Segments of several fragments are longer than one segment, and the
			// segments are not CMAF fragments.
			f = slices.DeleteFunc(f, func(x report.Finding) bool {
				return x.Rule == "SEG-DURATION" && c.seg == segs[0] || strings.HasPrefix(x.Rule, "SEG-CMAF-")
			})
			if got := ruleIDs(f); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", f, c.want)
			}
		})
	}

	// The decode times of a fragment without tfdt continue the previous fragment.
	m23, _ := CheckMedia(cat(seg2, seg3), init, video, segs[1], false)
	inferred, _ := CheckMedia(withoutTfdt(t, cat(seg2, seg3), false, 1), init, video, segs[1], false)
	if inferred.StartUnknown || inferred.DecodeTime != m23.DecodeTime || inferred.EPT != m23.EPT || inferred.Duration != m23.Duration {
		t.Errorf("second fragment without tfdt: got %+v, want %+v", inferred, m23)
	}
	relative, _ := CheckMedia(withoutTfdt(t, seg2, true, 0), init, video, segs[1], false)
	if !relative.StartUnknown || relative.DecodeTime != 0 || !math.IsNaN(relative.PeriodEPT(video)) {
		t.Errorf("first fragment without tfdt: start unknown %v, decode time %d, Period EPT %f",
			relative.StartUnknown, relative.DecodeTime, relative.PeriodEPT(video))
	}

	// Segment 4 without tfdt continues segment 2, unlike segment 4 with it.
	m2, _ := CheckMedia(seg2, init, video, segs[1], false)
	if got, want := m2.PeriodEPT(video), m2.EPT-float64(video.PTO)/float64(video.Timescale); got != want {
		t.Errorf("Period EPT of segment 2: got %f, want %f", got, want)
	}
	m4, _ := CheckMedia(withoutTfdt(t, seg4, true, 0), init, video, segs[3], false)
	if f := CheckContinuity(m2, m4, init, video, segs[3]); len(f) > 0 {
		t.Errorf("continuity to a segment without tfdt: %v", f)
	}
	if f := CheckContinuity(m4, m2, init, video, segs[1]); len(f) > 0 {
		t.Errorf("continuity from a segment without tfdt: %v", f)
	}
}

// TestNoTrun checks that a traf without trun, which has no samples, is
// allowed outside CMAF, and that the next fragment continues it.
func TestNoTrun(t *testing.T) {
	video := reps(t)[1]
	init, _ := CheckInit(read(t, "V300/init.mp4"), video, video.Loc())
	segs, _ := video.Segments(0, video.PeriodDur)
	seg1, seg2 := read(t, "V300/1.m4s"), read(t, "V300/2.m4s")
	// The fragment without trun starts where segment 1 ends, and so does segment 2.
	media, f := CheckMedia(withoutBox(t, bytes.Join([][]byte{seg1, seg2, seg2}, nil), "trun", false, 1), init, video, segs[0], false)
	// The segment is longer than one segment.
	f = slices.DeleteFunc(f, func(x report.Finding) bool { return x.Rule == "SEG-DURATION" })
	if len(f) > 0 {
		t.Errorf("fragment without trun: %v", f)
	}
	m12, _ := CheckMedia(bytes.Join([][]byte{seg1, seg2}, nil), init, video, segs[0], false)
	if media == nil || media.Samples != m12.Samples || media.Duration != m12.Duration {
		t.Errorf("fragment without trun: got %+v, want the samples of %+v", media, m12)
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
