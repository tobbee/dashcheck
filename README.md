# dashcheck

![Test](https://github.com/tobbee/dashcheck/workflows/Go/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/tobbee/dashcheck/badge.svg?branch=main)](https://coveralls.io/github/tobbee/dashcheck?branch=main)
[![Go Reference](https://pkg.go.dev/badge/github.com/tobbee/dashcheck.svg)](https://pkg.go.dev/github.com/tobbee/dashcheck)
[![license](https://img.shields.io/github/license/tobbee/dashcheck.svg)](https://github.com/tobbee/dashcheck/blob/main/LICENSE)

dashcheck checks MPEG-DASH streams: the MPD, its segments, and for live
streams the timing against a synchronized clock. The same Go code runs as a
command-line tool and in the browser as WebAssembly.

It checks:

* **MPD syntax:** well-formed XML and the MPEG-DASH 6th edition XML schema,
  with line and column for each error.
* **MPD rules:** a growing set of semantic rules on the
  [dash-mpd](https://github.com/Eyevinn/dash-mpd) model, such as unique ids,
  SegmentTemplate consistency, SegmentTimeline gaps and overlaps, and UTCTiming.
* **Segments:** init and media segments parsed with
  [mp4ff](https://github.com/Eyevinn/mp4ff), and cross-checked against the
  MPD: `@codecs` against the decoder configuration, size, sample rate, handler,
  earliest presentation time and duration against the MPD timeline, continuity
  between segments, and sync samples at segment start.
* **Live timing:** the clock is synchronized to the MPD's UTCTiming source,
  as a player would do, and compared with a reference clock. The newest segments
  in the availability window must be available, the next one must not be
  available too early, and a SegmentTimeline must reach the live edge.
* **MPD updates:** a live MPD is fetched again after about one segment
  (`-updates N` sets how many times) and compared with the version before:
  @publishTime, ids, Representation attributes and the timing of segments that
  both versions describe, plus the DVB-DASH update constraints.
* **Profiles:** DASH-IF IOP, CMAF, low latency, DVB-DASH 2014 and 2017, HbbTV
  and CTA WAVE, selected by the MPD's @profiles. `-profiles` chooses them
  instead: `-profiles auto,dvb` adds DVB-DASH to the profiles the MPD
  signals, and `-profiles dvb,hbbtv` checks only DVB-DASH and HbbTV, as if
  MPD@profiles listed them. The DASH-HLS interoperability rules of CTA-5005,
  which MPDs do not signal, are added with `-hls-interop`.

All findings carry a stable rule ID; `dashcheck -rules` lists them.

## Command line

```sh
go install github.com/tobbee/dashcheck/cmd/dashcheck@latest
dashcheck https://livesim2.dashif.org/livesim2/segtimeline_1/testpic_2s/Manifest.mpd
dashcheck -n 10 -level warning path/to/local/Manifest.mpd
dashcheck -json https://example.com/stream.mpd
dashcheck -profiles auto,hbbtv https://example.com/stream.mpd
dashcheck -short streams/*.mpd
dashcheck -short -json $(cat urls.txt) > results.json
```

The exit code is 1 if any error was found, which makes it usable in CI.

`-short` prints one line per MPD instead of the report, and takes several
MPDs, which it checks `-parallel` at a time. `-level` adds the findings below
each line.

```
$ dashcheck -short testdata/*.mpd testdata/*/*.mpd
FAIL  testdata/malformed.mpd  (3 errors, 0 warnings, 0 info)
OK    testdata/testpic_2s/Manifest.mpd  (0 errors, 0 warnings, 0 info)

2 MPDs: 1 with errors, 0 with warnings, 1 OK
```

With `-json`, `-short` writes an array of reports, each with its counts.
`dashcheck -version` prints the version, which Go takes from the git tag and
commit at build time.

## Browser

```sh
make serve   # builds web/dashcheck.wasm and serves web/ on http://localhost:8080 (localhost only)
```

The page runs the checker in a Web Worker. It can check a URL, or a local
folder with packager output, which is read in the browser and never uploaded.
A URL can be passed as `?url=...` to share a check.

Browser limits: the stream must send CORS headers (almost all DASH CDNs do,
since players need them), and an https page cannot fetch `http://` URLs. Use
the command-line tool for such streams.

Each release is published on GitHub Pages by `.github/workflows/pages.yml`
(it can also be run by hand). The published page counts visits and checks with
[GoatCounter](https://www.goatcounter.com), without cookies: it sends the page
path and title, the screen width, the referrer without its query string, and
for a check whether it was a URL or a folder, never an MPD URL.

## Design

```
cmd/dashcheck        CLI (net/http, local files)
cmd/dashcheck-wasm   wasm entry point for the browser
web/                 static page and Web Worker
dashcheck            runner: fetch MPD, check it, sync clock, check segments
mpdcheck             MPD rules (schema/ for XSD validation)
addressing           MPD to segment lists, availability times
segcheck             init and media segment checks with mp4ff
clock                UTCTiming measurement and synchronized clock
fetch                Fetcher interface: native and browser fetch
report               rules, findings, report
spec                 profiles to check, chosen or from @profiles
```

No check does I/O or reads the clock directly. Everything goes through
`fetch.Fetcher` and `clock.Clock`, so the checks are identical in all hosts and
can be tested with fixed clocks and test servers.

XSD validation uses [jacoelho/xsd](https://github.com/jacoelho/xsd), a pure Go
validator. The schema files in `schema/xsd` come from
[MPEGGroup/DASHSchema](https://github.com/MPEGGroup/DASHSchema), with the
internal DTD entities expanded.

## Status and next steps

This is an early version. [docs/dashif-coverage.md](docs/dashif-coverage.md)
lists all 408 checks of the DASH-IF Conformance tool and which dashcheck rules
cover them. It is generated from `docs/dashif-checks.json`; after changing that
file, run `go test ./internal/coverage -update`. To refresh the list from a newer
version of the DASH-IF tool, run `tools/dashif-inventory.py <clone> docs/dashif-checks.json`.
Bugs found in the DASH-IF tool while porting its checks are listed in
[docs/dashif-upstream-bugs.md](docs/dashif-upstream-bugs.md), ready to be reported.

Planned next steps:

* Port the remaining DASH-IF checks (DVB AVS3, which needs AVS3 support in mp4ff).
* Bandwidth checks.
* Chunk arrival timing for low-latency streams.
* A long-running monitor mode.

## Development

```sh
make             # golangci-lint, build, wasm and tests
make coverage    # coverage.html and coverage.txt
make codespell
make pre-commit-install   # gofmt, golangci-lint and tests before each commit
```

Changes are listed in [CHANGELOG.md](CHANGELOG.md).

## License

MIT, see [LICENSE](LICENSE). The XSD files in `schema/xsd` come from
[MPEGGroup/DASHSchema](https://github.com/MPEGGroup/DASHSchema) under its own
license.
