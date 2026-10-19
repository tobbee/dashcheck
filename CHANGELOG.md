# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/tobbee/dashcheck/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/tobbee/dashcheck/releases/tag/v0.1.0
