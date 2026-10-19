# dashcheck

A DASH conformance checker in Go that runs as a CLI and in the browser (wasm),
meant to replace the DASH-IF Conformance tool. See README.md for the overview.

## Design rules

- No check does I/O or reads the clock. Everything goes through `fetch.Fetcher`
  and `clock.Clock`, so checks are identical in the CLI and the browser and can
  be tested with test servers and fixed clocks.
- `fetch/js.go` must not import `net/http` (it adds about 7 MB to the wasm).
- The page counts visits and checks with GoatCounter by calling its /count
  endpoint from `web/app.js`, not with count.js, which also sends the query
  string. Never send the query string or an MPD URL: `?url=` holds one.
  Run `make check`, which also lints the js/wasm build.
- Each check is a `report.Rule` registered in a package-level var, with a stable
  ID and a spec reference. Prefixes: `MPD-` (mpdcheck), `ADDR-` (addressing),
  `INIT-`, `SEG-`, `XCHK-` (segcheck), `LIVE-`, `HTTP-` (runner), `CLOCK-`.
- Severity follows the spec wording: SHALL is an error, SHOULD a warning,
  information is info. MAY statements are not checked (n/a in the inventory).
- Representations come from `addressing.Reps`, which resolves inheritance
  (profiles, templates, codecs, ATO/ATC, ...). Add fields there rather than
  re-deriving inherited values in each rule.
- Rules that need to know whether an attribute is present, or attributes the
  dash-mpd model does not keep, use the raw tree in `mpdcheck/rawxml.go`.
- Gate profile-specific rules on the effective @profiles (`Rep.Profiles`, or
  `md.Profiles` and `effective` in mpdcheck), so that `-profiles` works: the
  `spec` package rewrites @profiles of the parsed MPD for the user's choice.
  Gates on other signals (CMAF brands, chunking) must honour
  `Rep.OnlyProfiles`. A new profile family goes into `spec.Profiles`.

## Porting DASH-IF checks

The inventory of all 408 upstream checks is `docs/dashif-checks.json`, at
upstream commit e311ec6e12b21394355fd1749101590400e31cbb (2026-10-16).

1. Read the upstream code at that commit, for example
   `gh api -H 'Accept: application/vnd.github.raw' "repos/Dash-Industry-Forum/DASH-IF-Conformance/contents/<path>?ref=e311ec6e12b21394355fd1749101590400e31cbb"`
   (or the local clone in `references/`). Port the intended rule, not the
   upstream bugs, and check the rule text and clause number in the spec
   (see Standards below): many upstream rules are outdated or stricter than the spec.
2. Write the rule and a table-driven test: a valid fixture MPD plus one
   mutation per case. Filter `MPD-XSD` when a mutation is schema-invalid on purpose.
3. Run it on real streams to catch false positives (see below).
4. Record the status with `tools/set-status.py` and note every difference from
   upstream, then `go test ./internal/coverage -update`.
5. Add every upstream bug found to `docs/dashif-upstream-bugs.md` right away,
   with a commit-pinned link, problem, effect and fix.
6. Commit one area per commit (see the user's commit rules).

Remaining at the last update: only DVB AVS3 (DVB-60 to DVB-64), which needs
AVS3 support in mp4ff, plus six partial checks (see the coverage notes).
Rules for specifications that MPDs do not signal are opt-in options
(`-hls-interop` for CTA-5005). Segment Sequences (@k, $SubNumber$) are not
addressed yet (ADDR-UNSUPPORTED).
`docs/dashif-coverage.md` has the current numbers.

## Standards

The user's private standards library is at
`~/Library/Mobile Documents/com~apple~CloudDocs/standards/` (the
`lookup-standard` skill searches it). Text extractions made with
`pdftotext -layout` are kept in `references/text/`, which is gitignored
and must not be committed or quoted at length:

- `iso-23009-1-6ed-fdis.txt` (DASH; from `MPEG/DASH/23009_DASH/Part1/6ed_2025/`)
- `etsi-ts-103285-v1.4.1.txt`, `etsi-ts-103285-v1.3.1.txt` (DVB-DASH; `DVB/`)
- `etsi-ts-101154-v2.9.1.txt` (DVB codecs, annex L has the DASH player
  conformance points and bitstream rules; `DVB/`)
- `dashif-cr-low-latency-live-r8.txt` (DASH-IF low latency, clauses 9.X; `DASH-IF/`)
- `cta-5001-f.txt` (CTA WAVE; `CTA/WAVE/`), `iso-23000-19-2024-amd1.txt` (CMAF amendment)
- `iso-23000-19-2023-fdis.txt` (CMAF 3rd edition FDIS; `MPEG/23000-19 CMAF/`) and
  `iso-14496-12-2022.txt` (ISO BMFF; `SIS/.../SS-ISO_IEC 14496-12_2022.pdf`)

Find a clause with `grep -n` and read around it with `sed -n`. Extract more
documents the same way when a rule needs them.

MPD update rules (`mpdcheck/update.go`) compare two fetches of a dynamic MPD;
the runner refetches it (`update.go`, `-updates`). Put further two-version
rules there.

## Commands

```sh
go test ./...                          # all tests
make check                             # golangci-lint, including js/wasm
go test ./internal/coverage -update    # regenerate docs/dashif-coverage.md
go build -o out/dashcheck ./cmd/dashcheck && out/dashcheck -n 2 <MPD URL>
make serve PORT=8090                   # browser UI on http://localhost:8090
```

Add user-visible changes to CHANGELOG.md under [Unreleased]. The version comes
from the git tag (`internal.Version`); a release is a CHANGELOG section and a
`vX.Y.Z` tag. The pre-commit hook (`make pre-commit-install`) runs gofmt,
golangci-lint and the tests; its venv is `.venv`, since Go would see the Go file
pre-commit ships in a `venv/` directory as a package.

## Test streams

- livesim2 live: `https://livesim2.dashif.org/livesim2/testpic_2s/Manifest.mpd`,
  with URL options `segtimeline_1/`, `chunkdur_0.25/ato_7/testpic4_8s/Manifest300.mpd`
  (low latency), `periods_60/`, `utc_none/`, and faults `timeoffset_10/`,
  `timeoffset_-10/`, `statuscode_[{cycle:2,rsq:0,code:404,rep:V300}]/` (URL-encode the brackets).
- On-demand with sidx: `https://storage.googleapis.com/shaka-demo-assets/angel-one/dash.mpd`.
- CMAF: `https://dash.akamaized.net/WAVE/vectors/cfhd_sets/12.5_25_50/t1/2022-10-17/stream.mpd`.
- DVB-DASH: `https://rdmedia.bbc.co.uk/testcard/vod/manifests/avc-full.mpd`
  and `.../testcard/simulcast/manifests/avc-full.mpd` (live).
- DASH-IF IOP: `https://dash.akamaized.net/dash264/TestCasesIOP41/MultiTrack/alternative_content/1/manifest_alternative_content_live.mpd`.

## Known quirks of the dependencies

- dash-mpd's XML parser accepts malformed XML; the schema check catches it.
- dash-mpd rejects xs:duration values with years or months.
- dash-mpd reads absent booleans and numbers as their zero value; use the raw tree.
- mp4ff's `Sample.IsSync` requires `sample_depends_on` = 2; use `segcheck.isSync`.
- jacoelho/xsd v0.6.4 does not check the enumeration of `xlink:actuate` with
  the DASH schema: `xlink:actuate="bogus"` validates (xmllint rejects it),
  while `xlink:type` and `xlink:show` fixed values are checked. A minimal
  schema with the same xlink declarations is checked, so something else in
  DASH-MPD.xsd triggers it. Ruled out: the attribute's default, the attribute
  wildcard, the second import of xlink.xsd, and DASH-MPD-UP.xsd. To bisect
  further and report upstream; `TestMutations` in `schema/` skips the case
  and fails once it is caught.
- When editing with scripted search and replace, remember that gofmt realigns
  struct fields and composite literals after earlier edits.
