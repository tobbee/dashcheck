# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- SEG-CONTINUITY-FRAGMENT: each chunk or fragment in a segment must start at the previous one's
  tfdt plus its sample durations (ISO/IEC 23000-19 7.3.2.2, 7.3.2.3).
- CMAF brand rules: SEG-CMAF-MINORVERSION, and for the cmf2 brand SEG-CMF2-ELST, SEG-CMF2-TRUN
  and SEG-CMF2-DEFAULTS (each chunk sets sample durations, sizes and flags, not only trex).
- `-sequence-numbers`, also in the browser tool's options: checks the convention of one mfhd
  sequence_number per segment, shared by its chunks and increasing (SEG-SEQUENCE-NUMBER).

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

[Unreleased]: https://github.com/tobbee/dashcheck/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/tobbee/dashcheck/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/tobbee/dashcheck/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/tobbee/dashcheck/releases/tag/v0.1.0
