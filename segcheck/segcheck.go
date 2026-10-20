// Package segcheck checks initialization and media segments with mp4ff,
// and cross-checks them against the MPD.
package segcheck

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

const (
	refCMAF         = "ISO/IEC 23000-19"
	refStartWithSAP = "ISO/IEC 23009-1 5.3.7 (@startWithSAP)"
	refTimeline     = "ISO/IEC 23009-1 5.3.9.6"
	refNumber       = "DASH-IF Timing Model 18.4"
	refSequence     = "ISO/IEC 23009-1 5.3.9.7.1"
)

var (
	RuleInitParse = report.Register(report.Rule{
		ID: "INIT-PARSE", Severity: report.Error,
		Desc: "The initialization segment must be a parseable ISOBMFF file",
	})
	RuleInitStructure = report.Register(report.Rule{
		ID: "INIT-STRUCTURE", Severity: report.Error,
		Desc: "The initialization segment must have ftyp and moov with mvex and a track", SpecRef: refCMAF,
	})
	RuleInitOneTrack = report.Register(report.Rule{
		ID: "INIT-ONE-TRACK", Severity: report.Warning,
		Desc: "The initialization segment should have exactly one track", SpecRef: refCMAF,
	})
	RuleInitSamples = report.Register(report.Rule{
		ID: "INIT-SAMPLES", Severity: report.Warning,
		Desc: "The initialization segment should not describe any samples", SpecRef: refCMAF,
	})
	RuleCodecs = report.Register(report.Rule{
		ID: "XCHK-CODECS", Severity: report.Error,
		Desc: "@codecs must match the sample entry type",
	})
	RuleCodecsDetail = report.Register(report.Rule{
		ID: "XCHK-CODECS-DETAIL", Severity: report.Warning,
		Desc: "@codecs should match the codec string derived from the decoder configuration",
	})
	RuleSize = report.Register(report.Rule{
		ID: "XCHK-SIZE", Severity: report.Warning,
		Desc: "@width and @height should match the sample entry",
	})
	RuleSampleRate = report.Register(report.Rule{
		ID: "XCHK-SAMPLERATE", Severity: report.Warning,
		Desc: "@audioSamplingRate should match the sample entry",
	})
	RuleHandler = report.Register(report.Rule{
		ID: "XCHK-HANDLER", Severity: report.Error,
		Desc: "The track handler must match @contentType or @mimeType",
	})
	RuleSegParse = report.Register(report.Rule{
		ID: "SEG-PARSE", Severity: report.Error,
		Desc: "The media segment must be parseable ISOBMFF",
	})
	RuleSegStructure = report.Register(report.Rule{
		ID: "SEG-STRUCTURE", Severity: report.Error,
		Desc: "A media segment must have moof with traf, tfdt and trun, and mdat", SpecRef: refCMAF,
	})
	RuleSegTrackID = report.Register(report.Rule{
		ID: "SEG-TRACKID", Severity: report.Error,
		Desc: "tfhd@track_ID must match the track in the initialization segment",
	})
	RuleSegEPT = report.Register(report.Rule{
		ID: "SEG-EPT", Severity: report.Error,
		Desc: "The segment's earliest presentation time must match the MPD",
	})
	RuleSegEPTCTO = report.Register(report.Rule{
		ID: "SEG-EPT-CTO", Severity: report.Warning,
		Desc: "tfdt matches the MPD, but composition time offsets without an edit list shift the earliest presentation time",
	})
	RuleSegDuration = report.Register(report.Rule{
		ID: "SEG-DURATION", Severity: report.Error,
		Desc: "The segment duration must match the MPD",
	})
	RuleSegContinuity = report.Register(report.Rule{
		ID: "SEG-CONTINUITY", Severity: report.Error,
		Desc: "Consecutive segments must have no gaps or overlaps in decode time", SpecRef: refCMAF + " 7.3.2.2",
	})
	RuleSegFragmentContinuity = report.Register(report.Rule{
		ID: "SEG-CONTINUITY-FRAGMENT", Severity: report.Error,
		Desc:    "Each chunk or fragment in a segment must have the tfdt of the previous one plus its sample durations",
		SpecRef: refCMAF + " 7.3.2.2 c, 7.3.2.3 g",
	})
	RuleSidxSpan = report.Register(report.Rule{
		ID: "SEG-SIDX-SPAN", Severity: report.Error,
		Desc:    "The sidx box of an indexed segment must index all of it: no movie fragment before its first or after its last subsegment",
		SpecRef: "ISO/IEC 23009-1 6.3.4.4, 8.12.4.2",
	})
	RuleSegSAP = report.Register(report.Rule{
		ID: "SEG-SAP", Severity: report.Error,
		Desc:    "With @startWithSAP 1 or 2, every segment, or the first Partial Segment of each Segment Sequence, must start with a sync sample",
		SpecRef: refStartWithSAP,
	})
)

// Init holds what is needed from an initialization segment to check media segments.
type Init struct {
	TrackID     uint32
	Timescale   uint32
	Handler     string
	SampleEntry string
	Codec       string
	EditOffset  int64 // media time of the first edit, subtracted from composition times
	HasEditList bool
	Video       videoConfig
	Brands      []string // ftyp major and compatible brands
	Encrypted   bool
	Scheme      string   // protection scheme type
	CMAF        bool     // the track is declared as CMAF
	DefaultKID  string   // tenc default KID of encrypted content
	PsshSystems []string // system IDs of pssh boxes in moov
	Channels    int      // audio channel count in the sample entry
	tkhdWidth   uint32   // tkhd width and height, rounded to integers
	tkhdHeight  uint32
	cmafBrand   string // the highest CMAF media profile brand of the track
	trex        *mp4.TrexBox
	aac         *audioConfig // AAC AudioSpecificConfig, if any
	tenc        *mp4.TencBox // track encryption defaults, if encrypted
	dac4        *mp4.Dac4Box
	kinds       []*mp4.KindBox
	// HLSInterop turns on the CTA-5005 checks in CheckMedia; the runner sets it.
	HLSInterop bool
	// SequenceNumbers turns on the sequence number convention of
	// SEG-SEQUENCE-NUMBER (-sequence-numbers); the runner sets it.
	SequenceNumbers bool
}

// CheckInit parses and checks an initialization segment.
func CheckInit(data []byte, rep *addressing.Rep, loc report.Location) (*Init, []report.Finding) {
	var f []report.Finding
	mf, err := mp4.DecodeFile(bytes.NewReader(data))
	if err != nil {
		return nil, append(f, RuleInitParse.New(loc, "%v", err))
	}
	if mf.Init == nil || mf.Init.Moov == nil {
		return nil, append(f, RuleInitStructure.New(loc, "no moov box"))
	}
	if mf.Init.Ftyp == nil {
		f = append(f, RuleInitStructure.New(loc, "no ftyp box"))
	}
	moov := mf.Init.Moov
	if moov.Trak == nil {
		return nil, append(f, RuleInitStructure.New(loc, "no trak box"))
	}
	if len(moov.Traks) > 1 {
		f = append(f, RuleInitOneTrack.New(loc, "%d tracks", len(moov.Traks)))
	}
	trak := moov.Trak
	if trak.Mdia == nil || trak.Mdia.Mdhd == nil || trak.Mdia.Minf == nil || trak.Mdia.Minf.Stbl == nil {
		return nil, append(f, RuleInitStructure.New(loc, "incomplete trak box"))
	}
	init := &Init{
		TrackID:    trak.Tkhd.TrackID,
		Timescale:  trak.Mdia.Mdhd.Timescale,
		tkhdWidth:  uint32((uint64(trak.Tkhd.Width) + 0x8000) >> 16),
		tkhdHeight: uint32((uint64(trak.Tkhd.Height) + 0x8000) >> 16),
	}
	if trak.Mdia.Hdlr != nil {
		init.Handler = trak.Mdia.Hdlr.HandlerType
	}
	if moov.Mvex == nil {
		f = append(f, RuleInitStructure.New(loc, "no mvex box, so the content is not fragmented"))
	} else {
		for _, trex := range moov.Mvex.Trexs {
			if trex.TrackID == init.TrackID {
				init.trex = trex
			}
		}
		if init.trex == nil {
			f = append(f, RuleInitStructure.New(loc, "no trex box for track %d", init.TrackID))
		}
	}
	if trak.Edts != nil && len(trak.Edts.Elst) > 0 {
		init.HasEditList = true
		for _, e := range trak.Edts.Elst[0].Entries {
			if e.MediaTime >= 0 {
				init.EditOffset = e.MediaTime
				break
			}
		}
	}
	stbl := trak.Mdia.Minf.Stbl
	if stbl.Stsz != nil && stbl.Stsz.SampleNumber > 0 {
		f = append(f, RuleInitSamples.New(loc, "stsz lists %d samples", stbl.Stsz.SampleNumber))
	}
	info, err := describeSampleEntry(stbl.Stsd)
	if err != nil {
		return init, append(f, RuleInitStructure.New(loc, "%v", err))
	}
	init.SampleEntry = info.fourCC
	init.Codec = info.codec
	init.Video = info.video
	init.Encrypted = info.encrypted
	init.Scheme = info.scheme
	if ftyp := mf.Init.Ftyp; ftyp != nil {
		init.Brands = append([]string{ftyp.MajorBrand()}, ftyp.CompatibleBrands()...)
	}
	init.CMAF = isCMAF(rep, init.Brands)
	init.DefaultKID = info.defaultKID
	init.Channels = int(info.channels)
	init.aac = info.aac
	init.tenc = info.tenc
	init.dac4 = info.dac4
	if trak.Udta != nil {
		for _, c := range trak.Udta.Children {
			if k, ok := c.(*mp4.KindBox); ok {
				init.kinds = append(init.kinds, k)
			}
		}
	}
	for _, p := range moov.Psshs {
		init.PsshSystems = append(init.PsshSystems, p.SystemID.String())
	}
	f = append(f, crossCheckInit(rep, init, info, loc)...)
	f = append(f, iopInitRules(rep, init, loc)...)
	f = append(f, llInitRules(rep, init, loc)...)
	f = append(f, dvbInitRules(rep, init, info, loc)...)
	f = append(f, cmafInitRules(rep, init, info, mf.Init.Children, loc)...)
	f = append(f, cmafBrandInitRules(init, mf.Init.Ftyp, loc)...)
	init.cmafBrand = mediaProfileBrand(init.Brands, init.Handler, info.fourCC)
	if init.CMAF {
		f = append(f, cmafMediaProfileRules(rep, init, info, loc)...)
	}
	return init, f
}

func crossCheckInit(rep *addressing.Rep, init *Init, info sampleEntryInfo, loc report.Location) []report.Finding {
	var f []report.Finding
	if reason, major := compareCodecs(rep.Codecs, info); reason != "" {
		if major {
			f = append(f, RuleCodecs.New(loc, "%s", reason))
		} else {
			f = append(f, RuleCodecsDetail.New(loc, "%s", reason))
		}
	}
	kind := rep.ContentType
	if kind == "" {
		kind, _, _ = strings.Cut(rep.MimeType, "/")
	}
	wantHandler := map[string][]string{"video": {"vide"}, "audio": {"soun"}, "text": {"subt", "text", "sbtl"}}
	if want, ok := wantHandler[kind]; ok && init.Handler != "" && !contains(want, init.Handler) {
		f = append(f, RuleHandler.New(loc, "%s Representation but handler %q", kind, init.Handler))
	}
	if info.width != 0 && rep.Width != 0 && (rep.Sar == "" || rep.Sar == "1:1") {
		if uint32(info.width) != rep.Width || uint32(info.height) != rep.Height {
			f = append(f, RuleSize.New(loc, "MPD says %dx%d but sample entry is %dx%d",
				rep.Width, rep.Height, info.width, info.height))
		}
	}
	if info.sampleRate != 0 && rep.AudioSamplingRate != "" {
		first, _, _ := strings.Cut(rep.AudioSamplingRate, " ")
		if sr, err := strconv.Atoi(first); err == nil && uint32(sr) != info.sampleRate {
			f = append(f, RuleSampleRate.New(loc, "MPD says %d Hz but sample entry has %d Hz", sr, info.sampleRate))
		}
	}
	return f
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Media summarizes a checked media segment.
type Media struct {
	DecodeTime uint64 // tfdt of the first fragment, media timescale
	// SequenceNumber is the mfhd sequence_number of the first fragment.
	SequenceNumber uint32
	Duration       uint64  // sum of sample durations, media timescale
	EPT            float64 // earliest presentation time in seconds, before presentationTimeOffset
	Samples        int
	Size           int // bytes
	// Partial is set for a Partial Segment of a Segment Sequence, and
	// SubNumber is its position in the sequence, from 0.
	Partial   bool
	SubNumber uint64
	FirstCTO  int32 // composition time offset of the first sample
	FirstSync bool  // the first sample is a sync sample
	// Event messages before the first moof and later in the segment, as
	// "scheme value id" keys.
	EarlyEmsgs, LateEmsgs []string
}

// End returns the decode time after the last sample.
func (m *Media) End() uint64 { return m.DecodeTime + m.Duration }

// CheckMedia parses and checks a media segment against the MPD and its init segment.
// last marks the final segment of a Period, which may be shorter than nominal.
func CheckMedia(data []byte, init *Init, rep *addressing.Rep, seg addressing.Segment, last bool) (*Media, []report.Finding) {
	loc := rep.SegLoc(seg)
	var f []report.Finding
	mf, err := mp4.DecodeFile(bytes.NewReader(data))
	if err != nil {
		return nil, append(f, RuleSegParse.New(loc, "%v", err))
	}
	if len(mf.Segments) == 0 || len(mf.Segments[0].Fragments) == 0 {
		return nil, append(f, RuleSegStructure.New(loc, "no moof box"))
	}
	media := &Media{Size: len(data), Partial: seg.Partial, SubNumber: seg.SubNumber}
	minCT := int64(math.MaxInt64)
	firstSample := true
	wrongTrack := false
	// The decode time where the next fragment (or chunk) must start: the
	// previous one's tfdt plus its sample durations. Not known after a
	// fragment that could not be read.
	var next uint64
	haveNext := false
	for si, ms := range mf.Segments {
		for fi, frag := range ms.Fragments {
			floc := loc
			floc.Path = fmt.Sprintf("segment[%d]/fragment[%d]", si, fi)
			expect, check := next, haveNext
			haveNext = false
			if frag.Moof == nil || frag.Moof.Traf == nil {
				f = append(f, RuleSegStructure.New(floc, "no traf box"))
				continue
			}
			if frag.Mdat == nil {
				f = append(f, RuleSegStructure.New(floc, "no mdat box"))
			}
			traf := frag.Moof.Traf
			if traf.Tfhd != nil && traf.Tfhd.TrackID != init.TrackID {
				f = append(f, RuleSegTrackID.New(floc, "track_ID %d, init has %d", traf.Tfhd.TrackID, init.TrackID))
				wrongTrack = true
				continue
			}
			if traf.Tfdt == nil {
				f = append(f, RuleSegStructure.New(floc, "no tfdt box"))
				continue
			}
			if traf.Trun == nil {
				f = append(f, RuleSegStructure.New(floc, "no trun box"))
				continue
			}
			tfdt := traf.Tfdt.BaseMediaDecodeTime()
			if check && tfdt != expect {
				diff := int64(tfdt) - int64(expect)
				kind := "gap"
				if diff < 0 {
					kind = "overlap"
				}
				f = append(f, RuleSegFragmentContinuity.New(floc,
					"%s of %d ticks (%.3fs): tfdt %d, but the previous fragment ends at %d",
					kind, diff, float64(diff)/float64(init.Timescale), tfdt, expect))
			}
			if firstSample {
				media.DecodeTime = tfdt
			}
			samples, err := frag.GetFullSamples(init.trex)
			if err != nil {
				f = append(f, RuleSegParse.New(floc, "samples: %v", err))
				continue
			}
			next, haveNext = tfdt, true
			for _, s := range samples {
				next += uint64(s.Dur)
				if firstSample {
					firstSample = false
					media.FirstCTO = s.CompositionTimeOffset
					media.FirstSync = isSync(s.Flags)
					// @startWithSAP applies to the first Partial Segment of a Segment Sequence (ISO/IEC 23009-1 5.3.5.7).
					sapRequired := (rep.StartWithSAP == 1 || rep.StartWithSAP == 2) && seg.SubNumber == 0
					if sapRequired && init.Handler == "vide" && !isSync(s.Flags) {
						f = append(f, RuleSegSAP.New(loc, "first sample is not a sync sample (startWithSAP=%d)", rep.StartWithSAP))
					}
				}
				minCT = min(minCT, int64(s.DecodeTime)+int64(s.CompositionTimeOffset))
				media.Duration += uint64(s.Dur)
				media.Samples++
			}
		}
	}
	media.EarlyEmsgs, media.LateEmsgs = emsgPlacement(mf)
	var otherSeqNr string
	media.SequenceNumber, otherSeqNr = firstSequenceNumber(mf)
	f = append(f, sequenceNumberMediaRules(init, otherSeqNr, media.SequenceNumber, loc)...)
	if media.Samples == 0 {
		if !wrongTrack {
			f = append(f, RuleSegStructure.New(loc, "no samples"))
		}
		return nil, f
	}
	mediaTS := float64(init.Timescale)
	media.EPT = float64(minCT-init.EditOffset) / mediaTS
	f = append(f, checkTiming(rep, seg, media, mediaTS, last, loc)...)
	f = append(f, iopMediaRules(rep, init, media, loc)...)
	f = append(f, cmafMediaRules(rep, init, media, mf, loc)...)
	f = append(f, cmafBrandMediaRules(rep, init, mf, loc)...)
	f = append(f, llMediaRules(rep, media, mf, mediaTS, last, loc)...)
	f = append(f, dvbMediaRules(rep, init, media, mf, mediaTS, last, loc)...)
	f = append(f, hlsMediaRules(init, mf, loc)...)
	f = append(f, ac4MediaRules(init, mf, loc)...)
	return media, f
}

// isSync reports whether sample flags mark a sync sample. In ISO/IEC 14496-12
// sample_is_non_sync_sample carries the same information as the sync sample
// table, so it alone decides. (mp4ff's IsSync also requires sample_depends_on
// to be 2, which rejects sync samples that leave sample_depends_on unknown.)
func isSync(flags uint32) bool {
	return flags&mp4.NonSyncSampleFlags == 0
}

// checkTiming compares segment time and duration with the MPD. A
// SegmentTimeline gives exact times, and @duration nominal ones. The MPD
// times of Partial Segments in a Segment Sequence are nominal, except for the
// start of the sequence (ISO/IEC 23009-1 5.3.9.7.1).
func checkTiming(rep *addressing.Rep, seg addressing.Segment, media *Media, mediaTS float64, last bool, loc report.Location) []report.Finding {
	var f []report.Finding
	mpdTS := float64(rep.Timescale)
	wantStart := float64(seg.Time) / mpdTS
	wantDur := float64(seg.Dur) / mpdTS
	gotDur := float64(media.Duration) / mediaTS
	var exactStart, exactDur bool
	ref := refNumber
	switch rep.Kind {
	case addressing.KindTimelineTime, addressing.KindTimelineNumber:
		exactStart, exactDur = seg.SubNumber == 0, !seg.Partial
		ref = refSequence
	case addressing.KindNumber:
		if seg.Partial {
			ref = refSequence
		}
	default:
		return nil
	}
	// Exact, up to rounding to the nearest tick in each timescale.
	exact := 0.5/mpdTS + 0.5/mediaTS
	// Nominal boundaries may deviate by up to half the nominal duration.
	nominal := wantDur / 2
	if exactStart {
		decodeStart := float64(media.DecodeTime) / mediaTS
		switch {
		case math.Abs(media.EPT-wantStart) <= exact:
		case math.Abs(decodeStart-wantStart) <= exact:
			f = append(f, ruleWithRef(RuleSegEPTCTO, refTimeline).New(loc,
				"tfdt matches SegmentTimeline, but the earliest presentation time is %.6fs later",
				media.EPT-wantStart))
		default:
			f = append(f, ruleWithRef(RuleSegEPT, refTimeline).New(loc,
				"earliest presentation time %.6fs (tfdt %d) but SegmentTimeline says %.6fs",
				media.EPT, media.DecodeTime, wantStart))
		}
	} else if math.Abs(media.EPT-wantStart) > nominal {
		f = append(f, ruleWithRef(RuleSegEPT, ref).New(loc,
			"earliest presentation time %.3fs is more than 50%% of the segment duration from the nominal %.3fs",
			media.EPT, wantStart))
	}
	switch {
	case exactDur && math.Abs(gotDur-wantDur) > exact:
		f = append(f, ruleWithRef(RuleSegDuration, refTimeline).New(loc,
			"duration %.6fs but SegmentTimeline says %.6fs", gotDur, wantDur))
	case !exactDur && math.Abs(gotDur-wantDur) > nominal && !(last && gotDur < wantDur):
		f = append(f, ruleWithRef(RuleSegDuration, ref).New(loc,
			"duration %.3fs is more than 50%% from the nominal %.3fs", gotDur, wantDur))
	}
	return f
}

// ruleWithRef returns a copy of the rule with a specific spec reference.
func ruleWithRef(r *report.Rule, ref string) *report.Rule {
	c := *r
	c.SpecRef = ref
	return &c
}

// CheckContinuity verifies that cur directly follows prev in decode time.
func CheckContinuity(prev, cur *Media, init *Init, rep *addressing.Rep, seg addressing.Segment) []report.Finding {
	if prev == nil || cur == nil {
		return nil
	}
	f := emsgRepeatRules(prev, cur, init, rep, seg)
	f = append(f, sequenceNumberContinuity(prev, cur, init, rep, seg)...)
	if prev.End() == cur.DecodeTime {
		return f
	}
	diff := int64(cur.DecodeTime) - int64(prev.End())
	kind := "gap"
	if diff < 0 {
		kind = "overlap"
	}
	return append(f, RuleSegContinuity.New(rep.SegLoc(seg),
		"%s of %d ticks (%.3fs): previous segment ends at %d, this starts at %d",
		kind, diff, float64(diff)/float64(init.Timescale), prev.End(), cur.DecodeTime))
}
