// Package segcheck checks initialization and media segments with mp4ff,
// and cross-checks them against the MPD.
package segcheck

import (
	"bytes"
	"fmt"
	"math"
	"slices"
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
	RuleInitUnsupported = report.Register(report.Rule{
		ID: "INIT-UNSUPPORTED", Severity: report.Info,
		Desc: "Audio and video sample entries that dashcheck cannot decode, such as DTS, are not checked",
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
		Desc: "@audioSamplingRate should match the sample entry, or for HE-AAC the sampling rate after SBR",
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
		Desc: "A media segment must have moof with traf and trun, and mdat", SpecRef: "ISO/IEC 23009-1 6.3.5.2",
	})
	RuleSegTfdt = report.Register(report.Rule{
		ID: "SEG-TFDT", Severity: report.Error,
		Desc:    "A traf must have a tfdt box in a CMAF track, in a segment with the msdh brand, and in the first fragment of a segment with SegmentTimeline and $Time$ but no sidx",
		SpecRef: refCMAF + " Table 5, ISO/IEC 23009-1 6.3.5.3, 7.3.5",
	})
	RuleSegStartUnknown = report.Register(report.Rule{
		ID: "SEG-START-UNKNOWN", Severity: report.Info,
		Desc:    "A segment whose first fragment has no tfdt box starts where the previous segment ended, so its start is not checked",
		SpecRef: "ISO/IEC 14496-12 8.8.12",
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
		SpecRef: "ISO/IEC 23009-1 6.3.5.4, 8.12.4.2",
	})
	RuleSegSAP = report.Register(report.Rule{
		ID: "SEG-SAP", Severity: report.Error,
		Desc:    "With @startWithSAP 1 or 2, every segment, or the first Partial Segment of each Segment Sequence, must start with a sync sample",
		SpecRef: refStartWithSAP,
	})
)

// sampleEntry is the type of a sample entry and whether it is protected (has sinf).
type sampleEntry struct {
	fourCC    string
	protected bool
}

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
	aac         *audioConfig                // AAC AudioSpecificConfig, if any
	tenc        *mp4.TencBox                // track encryption defaults, if encrypted
	pssh        []*mp4.PsshBox              // pssh boxes in moov
	entries     []sampleEntry               // by sample description index minus one
	seig        []*mp4.SeigSampleGroupEntry // seig sample group entries in stbl
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
	if info.undecoded && (init.Handler == "soun" || init.Handler == "vide") {
		f = append(f, RuleInitUnsupported.New(loc, "%s sample entries are not supported, so their parameters are not checked", info.fourCC))
	}
	init.SampleEntry = info.fourCC
	init.Codec = info.codec
	init.Video = info.video
	init.Encrypted = info.encrypted
	if stbl.Stsd != nil {
		for _, c := range stbl.Stsd.Children {
			init.entries = append(init.entries, sampleEntry{c.Type(), entrySinf(c) != nil})
		}
	}
	init.seig = seigEntries(stbl.Sgpds)
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
	init.pssh = moov.Psshs
	for _, p := range moov.Psshs {
		init.PsshSystems = append(init.PsshSystems, p.SystemID.String())
	}
	f = append(f, crossCheckInit(rep, init, info, loc)...)
	f = append(f, protectionInitRules(rep, init, loc)...)
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
		sbr := sbrRate(rep.Codecs, info)
		if sr, err := strconv.Atoi(first); err == nil && uint32(sr) != info.sampleRate && uint32(sr) != sbr {
			msg := fmt.Sprintf("MPD says %d Hz but sample entry has %d Hz", sr, info.sampleRate)
			if sbr != 0 {
				msg += fmt.Sprintf(", and %d Hz after SBR", sbr)
			}
			f = append(f, RuleSampleRate.New(loc, "%s", msg))
		}
	}
	return f
}

// sbrRate returns the sampling rate of HE-AAC after SBR, which
// @audioSamplingRate signals (DASH-IF IOP v4.3 6.3.2), or 0 for other audio.
// Explicit hierarchical signaling gives the rate; otherwise SBR doubles the
// rate of the AAC core.
func sbrRate(codecs string, info sampleEntryInfo) uint32 {
	if info.aac != nil && info.aac.extRate != 0 {
		return uint32(info.aac.extRate)
	}
	if codecs != "mp4a.40.5" && codecs != "mp4a.40.29" {
		return 0
	}
	if info.aac != nil {
		return 2 * uint32(info.aac.sampleRate)
	}
	return 2 * info.sampleRate
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
	// StartUnknown is set when the first fragment has no tfdt box. The
	// segment then starts where the previous one ended, and DecodeTime is 0
	// and EPT relative to the start of the segment.
	StartUnknown bool
	// SequenceNumber is the mfhd sequence_number of the first fragment.
	SequenceNumber uint32
	Duration       uint64  // sum of sample durations, media timescale
	EPT            float64 // earliest presentation time in seconds, before presentationTimeOffset
	Cut            uint64  // start before the media time of the edit list (audio priming), not presented
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

// Presented returns the duration of the presented part, media timescale.
func (m *Media) Presented() uint64 { return m.Duration - m.Cut }

// PeriodEPT returns the earliest presentation time in the Period in seconds,
// or NaN when the start of the segment is not known.
func (m *Media) PeriodEPT(rep *addressing.Rep) float64 {
	if m.StartUnknown || rep.Timescale == 0 {
		return math.NaN()
	}
	return m.EPT - float64(rep.PTO)/float64(rep.Timescale)
}

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
	// With SegmentTimeline and $Time$ and without sidx, the tfdt of the first
	// fragment gives the start of the segment (ISO/IEC 23009-1 7.3.5).
	timeStart := rep.Kind == addressing.KindTimelineTime && seg.SubNumber == 0 && !hasSidx(mf)
	for si, ms := range mf.Segments {
		for fi, frag := range ms.Fragments {
			floc := loc
			floc.Path = fmt.Sprintf("segment[%d]/fragment[%d]", si, fi)
			expect, haveExpect := next, haveNext
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
			if traf.Trun == nil {
				f = append(f, RuleSegStructure.New(floc, "no trun box"))
				continue
			}
			// base is the decode time of the first sample. mp4ff starts the
			// samples of a traf without tfdt at 0, and offset moves them to base.
			var base, offset uint64
			known := true
			if traf.Tfdt != nil {
				base = traf.Tfdt.BaseMediaDecodeTime()
				// After a relative start, the previous fragment's end is relative too.
				if haveExpect && !media.StartUnknown && base != expect {
					diff := int64(base) - int64(expect)
					kind := "gap"
					if diff < 0 {
						kind = "overlap"
					}
					f = append(f, RuleSegFragmentContinuity.New(floc,
						"%s of %d ticks (%.3fs): tfdt %d, but the previous fragment ends at %d",
						kind, diff, float64(diff)/float64(init.Timescale), base, expect))
				}
			} else {
				ref, what := tfdtRequired(rep, init, ms.Styp, timeStart && si == 0 && fi == 0)
				if ref != "" {
					f = append(f, ruleWithRef(RuleSegTfdt, ref).New(floc, "no tfdt box in %s", what))
				}
				// Without tfdt, a fragment starts where the previous fragment of
				// the track ended (ISO/IEC 14496-12 8.8.12). Before the first
				// fragment of a segment, that is the end of the previous segment.
				switch {
				case haveExpect:
					base, offset = expect, expect
				case firstSample:
					media.StartUnknown = true
					if ref == "" {
						f = append(f, RuleSegStartUnknown.New(loc, "the first fragment has no tfdt box"))
					}
				default:
					known = false // after a fragment that could not be read
				}
			}
			if firstSample {
				media.DecodeTime = base
			}
			samples, err := frag.GetFullSamples(init.trex)
			if err != nil {
				f = append(f, RuleSegParse.New(floc, "samples: %v", err))
				continue
			}
			next, haveNext = base, known
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
				if known {
					minCT = min(minCT, int64(s.DecodeTime+offset)+int64(s.CompositionTimeOffset))
				}
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
	ept := minCT - init.EditOffset
	// Whether the edit list cuts a segment with an unknown start is not known.
	if init.HasEditList && ept < 0 && !media.StartUnknown {
		// Media before the media time of the edit is not presented (ISO/IEC 14496-12 8.6.6.3).
		media.Cut = min(uint64(-ept), media.Duration)
		ept = 0
	}
	media.EPT = float64(ept) / mediaTS
	f = append(f, checkTiming(rep, init, seg, media, last, loc)...)
	f = append(f, iopMediaRules(rep, init, media, loc)...)
	f = append(f, cmafMediaRules(rep, init, media, mf, loc)...)
	f = append(f, ssrMediaRules(rep, init, media, loc)...)
	f = append(f, cmafBrandMediaRules(rep, init, mf, loc)...)
	f = append(f, llMediaRules(rep, media, mf, mediaTS, last, loc)...)
	f = append(f, dvbMediaRules(rep, init, media, mf, mediaTS, last, loc)...)
	f = append(f, hlsMediaRules(init, mf, loc)...)
	f = append(f, ac4MediaRules(init, mf, loc)...)
	return media, f
}

// tfdtRequired returns the reference of the requirement of a tfdt box in a
// traf, and what it applies to, or "" when none does. The content of
// CTA-5005-B (-hls-interop) is CMAF. timeStart marks the first fragment of a
// segment whose tfdt gives its start.
func tfdtRequired(rep *addressing.Rep, init *Init, styp *mp4.StypBox, timeStart bool) (ref, what string) {
	switch {
	case init.CMAF || init.HLSInterop:
		return refCMAF + " Table 5", "a CMAF track"
	case !rep.OnlyProfiles && styp != nil && (styp.MajorBrand() == "msdh" || slices.Contains(styp.CompatibleBrands(), "msdh")):
		return "ISO/IEC 23009-1 6.3.5.3", "a segment with the msdh brand"
	case timeStart:
		return "ISO/IEC 23009-1 7.3.5", "the first fragment of a segment with SegmentTimeline and $Time$ but no sidx"
	}
	return "", ""
}

// hasSidx reports whether a media segment has a sidx box.
func hasSidx(mf *mp4.File) bool {
	return mf.Sidx != nil || slices.ContainsFunc(mf.Segments, func(ms *mp4.MediaSegment) bool { return ms.Sidx != nil })
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
func checkTiming(rep *addressing.Rep, init *Init, seg addressing.Segment, media *Media, last bool, loc report.Location) []report.Finding {
	var f []report.Finding
	mediaTS := float64(init.Timescale)
	mpdTS := float64(rep.Timescale)
	wantStart := float64(seg.Time) / mpdTS
	wantDur := float64(seg.Dur) / mpdTS
	gotDur := float64(media.Presented()) / mediaTS
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
	switch {
	case media.StartUnknown:
		// Without the start, it is not known whether the edit list cuts the segment.
		if init.HasEditList {
			return nil
		}
	case exactStart:
		decodeStart := float64(media.DecodeTime) / mediaTS
		switch {
		case math.Abs(media.EPT-wantStart) <= exact:
		case math.Abs(decodeStart-wantStart) <= exact && !init.HasEditList:
			f = append(f, ruleWithRef(RuleSegEPTCTO, refTimeline).New(loc,
				"tfdt matches SegmentTimeline, but the earliest presentation time is %.6fs later",
				media.EPT-wantStart))
		default:
			f = append(f, ruleWithRef(RuleSegEPT, refTimeline).New(loc,
				"earliest presentation time %.6fs (tfdt %d) but SegmentTimeline says %.6fs",
				media.EPT, media.DecodeTime, wantStart))
		}
	case math.Abs(media.EPT-wantStart) > nominal:
		f = append(f, ruleWithRef(RuleSegEPT, ref).New(loc,
			"earliest presentation time %.3fs is more than 50%% of the segment duration from the nominal %.3fs",
			media.EPT, wantStart))
	}
	// The last Segment of a run with a negative S@r may end at the Period end
	// instead, like the last Segment with @duration (ISO/IEC 23009-1 5.3.9.6.2).
	toEnd := rep.PeriodDur - (float64(seg.Time)-float64(rep.PTO))/mpdTS
	cut := seg.PeriodEnd && gotDur < wantDur && gotDur > toEnd-exact
	switch {
	case exactDur && math.Abs(gotDur-wantDur) > exact && !cut:
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
	// A segment without tfdt continues the previous one by definition, and
	// the end of one with a relative start is not known.
	if prev.StartUnknown || cur.StartUnknown || prev.End() == cur.DecodeTime {
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
