# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1] - 2026-10-23

### Added
- MPD-XLINK-UNRESOLVED (info) for remote Periods, which dashcheck does not resolve. They are
  no longer MPD-SEGINFO-NONE errors.
- ADDR-UNSUPPORTED (info) for Representations that need Annex I URL parameters
  (`urn:mpeg:dash:urlparam:2014` or `2016`). Their segments are no longer fetched.

### Fixed
- MPD-UPDATE-PUBLISHTIME ignores the time in a `urn:mpeg:dash:utc:direct:2014` UTCTiming,
  which changes with each response.
- CLOCK-SKEW allows for whole seconds from utc:direct, http-iso and http-xsdate time sources.
- MPD-CODECS no longer warns about images such as thumbnail tiles, which have no codecs parameter.
- SEG-EPT and SEG-DURATION leave out media that the edit list cuts, such as audio priming. With an
  edit list, an `S@t` equal to the tfdt is SEG-EPT, not SEG-EPT-CTO.
- LIVE-STALE measures a Period that has ended at its end. Earlier Periods of a multi-Period live
  MPD are no longer reported as stale.
- MPD-LL-MIMETYPE accepts `application/mp4` for text, the media type of CMAF text tracks.
- MPD-XSD accepts a space-separated list in `ContentSteering@defaultServiceLocation`, as
  ISO/IEC 23009-1 and ETSI TS 103 998 define it.
- `@availabilityTimeOffset` alone no longer turns on the low-latency rules. Without the
  low-latency profile, `@availabilityTimeComplete` false does.

## [0.2.0] - 2026-10-21

### Added
- SEG-CONTINUITY-FRAGMENT: each chunk or fragment in a segment must start at the previous one's
  tfdt plus its sample durations (ISO/IEC 23000-19 7.3.2.2, 7.3.2.3).
- CMAF brand rules: SEG-CMAF-MINORVERSION, and for the cmf2 brand SEG-CMF2-ELST, SEG-CMF2-TRUN
  and SEG-CMF2-DEFAULTS (each chunk sets sample durations, sizes and flags, not only trex).
- `-sequence-numbers`, also in the browser tool's options: checks the convention of one mfhd
  sequence_number per segment, shared by its chunks and increasing (SEG-SEQUENCE-NUMBER).
- MPD-DVB-SCTE-BINARY: an SCTE-35 event must carry a base64 splice_info_section in its
  Signal/Binary element.
- SEG-CMAF-CHUNK (one traf with one trun per moof) and SEG-CMAF-SAP (video CMAF fragments start
  with a sync sample). The CMAF rules now also apply to low-latency chunked Representations.
- XCHK-CMAF-SIZE: in the DASH profile for CMAF content, `@maxWidth` and `@maxHeight` (or `@width`
  and `@height`) must be the tkhd size of the largest track.
- SEG-SIDX-SPAN: the sidx box of a SegmentBase Representation must index every movie fragment of
  its file.
- Segment Sequence MPD rules: MPD-SSR-SIGNAL (a profile that allows them or the
  `urn:mpeg:dash:ssr:2023` EssentialProperty), MPD-SSR-TIMELINE, MPD-SSR-ATTRIBUTES and
  MPD-SSR-PROPERTIES.
- SEG-SSR-SAP: the Partial Segments that SegmentSequenceProperties@cadence selects must start with
  a sync sample when `@sapType` is 1 or 2.
- SEG-CMAF-SINF: every sample entry of an encrypted CMAF track must be protected (no clear sample
  entry for a clear lead), with tenc default_isProtected 1.
- ContentProtection rules: XCHK-CP-MISSING, XCHK-CP-SCHEME and XCHK-CP-KID against the init segment,
  MPD-CP-CENC, MPD-CP-ROBUSTNESS, MPD-CP-REF, MPD-CP-CLEARKEY and MPD-CP-DEFAULTKID, and per profile
  SEG-AS-KID, SEG-IOP-PSSH, SEG-IOP-PSSH-INIT, MPD-IOP-CP-DRM and MPD-DVB-CP-DRM.
- `docs/rules.md` documents all rules with the DASH-IF checks each covers, and marks the rules that
  the DASH-IF Conformance tool does not have.

### Changed
- Segment Sequences (ISO/IEC 23009-1 5.3.9.7, `@k` and `$SubNumber$`) are now addressed and their
  Partial Segments checked, with `@startWithSAP` on the first of each sequence. Only SegmentTimeline
  duration patterns (S@p, S@ssp) remain ADDR-UNSUPPORTED.
- XCHK-CP-MISSING replaces SEG-LL-PROTECTION and checks all encrypted tracks, not only low latency.
- SEG-HLS-BITRATE (`-hls-interop`) also checks SegmentTemplate and SegmentList content when every
  segment is checked, and live streams whose checked segments cover an hour (at most 110%).

### Fixed
- ContentProtection@ref is resolved, so MPD-CP-VALUE, MPD-IOP-CP and SEG-DVB-PSSH see the `@value`,
  cenc:default_KID and cenc:pssh that a descriptor inherits from its source.
- MPD-CP-VALUE accepts a version (`cenc:1`) and a percent-encoded 4CC in the mp4protection `@value`.
- SEG-CMAF-SENC no longer reports clear-lead fragments, or audio with constant IVs, which has no
  sample auxiliary information. A track's protection comes from its first protected sample entry.
- `file:` URLs: a byte range that ends after the file is cut at its end, as over HTTP.
- MPD-XSD checks the XLink attributes against xlink.xsd (such as the `xlink:actuate` values), and its
  messages give the invalid value and what was expected (jacoelho/xsd v0.6.7).

## [0.1.2] - 2026-10-19

### Fixed
- The dot of LIVE TIMING in the logo no longer touches the text.

## [0.1.1] - 2026-10-19

### Added
- The browser tool shows the dashcheck logo, has a favicon, and an image for link previews.

## [0.1.0] - 2026-10-19

### Added
- `dashcheck` command and browser tool (WebAssembly) that check an MPEG-DASH MPD, its segments
  and, for live streams, the segment availability against a synchronized clock.
- MPD checks: the 6th edition XML schema and semantic rules for DASH, DASH-IF IOP, CMAF,
  low latency, DVB-DASH, HbbTV, CTA WAVE and Dolby, selected by `@profiles`.
- `-profiles`, and "Check against" in the browser tool: choose the profiles to check (DASH-IF IOP,
  CMAF, low latency, DVB-DASH 2014/2017, HbbTV, CTA WAVE), with or without those in MPD@profiles.
- Segment checks with mp4ff, cross-checked against the MPD timeline and attributes.
- `-updates`: refetches a dynamic MPD and checks the update against the version before.
- `-hls-interop`: the DASH-HLS interoperability rules of CTA-5005.
- 380 of the 408 checks of the DASH-IF Conformance tool, and 6 in part; see `docs/dashif-coverage.md`.
- `-short`: a FAIL, WARN or OK line per MPD instead of the report, also for several MPDs such as
  `dashcheck -short streams/*.mpd`, checked `-parallel` at a time.
- `-json`: the report, or with `-short` an array of reports, with error, warning and info counts.
- `-version`, and the version in the footer of the browser tool.
- The browser tool, published on GitHub Pages for each release, counts visits and checks with
  GoatCounter, without cookies or MPD URLs.

[Unreleased]: https://github.com/tobbee/dashcheck/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/tobbee/dashcheck/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/tobbee/dashcheck/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/tobbee/dashcheck/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/tobbee/dashcheck/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/tobbee/dashcheck/releases/tag/v0.1.0
