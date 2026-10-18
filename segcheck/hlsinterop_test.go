package segcheck

import (
	"slices"
	"testing"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

func TestHLSInteropInit(t *testing.T) {
	_, video := dvbReps(t)
	iv16 := make([]byte, 16)
	cbcs := func(handler string, crypt, skip byte) *Init {
		return &Init{Handler: handler, SampleEntry: "avc1", Encrypted: true, Scheme: "cbcs",
			tenc: &mp4.TencBox{Version: 1, DefaultCryptByteBlock: crypt, DefaultSkipByteBlock: skip, DefaultConstantIV: iv16}}
	}
	role := &m.DescriptorType{SchemeIdUri: "urn:mpeg:dash:role:2011", Value: "main"}
	cases := []struct {
		desc   string
		init   *Init
		codecs string
		roles  []*m.DescriptorType
		want   []string
	}{
		{"cbcs video 1:9", cbcs("vide", 1, 9), "", nil, nil},
		{"cbcs video 5:5", cbcs("vide", 5, 5), "", nil, []string{"SEG-HLS-ENCRYPTION"}},
		{"cbcs audio 10:0", cbcs("soun", 10, 0), "", nil, nil},
		{"cbcs audio 0:0", cbcs("soun", 0, 0), "", nil, nil},
		{"cbcs per-sample IV", &Init{Handler: "vide", Encrypted: true, Scheme: "cbcs",
			tenc: &mp4.TencBox{DefaultCryptByteBlock: 1, DefaultSkipByteBlock: 9, DefaultPerSampleIVSize: 8}}, "", nil, []string{"SEG-HLS-ENCRYPTION"}},
		{"cenc 8-byte IV", &Init{Handler: "vide", Encrypted: true, Scheme: "cenc", tenc: &mp4.TencBox{DefaultPerSampleIVSize: 8}}, "", nil, nil},
		{"cenc 16-byte IV", &Init{Handler: "vide", Encrypted: true, Scheme: "cenc", tenc: &mp4.TencBox{DefaultPerSampleIVSize: 16}}, "", nil, []string{"SEG-HLS-ENCRYPTION"}},
		{"cens", &Init{Handler: "vide", Encrypted: true, Scheme: "cens", tenc: &mp4.TencBox{}}, "", nil, []string{"SEG-HLS-ENCRYPTION"}},
		{"IMSC1 text", &Init{Handler: "subt", SampleEntry: "stpp"}, "stpp.ttml.im1t", nil, nil},
		{"IMSC1 image", &Init{Handler: "subt", SampleEntry: "stpp"}, "stpp.ttml.im1i", nil, []string{"SEG-HLS-TEXT"}},
		{"WebVTT", &Init{Handler: "text", SampleEntry: "wvtt"}, "wvtt", nil, nil},
		{"role in kind box", &Init{Handler: "vide", kinds: []*mp4.KindBox{{SchemeURI: "urn:mpeg:dash:role:2011", Value: "main"}}}, "", []*m.DescriptorType{role}, nil},
		{"role without kind box", &Init{Handler: "vide"}, "", []*m.DescriptorType{role}, []string{"SEG-HLS-ROLES"}},
		{"kind box with other scheme", &Init{Handler: "vide", kinds: []*mp4.KindBox{{SchemeURI: "urn:example:kind", Value: "main"}}}, "", nil, []string{"SEG-HLS-ROLES"}},
	}
	for _, c := range cases {
		r := video
		r.Codecs, r.Roles = c.codecs, c.roles
		if got := ruleIDs(CheckHLSInteropInit(&r, c.init, r.Loc())); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.desc, got, c.want)
		}
	}
}

func TestHLSInteropMedia(t *testing.T) {
	kid1, kid2 := mp4.UUID(make([]byte, 16)), mp4.UUID(append([]byte{1}, make([]byte, 15)...))
	init := &Init{HLSInterop: true, Encrypted: true, tenc: &mp4.TencBox{DefaultKID: kid1, DefaultConstantIV: make([]byte, 16)}}
	traf := func(kid mp4.UUID, sbgp bool) *mp4.TrafBox {
		tr := &mp4.TrafBox{}
		if kid != nil {
			tr.Sgpd = &mp4.SgpdBox{GroupingType: "seig", SampleGroupEntries: []mp4.SampleGroupEntry{&mp4.SeigSampleGroupEntry{KID: kid, ConstantIV: make([]byte, 16)}}}
		}
		if sbgp {
			tr.Sbgp = &mp4.SbgpBox{GroupingType: "seig"}
		}
		return tr
	}
	file := func(trafs ...*mp4.TrafBox) *mp4.File {
		seg := &mp4.MediaSegment{}
		for _, tr := range trafs {
			seg.Fragments = append(seg.Fragments, &mp4.Fragment{Moof: &mp4.MoofBox{Trafs: []*mp4.TrafBox{tr}}})
		}
		return &mp4.File{Segments: []*mp4.MediaSegment{seg}}
	}
	cases := []struct {
		desc string
		file *mp4.File
		want []string
	}{
		{"default key", file(traf(nil, false), traf(nil, false)), nil},
		{"rotated key with sbgp", file(traf(kid2, true), traf(kid2, true)), nil},
		{"rotated key without sbgp", file(traf(kid2, false)), []string{"SEG-HLS-KEY-ROTATION"}},
		{"two keys in a segment", file(traf(nil, false), traf(kid2, true)), []string{"SEG-HLS-SEGMENT-KEY"}},
		{"senc without saio", file(&mp4.TrafBox{Senc: &mp4.SencBox{}}), []string{"SEG-HLS-ENCRYPTION"}},
	}
	for _, c := range cases {
		if got := ruleIDs(hlsMediaRules(init, c.file, report.Location{})); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.desc, got, c.want)
		}
	}
}

func TestHLSInteropSidxAndVariants(t *testing.T) {
	_, video := dvbReps(t)
	video.Dynamic = false
	refs := func(sizes ...uint32) *mp4.SidxBox {
		s := &mp4.SidxBox{Timescale: 1000}
		for _, size := range sizes {
			s.SidxRefs = append(s.SidxRefs, mp4.SidxRef{ReferencedSize: size, SubSegmentDuration: 2000})
		}
		return s
	}
	even := refs(1000, 1050, 950, 500)
	even.SidxRefs[3].SubSegmentDuration = 1000 // a short last fragment
	if f := CheckHLSInteropSidx(&video, even, 1); len(f) > 0 {
		t.Errorf("even bitrate: %v", f)
	}
	if got := ruleIDs(CheckHLSInteropSidx(&video, refs(1000, 2000, 1000, 1000), 2)); !slices.Equal(got, []string{"SEG-HLS-SIDX", "SEG-HLS-BITRATE"}) {
		t.Errorf("uneven bitrate and two sidx: got %v", got)
	}
	enc := func(handler, scheme string) *Init {
		return &Init{Handler: handler, Encrypted: true, Scheme: scheme, HLSInterop: true}
	}
	reps := []*addressing.Rep{&video, &video, &video}
	if f := hlsPresentationRules(reps, []*Init{enc("vide", "cenc"), enc("vide", "cbcs"), enc("soun", "cbcs")}); len(f) > 0 {
		t.Errorf("with cbcs variant: %v", f)
	}
	if got := ruleIDs(hlsPresentationRules(reps, []*Init{enc("vide", "cenc"), enc("soun", "cbcs"), nil})); !slices.Equal(got, []string{"SEG-HLS-CBCS-VARIANT"}) {
		t.Errorf("cenc only: got %v", got)
	}
}
