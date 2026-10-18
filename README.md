# dashcheck

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

All findings carry a stable rule ID; `dashcheck -rules` lists them.

## Command line

```sh
go install github.com/tobbee/dashcheck/cmd/dashcheck@latest
dashcheck https://livesim2.dashif.org/livesim2/segtimeline_1/testpic_2s/Manifest.mpd
dashcheck -n 10 -level warning path/to/local/Manifest.mpd
dashcheck -json https://example.com/stream.mpd
```

The exit code is 1 if any error was found, which makes it usable in CI.

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

* Port the remaining DASH-IF checks, Schematron rules first.
* Profile-specific checks (CMAF, DVB, HbbTV, low latency) and bandwidth checks.
* MPD update checks for live: fetch the MPD repeatedly and check the allowed
  changes between versions.
* Chunk arrival timing for low-latency streams.
* A long-running monitor mode.
