# dashcheck rules

<!-- Generated from the rule registry and dashif-checks.json by `go test ./internal/coverage -update`. Do not edit. -->

All 266 rules that dashcheck reports, as `dashcheck -rules` lists them. Severities follow the spec wording: **error** for SHALL, **warning** for SHOULD, and **info** for information. The last column names the checks of the [DASH-IF Conformance tool](dashif-coverage.md) that a rule covers. The 65 rules marked **dashcheck** have no counterpart in that tool.

| Group | Rules | dashcheck only |
|---|---:|---:|
| [MPD syntax](#mpd-syntax) | 3 | 1 |
| [MPD](#mpd) | 64 | 22 |
| [MPD updates](#mpd-updates) | 5 | 3 |
| [Segment addressing](#segment-addressing) | 7 | 6 |
| [Segments](#segments) | 30 | 13 |
| [Live timing and HTTP](#live-timing-and-http) | 13 | 13 |
| [DASH-IF IOP](#dash-if-iop) | 22 | 4 |
| [CMAF](#cmaf) | 19 | 2 |
| [Low latency](#low-latency) | 25 | 0 |
| [DVB-DASH](#dvb-dash) | 56 | 1 |
| [HbbTV](#hbbtv) | 3 | 0 |
| [Dolby AC-4](#dolby-ac-4) | 7 | 0 |
| [CTA WAVE](#cta-wave) | 3 | 0 |
| [DASH-HLS interoperability (CTA-5005)](#dash-hls-interoperability-cta-5005) | 9 | 0 |
| **Total** | **266** | **65** |

## MPD syntax

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| MPD-PARSE | error | The MPD must be parseable into the MPD data model |  | **dashcheck** |
| MPD-XML | error | The MPD must be well-formed XML | W3C XML 1.0 | COMMON-2 |
| MPD-XSD | error | The MPD must validate against the DASH MPD XML schema | ISO/IEC 23009-1 6th ed. DASH-MPD.xsd | COMMON-2 |

## MPD

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| MPD-AS-ID | error | AdaptationSet@id must be unique within a Period | ISO/IEC 23009-1 5.3.3 | SCH-20 |
| MPD-AS-MINMAX | error | AdaptationSet minimum values must not be larger than the maximum values | ISO/IEC 23009-1 5.3.3 | SCH-23 |
| MPD-AS-NOREP | error | An AdaptationSet must have at least one Representation | ISO/IEC 23009-1 5.3.3 | SCH-27 |
| MPD-AS-RANGE | error | Representation bandwidth, width, height and frame rate must be within the AdaptationSet ranges | ISO/IEC 23009-1 5.3.3 | SCH-24, SCH-25, SCH-26, SCH-29 |
| MPD-AUDIO-CHANNELCONFIG | error | AudioChannelConfiguration must use the scheme required by the AC-4 or MPEG-H profile | DASH-IF IOP 9 | SCH-118, SCH-119 |
| MPD-BITSTREAMSWITCHING | error | With Period@bitstreamSwitching true, no AdaptationSet may set it to false | ISO/IEC 23009-1 5.3.2 | SCH-9 |
| MPD-CHAINING | error | Chaining and fallback descriptors need a @value, and fallback must be on the MPD level | ISO/IEC 23009-1 5.11 | SCH-68, SCH-69, SCH-70 |
| MPD-CHANNEL-POSITIONS | error | outputChannelPositionList values must be space-separated lists | ISO/IEC 23009-1 5.8.4.7 | SCH-63 |
| MPD-CODECS | warning | @codecs should be present on the AdaptationSet or the Representation, except for images | DASH-IF IOP | **dashcheck** |
| MPD-COMMON-BOTH | info | A common attribute is on both the AdaptationSet and a Representation, which ISO/IEC 23009-1 allows but the DASH-IF Conformance tool reports as an error | ISO/IEC 23009-1 5.3.7.1 | SCH-22 |
| MPD-CONTENTCOMPONENT | error | ContentComponent ids must be unique and must not repeat AdaptationSet attributes | ISO/IEC 23009-1 5.3.4 | SCH-21, SCH-33 |
| MPD-CONTENTTYPE | warning | AdaptationSet@contentType should agree with @mimeType | ISO/IEC 23009-1 5.3.3 | **dashcheck** |
| MPD-CP-CENC | error | cenc:default_KID must be a list of UUIDs, and cenc:pssh a base64 pssh box with the SystemID of its ContentProtection | ISO/IEC 23001-7 11.2 | **dashcheck** |
| MPD-CP-CLEARKEY | error | The Clear Key ContentProtection must have @value ClearKey1.0 and no Clear Key initialization data (cenc:pssh) | DASH-IF IOP 7.6.2.4 | **dashcheck** |
| MPD-CP-DEFAULTKID | warning | The mp4protection descriptor of Common Encryption content should have cenc:default_KID | ISO/IEC 23009-1 5.8.5.2.2, ISO/IEC 23001-7 11.3.2 | **dashcheck** |
| MPD-CP-REF | error | ContentProtection on the MPD or a Period needs @refId, @ref excludes @refId, and the source of a @ref must not be in a later Period | ISO/IEC 23009-1 5.3.1.2, 5.3.2.2, 5.8.4.1.3, 5.8.4.1.4 | **dashcheck** |
| MPD-CP-ROBUSTNESS | error | The mp4protection and CA_descriptor ContentProtection must not have @robustness | ISO/IEC 23009-1 5.8.5.2.2, 5.8.5.2.3 | **dashcheck** |
| MPD-CP-VALUE | error | ContentProtection @value must be a 4CC with an optional version for mp4protection and 4 lower-case hex digits for CA_descriptor | ISO/IEC 23009-1 5.8.5.2 | SCH-55, SCH-56 |
| MPD-DURATION-MISSING | error | Without @minimumUpdatePeriod and a last Period@duration, MPD@mediaPresentationDuration must be present | ISO/IEC 23009-1 5.3.1 | SCH-3, SCH-6 |
| MPD-DURATION-YM | warning | xs:duration values should not use years or months, whose length varies | DASH-IF IOP | IOP-1 |
| MPD-DYN-AST | error | A dynamic MPD must have @availabilityStartTime | ISO/IEC 23009-1 5.3.1 | SCH-1 |
| MPD-DYN-PUBLISHTIME | error | A dynamic MPD must have @publishTime | ISO/IEC 23009-1 5.3.1 | SCH-2 |
| MPD-DYN-TSBD | info | A dynamic MPD without @timeShiftBufferDepth has an infinite time-shift buffer | ISO/IEC 23009-1 5.3.1 | **dashcheck** |
| MPD-DYN-UTCTIMING | warning | A dynamic MPD should have at least one UTCTiming element | DASH-IF Timing Model 13.1 | **dashcheck** |
| MPD-EVENTSTREAM | error | EventStream must have @schemeIdUri, and @xlink:actuate only together with @xlink:href | ISO/IEC 23009-1 5.10.2 | SCH-64, SCH-65 |
| MPD-FRAMEPACKING | error | FramePacking must use a known scheme with a value 0 to 6 that fits the codec or container | ISO/IEC 23009-1 5.8.4.6 | SCH-59, SCH-60, SCH-61, SCH-62 |
| MPD-INDEXRANGEEXACT | error | @indexRangeExact must not be present without @indexRange | ISO/IEC 23009-1 5.3.9.2 | SCH-43, SCH-50, SCH-51 |
| MPD-MAXSEGDUR | error | No segment or Segment Sequence may be longer than MPD@maxSegmentDuration | ISO/IEC 23009-1 5.3.1, 5.3.9.6.2 | SCH-53 |
| MPD-MIMETYPE | error | @mimeType must be present on the AdaptationSet or the Representation | ISO/IEC 23009-1 5.3.7 | SCH-34 |
| MPD-PERIOD-ID | error | Period@id must be present in dynamic MPDs and unique within the MPD | ISO/IEC 23009-1 5.3.2 | SCH-10, SCH-12 |
| MPD-PERIOD-ORDER | error | Periods must be ordered by increasing start time | ISO/IEC 23009-1 5.3.2 | **dashcheck** |
| MPD-PERIOD-ZERO | warning | A Period with @duration 0 should have at most one AdaptationSet | ISO/IEC 23009-1 5.3.2 | SCH-14 |
| MPD-PROFILE-CHOSEN | warning | A profile chosen for the check (-profiles) should be listed in MPD@profiles | ISO/IEC 23009-1 8 | **dashcheck** |
| MPD-PROFILE-MAINROLE | error | With several video AdaptationSets in a Period, one must have Role main (extended and IOP profiles) | ISO/IEC 23009-1 8 | SCH-17, SCH-19, SCH-88 |
| MPD-PROFILE-SEGINFO | error | Segment information must be allowed by the profile | ISO/IEC 23009-1 8 | SCH-15, SCH-16, SCH-18, SCH-35, SCH-87 |
| MPD-PROFILE-SUBSET | error | @profiles must be a subset of the @profiles on higher levels | ISO/IEC 23009-1 5.3.7 | SCH-100, SCH-111, SCH-113 |
| MPD-PROFILE-TYPE | error | On-demand profiles require a static MPD | ISO/IEC 23009-1 8 | SCH-5, SCH-8 |
| MPD-PROFILE-UNKNOWN | info | Profiles unknown to dashcheck are not checked | ISO/IEC 23009-1 8 | SCH-4 |
| MPD-PROGINFO-LANG | error | With several ProgramInformation elements, each must have @lang | ISO/IEC 23009-1 5.3.1 | SCH-54 |
| MPD-REP-ID | error | Representation@id must be unique within a Period | ISO/IEC 23009-1 5.3.5 | **dashcheck** |
| MPD-ROLE-VALUE | error | Role values must be valid for their scheme | ISO/IEC 23009-1 5.8.5.5 | SCH-57, SCH-58 |
| MPD-SEGBASE-TSBD | error | A segment-level @timeShiftBufferDepth must not be smaller than MPD@timeShiftBufferDepth | ISO/IEC 23009-1 5.3.9.2 | SCH-52 |
| MPD-SEGINFO-MULTIPLE | error | At most one of SegmentBase, SegmentTemplate and SegmentList may be present on one element | ISO/IEC 23009-1 5.3.9.1 | SCH-11, SCH-28, SCH-36 |
| MPD-SEGINFO-NONE | error | A Period needs a BaseURL, SegmentTemplate or SegmentList at some level | ISO/IEC 23009-1 5.3.9.1 | SCH-13 |
| MPD-SRD-BOUNDS | error | SRD objects must fit within the total size: x+w &lt;= W and y+h &lt;= H | ISO/IEC 23009-1 H.2 | SCH-74, SCH-75, SCH-76, SCH-77 |
| MPD-SRD-ESSENTIAL | error | When every AdaptationSet in a Period has an SRD, at least one must be a SupplementalProperty | ISO/IEC 23009-1 H.2 | SCH-71 |
| MPD-SRD-PLACEMENT | error | SRD descriptors may only be on AdaptationSets and SubRepresentations | ISO/IEC 23009-1 H.2 | SCH-78 |
| MPD-SRD-TOTAL | error | Each SRD source needs a total size, given by every descriptor if they differ | ISO/IEC 23009-1 H.2 | SCH-72, SCH-73 |
| MPD-SRD-VALUE | error | SRD descriptors need a @value of 5, 7 or 8 comma-separated integers | ISO/IEC 23009-1 H.2 | SCH-79, SCH-80, SCH-81, SCH-82 |
| MPD-SSR-ATTRIBUTES | error | @k must be positive, S@k needs S@d, and @endSubNumber needs @k and no SegmentTimeline | ISO/IEC 23009-1 5.3.9.2.2, 5.3.9.6.3 | **dashcheck** |
| MPD-SSR-PROPERTIES | error | SegmentSequenceProperties@cadence must be greater than 0, and @alignment must list Representation@id values | ISO/IEC 23009-1 5.3.5.7 | **dashcheck** |
| MPD-SSR-SIGNAL | error | Segment Sequences need a profile that allows them or an EssentialProperty with urn:mpeg:dash:ssr:2023 | ISO/IEC 23009-1 5.3.5.7, 5.3.9.7.1 | **dashcheck** |
| MPD-SSR-TIMELINE | error | If one S element of a SegmentTimeline has @k or @ssp, all of them must have it | ISO/IEC 23009-1 5.3.9.7.1 | **dashcheck** |
| MPD-SUBREP-BANDWIDTH | error | A SubRepresentation with @level must have @bandwidth | ISO/IEC 23009-1 5.3.6 | SCH-40 |
| MPD-SUBSET-ID | error | Subset@id must be unique | ISO/IEC 23009-1 5.3.8 | SCH-66 |
| MPD-TEMPLATE-INIT | error | SegmentTemplate @initialization and @bitstreamSwitching must not contain $Number$ or $Time$ | ISO/IEC 23009-1 5.3.9.4 | SCH-44, SCH-45 |
| MPD-TEMPLATE-REPID | error | A SegmentTemplate shared by several Representations needs $RepresentationID$ or $Bandwidth$ | ISO/IEC 23009-1 5.3.9.4 | **dashcheck** |
| MPD-THUMBNAIL | error | Thumbnail tile Representations need a tile layout value, @bandwidth, @width and @height | DASH-IF IOP 6.2.6 | SCH-120, SCH-121, SCH-122, SCH-123 |
| MPD-TIMELINE-GAP | warning | SegmentTimeline should have no gaps | DASH-IF Timing Model 9.2 | **dashcheck** |
| MPD-TIMELINE-NEGR | error | A negative S@r must be on the last S or be followed by an S with @t | ISO/IEC 23009-1 5.3.9.6 | **dashcheck** |
| MPD-TIMELINE-ORDER | error | SegmentTimeline entries must not overlap | ISO/IEC 23009-1 5.3.9.6 | **dashcheck** |
| MPD-UTCTIMING-SCHEME | warning | UTCTiming should use http-xsdate, http-iso, http-head or direct (2014 schemes) | DASH-IF Timing Model 13.1 | SCH-67 |
| MPD-XLINK-START | warning | A Period after a remote Period should have @start, or the remote Period @duration, so that its start is known without resolving the remote Period | ISO/IEC 23009-1 5.3.2.1 | **dashcheck** |
| MPD-XLINK-UNRESOLVED | info | Remote Periods (@xlink:href) are not resolved, so their content is not checked, nor the segments of a Period whose timing depends on them | ISO/IEC 23009-1 5.5 | **dashcheck** |

## MPD updates

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| MPD-UPDATE-ID | error | MPD@id, Period@id and AdaptationSet@id must stay the same in MPD updates | ISO/IEC 23009-1 5.4.1 | DVB-46, DVB-49 |
| MPD-UPDATE-PUBLISHTIME | error | A changed MPD must have a new @publishTime, and a later fetch must not return an earlier version | ISO/IEC 23009-1 5.3.1.2, 5.4.1 | **dashcheck** |
| MPD-UPDATE-REP | error | A Representation must keep functionally equivalent attributes in MPD updates | ISO/IEC 23009-1 5.4.1 | DVB-53, DVB-54 |
| MPD-UPDATE-RESET | info | An MPD update signals a reset, so the MPD update rules do not apply to it | ISO/IEC 23009-1 5.4.2 | **dashcheck** |
| MPD-UPDATE-SEGMENTS | error | Segments described by both versions of an MPD must keep their timing and must not disappear while available | ISO/IEC 23009-1 5.4.1 | **dashcheck** |

## Segment addressing

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| ADDR-BASEURL | error | BaseURL elements must be valid URLs |  | **dashcheck** |
| ADDR-NONE | error | Each Representation needs SegmentTemplate, SegmentList or SegmentBase addressing | ISO/IEC 23009-1 5.3.9 | **dashcheck** |
| ADDR-PERIOD | error | Period start and duration must be derivable | ISO/IEC 23009-1 5.3.2 | **dashcheck** |
| ADDR-SEGMENTS | error | The segment list of a Representation must be derivable |  | **dashcheck** |
| ADDR-SINGLE | info | Representations that consist of a single segment given only by BaseURL are not checked yet |  | **dashcheck** |
| ADDR-TEMPLATE | error | SegmentTemplate and SegmentList must have a consistent set of identifiers and timing attributes | ISO/IEC 23009-1 5.3.9.4 | SCH-41, SCH-42, SCH-46, SCH-47, SCH-48, SCH-49, IOP-2 |
| ADDR-UNSUPPORTED | info | SegmentTimeline duration patterns (S@p, S@ssp) and URL parameters (Annex I) are not supported yet, so their segments are not checked |  | **dashcheck** |

## Segments

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| INIT-ONE-TRACK | warning | The initialization segment should have exactly one track | ISO/IEC 23000-19 | **dashcheck** |
| INIT-PARSE | error | The initialization segment must be a parseable ISOBMFF file |  | COMMON-5 |
| INIT-SAMPLES | warning | The initialization segment should not describe any samples | ISO/IEC 23000-19 | **dashcheck** |
| INIT-STRUCTURE | error | The initialization segment must have ftyp and moov with mvex and a track | ISO/IEC 23000-19 | **dashcheck** |
| INIT-UNSUPPORTED | info | Audio and video sample entries that dashcheck cannot decode, such as DTS, are not checked |  | **dashcheck** |
| SEG-AS-KID | error | All Representations of an IOP or CMAF AdaptationSet must be encrypted with the same tenc default_KID | DASH-IF IOP 7.7.1, 7.7.5, ISO/IEC 23000-19 8.2.3.1 | **dashcheck** |
| SEG-CMF2-DEFAULTS | error | Each chunk of a CMAF track with the cmf2 brand must set sample durations, sizes and flags in tfhd or trun, not only in trex | ISO/IEC 23000-19 7.7.3 | **dashcheck** |
| SEG-CMF2-ELST | error | A video CMAF track with the cmf2 brand must not have an edit list | ISO/IEC 23000-19 7.7.2 | **dashcheck** |
| SEG-CMF2-TRUN | error | A video CMAF track with the cmf2 brand must use trun version 1, except in a CMAF track file | ISO/IEC 23000-19 7.7.3 | **dashcheck** |
| SEG-CONTINUITY | error | Consecutive segments must have no gaps or overlaps in decode time | ISO/IEC 23000-19 7.3.2.2 | CMAF-5 |
| SEG-CONTINUITY-FRAGMENT | error | Each chunk or fragment in a segment must have the tfdt of the previous one plus its sample durations | ISO/IEC 23000-19 7.3.2.2 c, 7.3.2.3 g | CMAF-5, LL-12 |
| SEG-DURATION | error | The segment duration must match the MPD |  | LL-42 |
| SEG-EPT | error | The segment's earliest presentation time must match the MPD |  | LL-38, LL-41 |
| SEG-EPT-CTO | warning | tfdt matches the MPD, but composition time offsets without an edit list shift the earliest presentation time |  | LL-41 |
| SEG-PARSE | error | The media segment must be parseable ISOBMFF |  | CMAF-4, COMMON-5 |
| SEG-SAP | error | With @startWithSAP 1 or 2, every segment, or the first Partial Segment of each Segment Sequence, must start with a sync sample | ISO/IEC 23009-1 5.3.7 (@startWithSAP) | DVB-95 |
| SEG-SEQUENCE-NUMBER | warning | The chunks of a segment should share one mfhd sequence_number, which should increase from segment to segment (a convention checked with -sequence-numbers, not a CMAF requirement) |  | **dashcheck** |
| SEG-SIDX-SPAN | error | The sidx box of an indexed segment must index all of it: no movie fragment before its first or after its last subsegment | ISO/IEC 23009-1 6.3.5.4, 8.12.4.2 | LL-48 |
| SEG-SSR-SAP | error | The Partial Segments that SegmentSequenceProperties@cadence selects must start with a SAP type of at most @sapType | ISO/IEC 23009-1 5.3.5.7 | **dashcheck** |
| SEG-STRUCTURE | error | A media segment must have moof with traf, tfdt and trun, and mdat | ISO/IEC 23000-19 | LL-12 |
| SEG-TRACKID | error | tfhd@track_ID must match the track in the initialization segment |  | **dashcheck** |
| SEG-UNSUPPORTED | info | Segments in formats other than ISOBMFF are not checked |  | **dashcheck** |
| XCHK-CODECS | error | @codecs must match the sample entry type |  | DVB-85, LL-23 |
| XCHK-CODECS-DETAIL | warning | @codecs should match the codec string derived from the decoder configuration |  | DVB-85, LL-23 |
| XCHK-CP-KID | error | cenc:default_KID must list the default_KID in the tenc box of the track | ISO/IEC 23001-7 11.3.2 | LL-24 |
| XCHK-CP-MISSING | error | Encrypted tracks need ContentProtection in the MPD, and Common Encryption the mp4protection descriptor | ISO/IEC 23009-1 5.8.4.1.1, ISO/IEC 23001-7 11.3.2 | LL-24 |
| XCHK-CP-SCHEME | error | The mp4protection @value must be the scheme type in the schm box of the track | ISO/IEC 23009-1 5.8.5.2.2 | LL-24 |
| XCHK-HANDLER | error | The track handler must match @contentType or @mimeType |  | DVB-82, LL-19 |
| XCHK-SAMPLERATE | warning | @audioSamplingRate should match the sample entry, or for HE-AAC the sampling rate after SBR |  | DVB-83 |
| XCHK-SIZE | warning | @width and @height should match the sample entry |  | **dashcheck** |

## Live timing and HTTP

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| CLOCK-LOCAL | info | The offset between the local clock and the reference clock |  | **dashcheck** |
| CLOCK-SKEW | warning | A UTCTiming source should be within 500ms of the reference clock |  | **dashcheck** |
| CLOCK-SOURCE | error | Each UTCTiming source must respond with a parseable time | ISO/IEC 23009-1 (UTCTiming) | **dashcheck** |
| CLOCK-UNSUPPORTED | info | UTCTiming schemes that the checker cannot measure are skipped |  | **dashcheck** |
| HTTP-INIT | error | The initialization segment must be fetchable |  | **dashcheck** |
| HTTP-MPD | error | The MPD must be fetchable |  | **dashcheck** |
| HTTP-RANGE | warning | HTTP servers should answer byte range requests with the requested range | DASH-IF IOP v4.3 3.4.2 | **dashcheck** |
| HTTP-SEGMENT | error | Media segments must be fetchable while they are in the availability window |  | **dashcheck** |
| LIVE-EARLY | warning | Segments should not be available before their availability start time | DASH-IF Timing Model 13.3 | **dashcheck** |
| LIVE-NO-SEGMENTS | error | A live Representation must have segments in the availability window | DASH-IF Timing Model 13.3 | **dashcheck** |
| LIVE-NOT-STARTED | warning | The presentation has not started yet, so no segments can be checked |  | **dashcheck** |
| LIVE-PUBLISHTIME | warning | MPD@publishTime should not be in the future |  | **dashcheck** |
| LIVE-STALE | error | The MPD must describe segments up to the live edge | DASH-IF Timing Model 9.2.2 | **dashcheck** |

## DASH-IF IOP

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| HTTP-IOP-RANGE | error | HTTP servers must answer byte range requests for IOP on-demand Representations with the requested range | DASH-IF IOP v4.3 3.4.2 | **dashcheck** |
| MPD-IOP-AUDIO | error | IOP audio must signal @lang, @audioSamplingRate and AudioChannelConfiguration | DASH-IF IOP 3.2.4 | SCH-93, SCH-108 |
| MPD-IOP-CODECS | error | IOP @codecs must have a single value | DASH-IF IOP 3.2.13 | SCH-109 |
| MPD-IOP-CP | error | IOP ContentProtection must be on the AdaptationSet, and mp4protection must have cenc:default_KID | DASH-IF IOP 7 | SCH-115, SCH-116, SCH-117 |
| MPD-IOP-CP-DRM | warning | IOP DRM ContentProtection descriptors (urn:uuid) should have cenc:pssh and a @value with the DRM name and version | DASH-IF IOP 7.7.2 | **dashcheck** |
| MPD-IOP-DEPENDENCY | error | A Dolby Vision enhancement layer @dependencyId must refer to an earlier Representation | DASH-IF IOP 10 | SCH-110 |
| MPD-IOP-LIVE | error | A dynamic IOP MPD must use the live profile, and live AdaptationSets must have @segmentAlignment true | DASH-IF IOP 3.2.2.2 | SCH-85, SCH-97 |
| MPD-IOP-MIMETYPE | error | IOP AdaptationSets must use an allowed @mimeType, the same for all Representations | DASH-IF IOP 3.2.13 | SCH-99, SCH-104 |
| MPD-IOP-MINBUFFER | error | An IOP MPD must have @minBufferTime | DASH-IF IOP 3.10.2 | SCH-86 |
| MPD-IOP-ONDEMAND | error | IOP on-demand content needs SegmentBase@indexRange, subsegment alignment and subsegments starting with SAP 1 or 2 | DASH-IF IOP 3.10.3 | SCH-98, SCH-101, SCH-102, SCH-112, SCH-114, IOP-4, IOP-13 |
| MPD-IOP-SCANTYPE | error | IOP video must be progressive if @scanType is present | DASH-IF IOP 3.2.4 | SCH-92, SCH-107 |
| MPD-IOP-STATIC | error | A static IOP MPD must not have @timeShiftBufferDepth or @minimumUpdatePeriod | DASH-IF IOP 3.2 | SCH-83, SCH-84 |
| MPD-IOP-SWITCHING | error | AdaptationSets that can be switched between must have the same segment alignment | DASH-IF IOP 3.8 | SCH-90 |
| MPD-IOP-VIDEO | error | IOP video AdaptationSets and Representations must signal @par, size, frame rate and @sar | DASH-IF IOP 3.2.4 | SCH-91, SCH-94, SCH-95, SCH-96, SCH-105, SCH-106 |
| SEG-IOP-ASCODECS | error | AdaptationSet @codecs must signal the highest profile and level of its Representations | DASH-IF IOP 6.2.5.2 | IOP-7, IOP-8 |
| SEG-IOP-BITSTREAM | error | With bitstream switching, AVC must use avc3 and HEVC hev1 sample entries | DASH-IF IOP 6.2.5.2 | IOP-5, IOP-10 |
| SEG-IOP-EDITLIST | error | Video edit lists are only allowed with the on-demand profile, and then in all Representations | DASH-IF IOP 6.2.5.2 | IOP-9, IOP-14 |
| SEG-IOP-FIRSTCT | error | With bitstream switching, the first video sample's composition and decode times must be equal | DASH-IF IOP 6.2.5.2 | IOP-15 |
| SEG-IOP-PARAMSETS | error | With bitstream switching, avcC must hold SPS and PPS, and hvcC VPS, SPS and PPS | DASH-IF IOP 6.2.5.2 | IOP-6, IOP-11 |
| SEG-IOP-PSSH | error | The initialization segments of an IOP AdaptationSet must have the same pssh boxes for each SystemID, and a W3C Common pssh box must list only the default_KID | DASH-IF IOP 7.7.1 | **dashcheck** |
| SEG-IOP-PSSH-INIT | warning | IOP initialization segments should have no pssh boxes, but cenc:pssh in the MPD instead | DASH-IF IOP 7.7.1 | **dashcheck** |
| SEG-IOP-SIDX | error | IOP on-demand Representations must have exactly one sidx box | DASH-IF IOP 3.10.3.2 | IOP-12 |

## CMAF

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| SEG-CMAF-CHUNK | error | Each moof of a CMAF track must contain one traf with one trun | ISO/IEC 23000-19 7.3.2.3 | LL-12 |
| SEG-CMAF-COLR | error | A CMAF HEVC sample entry must contain a colr box of type nclx | ISO/IEC 23000-19 7.3.2.4 | CMAF-14 |
| SEG-CMAF-COMPTIME | error | CMAF video with composition offsets must use either trun version 1 or an edit list with trun version 0, not both | ISO/IEC 23000-19 9.2.1 | CMAF-3 |
| SEG-CMAF-DATAOFFSET | error | CMAF samples must be addressed by trun data offsets relative to the moof | ISO/IEC 23000-19 7.3.5 | CMAF-4, LL-12 |
| SEG-CMAF-ENCRYPTION | error | Tracks must match the encryption of their CMAF representation profile (cmfhd clear, cmfhdc cenc, cmfhds cbcs) | ISO/IEC 23000-19 A.1 | CMAF-6, CMAF-7, CMAF-8 |
| SEG-CMAF-HEVC-SWITCHING | error | HEVC tracks of a CMAF switching set must have identical SPS colour and extension information and SEI | ISO/IEC 23000-19 B.2.4 | CMAF-9, CMAF-10 |
| SEG-CMAF-INIT | error | A CMAF header must describe the track: decoder configuration and size for video, sample rate and channels for audio | ISO/IEC 23000-19 7.3.5 | CMAF-11, CMAF-12, CMAF-13 |
| SEG-CMAF-MEDIAPROFILE | error | Tracks must not exceed the encoding parameters of the CMAF media profile brand they signal | ISO/IEC 23000-19 A.2, A.3, B.6 | CMAF-1, CMAF-22, WAVE-1, WAVE-18 |
| SEG-CMAF-MINORVERSION | error | With a structural CMAF brand (cmfc or cmf2) as major_brand, ftyp and styp must have minor_version 0 | ISO/IEC 23000-19 7.2 | **dashcheck** |
| SEG-CMAF-PRESENTATIONPROFILE | error | With the cmfhd presentation profile, video switching sets need a cfhd track and audio switching sets a caac track | ISO/IEC 23000-19 A.1.2 | CMAF-2, CMAF-23, WAVE-3, WAVE-16, WAVE-20 |
| SEG-CMAF-SAP | error | Each video CMAF fragment must start with a sync sample (SAP type 1 or 2) | ISO/IEC 23000-19 9.2.8 | LL-12 |
| SEG-CMAF-SENC | error | Each encrypted CMAF fragment must carry the sample auxiliary information that decrypting it needs: subsamples for video, or per-sample IVs | ISO/IEC 23000-19 7.3.2.4, 8.2.2.1 | CMAF-15 |
| SEG-CMAF-SIDX | error | Each subsegment referenced by a CMAF sidx box must be a single media fragment | ISO/IEC 23000-19 7.3.3.3 | CMAF-19, LL-48 |
| SEG-CMAF-SINF | error | Every sample entry of an encrypted CMAF track must be protected (sinf), with tenc default_isProtected 1; sample groups mark clear samples | ISO/IEC 23000-19 7.5.10, 8.2.3.2 | **dashcheck** |
| SEG-CMAF-SUBS | error | Each fragment of an im1i subtitle track must contain a subs box | ISO/IEC 23000-19 7.5.20 | CMAF-21 |
| SEG-CMAF-SUBTITLEPROFILE | error | All subtitle tracks of a CMAF switching set must conform to one media profile | ISO/IEC 23000-19 7.3.4.1 | CMAF-20, WAVE-14 |
| SEG-CMAF-SWITCHINGSET | error | All tracks of a CMAF switching set must have the same media type | ISO/IEC 23000-19 7.3.9.1 | CMAF-18 |
| SEG-CMAF-TOPLEVEL | error | meta and udta boxes must not be at the top level of CMAF files | ISO/IEC 23000-19 7.5.2 | CMAF-16, CMAF-17 |
| XCHK-CMAF-SIZE | error | In the DASH profile for CMAF content, @maxWidth and @maxHeight, or else @width and @height, must be the tkhd size of the CMAF principal header | ISO/IEC 23009-1 8.12.4.3 | LL-21, LL-22 |

## Low latency

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| MPD-LL-ADDRESSING | error | Low-latency AdaptationSets must use a SegmentTemplate with @duration and $Number$, or a SegmentTimeline | DASH-IF Low-Latency Modes 9.X.4.3 | LL-7, LL-8, LL-9 |
| MPD-LL-ALIGNMENT | error | Low-latency AdaptationSets must set @segmentAlignment or @subsegmentAlignment | DASH-IF Low-Latency Modes 9.X.4.5 | LL-26 |
| MPD-LL-CHUNKED | error | Chunked low-latency Representations need @availabilityTimeOffset and @availabilityTimeComplete false | DASH-IF Low-Latency Modes 9.X.4.5 | LL-16, LL-17 |
| MPD-LL-INBAND | warning | Low-latency Representations should carry MPD validity expiration events (urn:mpeg:dash:event:2012, @value 1) | DASH-IF Low-Latency Modes 9.X.4.2 | LL-2 |
| MPD-LL-INBAND-VALUE | error | The urn:mpeg:dash:event:2012 InbandEventStream must have @value 1 | DASH-IF Low-Latency Modes 9.X.4.2 | LL-3 |
| MPD-LL-K | error | When each chunk is addressable ($SubNumber$), the SegmentTimeline or SegmentTemplate must give @k | DASH-IF Low-Latency Modes 9.X.4.5, ISO/IEC 23009-1 5.3.9.7 | LL-40 |
| MPD-LL-LEAPSECOND | warning | Low-latency presentations should provide LeapSecondInformation | DASH-IF Low-Latency Modes 9.X.4.2 | LL-4 |
| MPD-LL-MIMETYPE | error | Low-latency @mimeType must be the @contentType with /mp4, or application/mp4 for text | DASH-IF Low-Latency Modes 9.X.4.5 | LL-20 |
| MPD-LL-MODE | error | A low-latency AdaptationSet must be either chunked or segment based, not both | DASH-IF Low-Latency Modes 9.X.4.3 | LL-37 |
| MPD-LL-PROFILE | warning | Low-latency presentations should signal the DASH-IF low-latency profile | DASH-IF Low-Latency Modes 9.X.4.1 | LL-1 |
| MPD-LL-PRT | error | Each low-latency AdaptationSet needs a valid ProducerReferenceTime with a unique @id | DASH-IF Low-Latency Modes 9.X.4.3 | LL-5, LL-6 |
| MPD-LL-RESYNC | warning | Chunked low-latency Representations should have a Resync element and @qualityRanking | DASH-IF Low-Latency Modes 9.X.4.5 | LL-15, LL-29, LL-32, LL-33 |
| MPD-LL-RESYNC-VALUE | error | Resync elements need a valid @type, @marker true and @dT not above the target latency | DASH-IF Low-Latency Modes 9.X.4.5 | LL-30, LL-31, LL-32 |
| MPD-LL-SERVICEDESC | error | Low-latency presentations need a ServiceDescription with a target Latency | DASH-IF Low-Latency Modes 9.X.4.2 | LL-10 |
| MPD-LL-SERVICEDESC-OPTIONAL | info | A low-latency ServiceDescription may give minimum and maximum latency and playback rates | DASH-IF Low-Latency Modes 9.X.4.2 | LL-10 |
| MPD-LL-TIMELINE-REPEAT | warning | Consecutive S elements with the same duration should be combined with @r | DASH-IF Low-Latency Modes 9.X.4.5 | LL-39 |
| MPD-LL-UTCTIMING | error | Low-latency presentations need a UTCTiming with the http-xsdate, http-iso or http-ntp scheme | DASH-IF Low-Latency Modes 9.X.4.2 | LL-11 |
| SEG-EMSG-REPEAT | error | An emsg box after the first moof of a segment must be repeated before the first moof of the next segment (low-latency chunked and DASH-HLS interoperable content) | DASH-IF IOP low latency 9.X.4.5 6b, CTA-5005-B 4.5.2 | LL-13, LL-28, WAVEHLS-17 |
| SEG-LL-CHUNKS | warning | Segments of chunked low-latency Representations should contain more than one chunk | DASH-IF Low-Latency Modes 9.X.4.5 | LL-14 |
| SEG-LL-DURATION | error | Low-latency segments must not exceed 50% of the target latency in segment mode | DASH-IF Low-Latency Modes 9.X.4.4 | LL-36 |
| SEG-LL-DURATION-RECOMMENDED | warning | Low-latency segments should not exceed 30% of the target latency in segment mode, or the target latency when chunked | DASH-IF Low-Latency Modes 9.X.4.4 | LL-18, LL-36 |
| SEG-LL-SIDX | error | A self-initializing low-latency segment needs one sidx before the first moof that matches the track | DASH-IF Low-Latency Modes 9.X.4.5 | LL-43, LL-44, LL-45, LL-46, LL-47, LL-49, LL-50, LL-51, LL-52, LL-53, LL-54 |
| SEG-LL-SINGLEMOOF | warning | Segments of low-latency segment-mode Representations should have a single moof box | DASH-IF Low-Latency Modes 9.X.4.4 | LL-34 |
| SEG-LL-SMDS | error | Segments with the smds brand need smds in @segmentProfiles | DASH-IF Low-Latency Modes 9.X.4.4 | LL-35 |
| SEG-LL-TIMESCALE | error | The SegmentTemplate @timescale of low-latency content must equal the mdhd timescale | DASH-IF Low-Latency Modes 9.X.4.5 | LL-25 |

## DVB-DASH

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| MPD-DVB-AC4 | error | DVB AC-4 needs presentation_version 1, md_compat at most 3, a 48, 96 or 192 kHz sampling rate, and audio frame rates matching video | ETSI TS 103 285 6.3 | SCH-DVB-10, SCH-DVB-11, SCH-DVB-12, SCH-DVB-13 |
| MPD-DVB-AC4-VERSION | warning | DVB AC-4 should use bitstream_version 2 | ETSI TS 103 285 6.3 | SCH-DVB-9 |
| MPD-DVB-AUDIO | error | DVB audio needs a Role, @mimeType, @codecs, @audioSamplingRate and AudioChannelConfiguration, and E-AC-3 the Dolby channel scheme | ETSI TS 103 285 6.1 | DVB-3, DVB-4, DVB-5 |
| MPD-DVB-AUDIO-COMMON | error | DVB audio Representations in an AdaptationSet must share @mimeType | ETSI TS 103 285 6.1 | SCH-DVB-1, DVB-6 |
| MPD-DVB-AUDIO-SHARE | warning | Audio should be at most 20% of the total bandwidth | ETSI TS 103 285 11.3.0 | DVB-11 |
| MPD-DVB-AUDIO-SIMILAR | warning | DVB audio Representations in an AdaptationSet should share @codecs, @audioSamplingRate and AudioChannelConfiguration | ETSI TS 103 285 6.1 | SCH-DVB-2, SCH-DVB-3, SCH-DVB-4, DVB-7, DVB-8, DVB-9 |
| MPD-DVB-AUXILIARY | error | DVB auxiliary audio AdaptationSets must not have @lang, Role, Accessibility or AudioChannelConfiguration | ETSI TS 103 285 6.8 | SCH-DVB-5 |
| MPD-DVB-CODEC | warning | Codecs should be ones that DVB-DASH supports | ETSI TS 103 285 5, 6 | DVB-12 |
| MPD-DVB-CONTINUITY | error | A period-continuity descriptor must name an earlier Period with an AdaptationSet of the same @id, and the two AdaptationSets must be associated: equivalent AssetIdentifier on both Periods and the same @lang, @contentType, @par, Role, Accessibility, Viewpoint and, for audio, @mimeType, @codecs, @audioSamplingRate and AudioChannelConfiguration | ETSI TS 103 285 10.5.2.2, 10.5.2.3 | DVB-79 |
| MPD-DVB-CP | error | DVB ContentProtection descriptors must be on the AdaptationSet and include the mp4protection descriptor | ETSI TS 103 285 8.3, 8.4 | DVB-13, DVB-75 |
| MPD-DVB-CP-DRM | error | A protected DVB AdaptationSet needs one mp4protection descriptor, with @value cenc or cbcs, and a urn:uuid descriptor for each DRM system | ETSI TS 103 285 8.4 | **dashcheck** |
| MPD-DVB-EVENT | error | DVB TV-Anytime events need @presentationTime and valid XML data | ETSI TS 103 285 9.1.2.1 | DVB-18, DVB-19 |
| MPD-DVB-FALLBACK | error | A fallback audio AdaptationSet must refer to an existing audio AdaptationSet | ETSI TS 103 285 6.6.3 | DVB-10 |
| MPD-DVB-FONT | error | Downloadable font descriptors are only allowed on subtitle AdaptationSets | ETSI TS 103 285 7.2.1.1 | DVB-2, DVB-56 |
| MPD-DVB-FRAMERATES | warning | Frame rates in an AdaptationSet should be integer multiples of each other | ETSI TS 103 285 11.2.2 | DVB-59 |
| MPD-DVB-LIMITS | error | A DVB MPD must be at most 256 kB with at most 64 Periods, 16 AdaptationSets per Period and 16 Representations per AdaptationSet | ETSI TS 103 285 4.5 | DVB-14, DVB-15, DVB-16, DVB-17 |
| MPD-DVB-MAINROLE | error | With several video or audio AdaptationSets in a Period, one of them must have Role main | ETSI TS 103 285 4.2.2, 6.1.2 | DVB-25, DVB-26 |
| MPD-DVB-MUP | warning | DVB MPD@minimumUpdatePeriod should be at least 1 second | ETSI TS 103 285 | DVB-1 |
| MPD-DVB-PERIOD | error | DVB Periods must be either live or on-demand, without Period.SegmentList, and without Period.SegmentTemplate for on-demand | ETSI TS 103 285 4.2.2 | DVB-22, DVB-23, DVB-24 |
| MPD-DVB-PRESELECTION | error | DVB audio Preselections need a role, a main one per bundle, and components that are AdaptationSet ids | ETSI TS 103 285 6.8 | SCH-DVB-6, SCH-DVB-7, SCH-DVB-8 |
| MPD-DVB-PROFILE | error | Each DVB Representation must use exactly one of the live and on-demand profiles, and on-demand means a single segment | ETSI TS 103 285 11.1 | DVB-27, DVB-28, DVB-29 |
| MPD-DVB-REPORTING | error | DVB metric reporting needs an absolute HTTP(S) reporting URL and a probability between 1 and 1000 | ETSI TS 103 285 10.12.3 | DVB-20, DVB-21 |
| MPD-DVB-RESOLUTION | warning | Video resolutions should be ones listed by DVB-DASH | ETSI TS 103 285 10.3 | DVB-30 |
| MPD-DVB-SCTE | error | SCTE-35 event streams need a frame-rate multiple @timescale, and events a @presentationTime and a finite @duration without @messageData or @contentEncoding | ETSI TS 103 285 9.1.9 | DVB-33, DVB-34, DVB-35, DVB-36, DVB-37 |
| MPD-DVB-SCTE-BINARY | error | An SCTE-35 event must carry a base64 splice_info_section in Signal/Binary | ETSI TS 103 285 9.1.9 | DVB-34 |
| MPD-DVB-SCTE-VALUE | warning | SCTE-35 event streams should not have @value | ETSI TS 103 285 9.1.9 | DVB-31 |
| MPD-DVB-SUBTITLE | error | DVB subtitle @codecs must start with stpp | ETSI TS 103 285 7.1.1 | DVB-38 |
| MPD-DVB-TLS-BITRATE | info | Over TLS, terminals without UHD support handle 12 Mbit/s, UHD terminals 39 Mbit/s and UHD HFR terminals 51 Mbit/s | ETSI TS 103 285 10.2 | DVB-39, DVB-40, DVB-41 |
| MPD-DVB-UPDATE | error | A DVB MPD update must keep @timeShiftBufferDepth, @availabilityStartTime, Period@start, AssetIdentifier, Role, AudioChannelConfiguration, @timescale and @lang, must not increase @maxSegmentDuration, and must not remove AdaptationSets or Representations | ETSI TS 103 285 4.8.1, 4.8.3 | DVB-43, DVB-44, DVB-45, DVB-47, DVB-48, DVB-50, DVB-51, DVB-52, DVB-55 |
| MPD-DVB-UTCTIMING | warning | A DVB MPD with an availabilityStartTime should have a UTCTiming with an NTP or HTTP scheme | ETSI TS 103 285 4.7.2 | DVB-42 |
| MPD-DVB-VIDEO | error | DVB video AdaptationSets need size, frame rate and @par, and @scanType when interlaced video is used | ETSI TS 103 285 4.4 | DVB-57, DVB-58 |
| MPD-DVB-XSD | warning | A DVB MPD should validate against the schema of ISO/IEC 23009-1:2019 (4th edition with Amd 1), the edition DVB-DASH V1.3.1 and later reference | ETSI TS 103 285 V1.4.1 2.1 [1] | COMMON-4 |
| MPD-DVB-XSD-2014 | info | A DVB MPD is not valid against the schema of ISO/IEC 23009-1:2014 with Amd 1, which DVB-DASH 2014 (V1.2.1) references | ETSI TS 103 285 V1.2.1 2.1 | COMMON-3 |
| SEG-DVB-AVC | warning | A DVB Period with AVC video should have a Representation for the avc_hd_50_level40 or avc_hd_60_level40 player conformance point (Main or High profile up to level 4.0), unless it targets devices with other capabilities | ETSI TS 103 285 5.1.1, ETSI TS 101 154 L.1 | DVB-89, DVB-90, HBBTV-8, HBBTV-9 |
| SEG-DVB-AVC3 | warning | DVB AVC content should use the avc3 or avc4 sample entry | ETSI TS 103 285 5.1.2 | DVB-73 |
| SEG-DVB-CENC | error | DVB encrypted content must use the cenc or cbcs scheme, and cbcs not with the 2014 DVB profile | ETSI TS 103 285 8.6.0 | DVB-78 |
| SEG-DVB-CENC-FALLBACK | error | A DVB Period with AVC video must have an AVC video AdaptationSet and an AdaptationSet of each other media type that is unencrypted or uses the cenc scheme | ETSI TS 103 285 8.6.0 | DVB-78 |
| SEG-DVB-CHANNELS | error | The MPEG AudioChannelConfiguration value must match the channel count of the sample entry | ETSI TS 103 285 6.1.1 | DVB-84 |
| SEG-DVB-CODEC | warning | The sample entry should be a codec that DVB-DASH supports | ETSI TS 103 285 5, 6, 7 | DVB-88, HBBTV-7 |
| SEG-DVB-COLOUR | error | ColourPrimaries, MatrixCoefficients and TransferCharacteristics EssentialProperty values must match the VUI of the video | ETSI TS 103 285 5.2.5 | DVB-66, DVB-67, DVB-68 |
| SEG-DVB-CONTINUITY-EPT | error | All Representations of a period-continuous AdaptationSet must share the earliest presentation time | ETSI TS 103 285 10.5.2.3 | DVB-80, DVB-81 |
| SEG-DVB-DURATION | error | DVB segments and subsegments must be between 1 and 15 seconds long | ETSI TS 103 285 4.5 | DVB-86, DVB-87 |
| SEG-DVB-HEVC | warning | A DVB Period with HEVC video should have a Representation for an HEVC player conformance point (Main or Main 10 profile, Main tier, level 4.1 up to 1080p, 5.2 up to 2160p, 6.1 up to 4320p), unless it targets devices with other capabilities | ETSI TS 103 285 5.2.3, ETSI TS 101 154 L.1 | DVB-91, DVB-93, DVB-94 |
| SEG-DVB-HEVC-CODECS | error | HEVC @codecs must follow ISO/IEC 14496-15 A.3 and signal a profile, tier, level and constraints that the media meets | ETSI TS 103 285 5.2.2 | DVB-65 |
| SEG-DVB-HEVC-SPS | error | DVB HEVC must be progressive frames, 8-bit for Main profile and 8 or 10 bits with equal chroma bit depth for Main 10 | ETSI TS 103 285 5.2.3, ETSI TS 101 154 L.3.3.2 | DVB-92 |
| SEG-DVB-HLG10 | warning | HLG10 video should be signalled with the 2017 profile, EssentialProperty ColourPrimaries 9, MatrixCoefficients 9 and TransferCharacteristics 14, and SupplementalProperty TransferCharacteristics 18; only MPDs for players known to support BT.2020 may leave out the profile and EssentialProperty descriptors | ETSI TS 103 285 5.2.6 | DVB-69 |
| SEG-DVB-PARAMSETS | error | With avc1 or avc2, the initialization segment must hold all SPS and PPS needed to decode | ETSI TS 103 285 5.1.2 | DVB-74 |
| SEG-DVB-PQ10 | error | PQ10 video must be signalled with the 2017 profile and EssentialProperty ColourPrimaries 9, MatrixCoefficients 9 and TransferCharacteristics 16 | ETSI TS 103 285 5.2.7 | DVB-70 |
| SEG-DVB-PSSH | info | A DRM system in the MPD has no pssh in the MPD or the initialization segment, so it must come from the media segments or elsewhere | ETSI TS 103 285 8.5.0 | DVB-76 |
| SEG-DVB-SAP | error | DVB video segments must start with a SAP of type 1 or 2 | ETSI TS 103 285 5.1.2 | DVB-95 |
| SEG-DVB-SEI | error | HEVC SEI arrays must follow the parameter set arrays in hvcC; in samples, prefix SEI must follow the parameter sets and precede the first VCL NAL unit, and suffix SEI must follow the last one | ETSI TS 103 285 5.2.4, ETSI TS 101 154 L.3.3.8 | DVB-96 |
| SEG-DVB-SIDX | error | DVB on-demand Representations must have exactly one sidx box | ETSI TS 103 285 4.3 | DVB-72 |
| SEG-DVB-SUBTITLE | error | DVB stpp subtitles need a subt handler and an stpp sample entry with a namespace | ETSI TS 103 285 7.1 | DVB-98, DVB-99, DVB-100 |
| SEG-DVB-SWITCHABLE | error | The Representations of a DVB AdaptationSet must have the same sample entry type, track ID and default KID | ETSI TS 103 285 4.3, 8.3 | DVB-101, DVB-102, DVB-103 |
| SEG-DVB-TRAF | error | Each moof box must contain exactly one traf box | ETSI TS 103 285 4.3 | DVB-71 |
| SEG-DVB-VVC-SEI | error | In VVC picture units, prefix SEI must follow DCI, VPS, SPS and PPS and precede the picture header or first VCL NAL unit, and suffix SEI must follow the last VCL NAL unit | ETSI TS 103 285 5.3.4, ETSI TS 101 154 5.15.1.8.1 | DVB-97 |

## HbbTV

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| MPD-HBBTV | error | MPDs with the HbbTV profile must follow its restrictions on segments and alignment | HbbTV 2.0 E.2 | SCH-30, SCH-31, SCH-32, SCH-37, SCH-38, SCH-39, SCH-103 |
| MPD-HBBTV-CODECS | error | An HbbTV MPD must have an AdaptationSet with a codec of ETSI TS 102 796 7.3.1 (AVC, HEVC, HE-AAC or E-AC-3) | ETSI TS 102 796 E.2.1 | HBBTV-2 |
| SEG-HBBTV-AUDIO-CONFIG | error | With the HbbTV profile, the initialization segment must hold everything needed to decode the audio | ETSI TS 102 796 annex E | HBBTV-10 |

## Dolby AC-4

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| MPD-AC4-ATTRS | error | AC-4 AdaptationSets need the right @mimeType, @audioSamplingRate and @startWithSAP | ETSI TS 103 190-2 G.2 | SCH-AC4-2, SCH-AC4-3, SCH-AC4-4, SCH-AC4-5 |
| MPD-AC4-CHANNELCONFIG | error | AC-4 AudioChannelConfiguration must use a valid scheme and value, and the MPEG scheme when it can | ETSI TS 103 190-2 G.2 | SCH-AC4-6, SCH-AC4-10, SCH-AC4-11, SCH-AC4-14, SCH-AC4-15, SCH-AC4-17 |
| MPD-AC4-CHANNELCONFIG-COUNT | warning | AC-4 Representations should have exactly one AudioChannelConfiguration and use specified schemes | ETSI TS 103 190-2 G.2 | SCH-AC4-9, SCH-AC4-12, SCH-AC4-13, SCH-AC4-16 |
| MPD-AC4-CODECS | error | AC-4 @codecs must be ac-4 followed by three two-digit hexadecimal fields | ETSI TS 103 190-1 F.1 | SCH-AC4-1 |
| MPD-AC4-ROLE | error | AC-4 (bitstream version 0) Role values must be main, alternate, commentary or dub | ETSI TS 103 190-1 F.1 | SCH-AC4-7 |
| MPD-AC4-VIRTUALIZED | error | The AC-4 immersive audio for headphones descriptor must have @value 1 | ETSI TS 103 190-2 G.2.12 | SCH-AC4-8 |
| SEG-AC4-TOC | error | The dac4 box must match the TOC of the AC-4 frames: bitstream_version, fs_index, frame_rate_index, n_presentations and short_program_id | ETSI TS 103 190-2 E.6.3 to E.6.7 | DOLBY-1, DOLBY-2, DOLBY-3, DOLBY-4, DOLBY-5 |

## CTA WAVE

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| SEG-WAVE-PROFILE | error | Each selection set of WAVE content needs a switching set whose tracks all conform to one WAVE media profile (CTA-5001 tables 1 to 3) | CTA-5001-F 4.1 | WAVE-2, WAVE-15, WAVE-19 |
| SEG-WAVE-SCHEME | warning | WAVE content should use only one Common Encryption scheme: CMAF presentation profiles allow one per presentation, and the Baseline Splice Constraint Profile one per program | CTA-5001-F 5, 7.2.2 | WAVE-5 |
| SEG-WAVE-SPLICE | info | Sequential switching sets that do not meet the optional WAVE Baseline Splice Constraint Profile (same media profile and sample entry type) or its recommendations (same picture aspect ratio and frame rate family) | CTA-5001-F 7.2.2 | WAVE-4, WAVE-6, WAVE-7, WAVE-9, WAVE-17, WAVE-21 |

## DASH-HLS interoperability (CTA-5005)

| Rule | Severity | Description | Spec | DASH-IF checks |
|---|---|---|---|---|
| SEG-HLS-BITRATE | warning | The average bitrate of each CMAF fragment should be within 10% of that of the track on demand, and live at most 110% of that of the newest hour | CTA-5005-B 4.1.2 | WAVEHLS-2 |
| SEG-HLS-CBCS-VARIANT | error | Each cenc-encrypted component needs an equivalent cbcs-encrypted component for HLS | CTA-5005-B 4.3.2, 4.3.3 | WAVEHLS-5 |
| SEG-HLS-ENCRYPTION | error | Encrypted tracks must use cbcs with constant 16-byte IVs and a 1:9 pattern for video and 10:0 for audio, or cenc with 8-byte IVs, and address sample auxiliary information with saio | CTA-5005-B 4.3.2 | WAVEHLS-3, WAVEHLS-4, WAVEHLS-6, WAVEHLS-7, WAVEHLS-8, WAVEHLS-9 |
| SEG-HLS-KEY-ROTATION | error | A segment with a key other than the default_KID must describe it in sgpd (seig) and sbgp boxes, and carry pssh boxes if the CMAF header has them | CTA-5005-B 4.6.2 | WAVEHLS-10, WAVEHLS-11 |
| SEG-HLS-ROLES | error | Track roles must be stored in kind boxes in the CMAF header, using the DASH Role scheme when possible | CTA-5005-B 4.7.2 | WAVEHLS-18, WAVEHLS-19 |
| SEG-HLS-SEGMENT-KEY | error | Each CMAF segment must use a single encryption key and initialization vector | CTA-5005-B 4.3.2 | WAVEHLS-12, WAVEHLS-13 |
| SEG-HLS-SIDX | error | CMAF track files delivered to clients must have a single sidx box between the CMAF header and the fragments | CTA-5005-B 4.1.2 | WAVEHLS-1 |
| SEG-HLS-SPLICE | error | At a splice point (Period end), the last CMAF fragment must end at the splice point within one sample duration | CTA-5005-B 4.4.2 | WAVEHLS-14, WAVEHLS-15 |
| SEG-HLS-TEXT | error | Text components must be IMSC1 Text, IMSC1.1 Text or WebVTT tracks | CTA-5005-B 4.1.2 | WAVEHLS-16 |
