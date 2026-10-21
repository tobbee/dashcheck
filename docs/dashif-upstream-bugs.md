# Bugs found in the DASH-IF Conformance tool

These bugs were found while porting the checks of the
[DASH-IF Conformance tool](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance)
to dashcheck, at commit
[e311ec6](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/tree/e311ec6e12b21394355fd1749101590400e31cbb)
(2026-10-16). Each entry is written so that it can be filed as an issue.
Check IDs such as SCH-14 refer to [dashif-checks.json](dashif-checks.json).

**Verified in code** means the problem follows from reading the code; **found by
testing** means it was observed when running the rules. None has been reported
yet unless stated.

| # | Where | Problem | Status |
|---|---|---|---|
| 1 | XLink resolver | All MPD-level elements are dropped before any check runs | Verified in code |
| 2 | SCH-3 | Ignores the last Period@duration | Verified in code |
| 3 | SCH-14 | Fails for every zero-duration Period with an AdaptationSet | Verified in code |
| 4 | SCH-41 | Ignores SegmentTemplate inheritance | Verified in code |
| 5 | SCH-52 | Misspelled attribute, never fires | Verified in code |
| 6 | SCH-53 | maxSegmentDuration summed without units | Verified in code |
| 7 | SCH-57 | Outdated list of Role values | Verified in code |
| 8 | SCH-4 | Outdated list of known profiles | Verified in code |
| 9 | SCH-71 | Fails for a Period without AdaptationSets | Verified in code |
| 10 | SCH-75, SCH-77 | SRD bounds: wrong quantifier, NaN, type error | Verified in code |
| 11 | SCH-90 | Matches any SupplementalProperty | Verified in code |
| 12 | SCH-98 | Misspelled profile and attribute, never fires | Verified in code |
| 13 | SCH-100, SCH-111, SCH-113 | Profile subsets compared as substrings | Verified in code |
| 14 | SCH-102 | Missing `@`, always fails | Verified in code |
| 15 | SCH-110 | Type error with @codecs on several levels | Found by testing |
| 16 | schematron.sch | Saxon aborts the whole report on dynamic errors | Found by testing |
| 17 | dvb-dash.sch 1–3 | Compare each Representation with itself | Verified in code |
| 18 | dvb-dash.sch 9–13 | Rule context never matches Representations | Verified in code |
| 19 | ac4-generic.sch 2 | Inverted @mimeType test | Verified in code |
| 20 | IOP Live | Misspelled profiles, never fires | Verified in code |
| 21 | IOP CrossAdaptation | AVC level compared hex with decimal, always fails | Verified in code |
| 22 | CMAF CompositionTimes | Requires elst without composition offsets | Verified in code |
| 23 | CMAF Durations | Ignores per-sample durations | Already reported: #772 |
| 24 | CMAF Subtitles | Not implemented, always fails | Verified in code |
| 25 | CMAF VideoMediaProfile | Frame rate never checked, cud8 always fails, cud1 BT.2020 rejected | Verified in code |
| 26 | CMAF SubtitleMediaProfile | Skips CMAF subtitles | Verified in code |
| 27 | Low Latency LeapSecond | Misspelled element name, always fails | Verified in code |
| 28 | Low Latency SegmentURL | SegmentTimeline never valid | Verified in code |
| 29 | Low Latency SegmentOrChunked | Mode chosen by presence, not value | Verified in code |
| 30 | Low Latency booleans | Compares with TRUE/FALSE, always fails | Verified in code |
| 31 | Low Latency SegmentAdaptation | Duration limits inverted | Verified in code |
| 32 | Low Latency placeholders | Checks that always pass or always fail | Verified in code |
| 33 | DVB AudioChecks | Common attribute checks test an empty array, always fail | Verified in code |
| 34 | DVB AudioChecks | Reversed arguments, AC-4 channel configuration never checked | Verified in code |
| 35 | DVB SCTEChecks | Optional @presentationTimeOffset required | Verified in code |
| 36 | DVB SCTEChecks | Indefinite @duration never detected | Verified in code |
| 37 | DVB SCTEChecks | Fractional frame rates truncated | Verified in code |
| 38 | DVB LegacyCodecs | AVC profile and level required of every Representation | Found by testing, verified in spec |
| 39 | DVB Subtitle | Skips stpp with codecs parameters | Verified in code |
| 40 | DVB CrossAudio | Channel count fails with other schemes | Verified in code |
| 41 | DVB LegacyCodecs | HEVC tier, profile, level and bit depth rules do not match the spec | Verified in code and spec |
| 42 | SCH-7 | Requires a static MPD for the extended live profile | Verified in spec |
| 43 | DVB ContentProtection | Outdated cenc-only scheme rule and a pssh SHOULD not in the spec | Verified in spec |
| 44 | DVB UpdateConstraints | Several MPD update checks can never fail, others fail wrongly | Verified in code |
| 45 | DVB BitStream | Colour checked against 'colr' instead of the VUI; HDR checks never fail; @codecs too strict | Verified in code and spec |
| 46 | DVB SEILocations | hvc1 never checked; prefix SEI after VCL not found; wrong VVC NAL types | Verified in code and spec |
| 47 | CTAWave | Outdated: required profiles per AdaptationSet, splice constraints always enforced | Verified in spec |
| 48 | WaveHLSInterop | Valid cenc tracks fail; sidx required in every segment; key count wrong | Verified in code and spec |
| 49 | HbbTV | Codec and AVC rules outdated and applied per Representation | Verified in spec |
| 50 | Common XSDValidation | The DVB 2018 schema lacks UTCTiming | Verified in code |
| 51 | DVB Codecs, LegacyCodecs | Codec lists lack VVC, AVS3, MPEG-H and DTS-UHD | Verified in spec |
| 52 | CMAF Durations | First tfdt of a segment compared with every fragment | Verified in code |
| 53 | Low Latency DASHProfile | @maxWidth and @maxHeight compared with every track; @width ignored | Verified in code and spec |
| 54 | WaveHLSInterop Bitrate | On-demand rule applied to live and to the downloaded segments only | Verified in code and spec |
| 55 | Low Latency DASHProfile | KIDs compared in different formats; cenc:pssh read as an attribute | Verified in code and spec |
| 56 | SCH-55, SCH-56 | ContentProtection @value checked only by its length | Verified in code and spec |
| 57 | CMAF LegacyInitialization | Counts senc boxes against moof boxes | Verified in code and spec |

## Manifest handling

### 1. XLink resolution drops all MPD-level elements

**Where:** [app/Services/Manifest/XLink.php#L56-L71](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Services/Manifest/XLink.php#L56-L71),
used by [app/Services/MPDCache.php#L82](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Services/MPDCache.php#L82)

**Problem:** `resolveAndValidate()` copies the MPD element with `cloneNode(false)`
and then appends only the Period elements. All other children of MPD are lost:
BaseURL, Location, PatchLocation, UTCTiming, ServiceDescription,
LeapSecondInformation, ProgramInformation, ContentProtection, EssentialProperty,
SupplementalProperty and Metrics.

**Effect:** `MPDCache::getMPD()` returns this resolved document for all DOM
queries, so every check sees an MPD without these elements. Expected effects (not
run): checks that look for MPD-level UTCTiming, ServiceDescription or
LeapSecondInformation (DVB 4.7.2, Low Latency 9.X.4.2) fail for conforming
streams, and MPD-level BaseURL is missing from segment URL resolution.

**Fix:** Clone the MPD element deeply and replace each Period in place with its
resolved version.

## Schematron rules (schematron/schematron.sch)

### 2. SCH-3 ignores the last Period@duration

**Where:** [schematron.sch#L71](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L71)

**Problem:** The rule requires @mediaPresentationDuration or @minimumUpdatePeriod.
ISO/IEC 23009-1 also accepts a @duration on the last Period, which SCH-6
([line 83](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L83)) checks correctly.

**Effect:** False errors for static MPDs whose length is given by Period@duration.

**Fix:** Remove SCH-3; SCH-6 has the correct condition.

### 3. SCH-14 fails for any zero-duration Period with an AdaptationSet

**Where:** [schematron.sch#L106](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L106)

**Problem:** `count(child::dash:AdaptationSet)` is used as a boolean, which is
true for one AdaptationSet or more. The message says at most one is allowed.

**Effect:** A Period with @duration 0 and a single AdaptationSet fails.

**Fix:** Use `count(child::dash:AdaptationSet) > 1`.

### 4. SCH-41 ignores SegmentTemplate inheritance

**Where:** [schematron.sch#L186](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L186)

**Problem:** Each SegmentTemplate element must have @duration, SegmentTimeline or
@initialization by itself. SegmentTemplate attributes are inherited between the
Period, AdaptationSet and Representation levels.

**Effect:** False errors when, for example, @media is on the AdaptationSet and
@duration on the Representation.

**Fix:** Evaluate the combined template of each Representation.

### 5. SCH-52 tests a misspelled attribute and never fires

**Where:** [schematron.sch#L220](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L220)

**Problem:** The test reads `@timeShiftbuffer` instead of `@timeShiftBufferDepth`,
and `dash:MPD/@timeShiftBufferDepth` is relative to SegmentBase, so it is empty.

**Effect:** A segment-level @timeShiftBufferDepth smaller than the MPD's is never reported.

**Fix:** Compare `@timeShiftBufferDepth` with `ancestor::dash:MPD/@timeShiftBufferDepth` as durations.

### 6. SCH-53 adds duration components without unit conversion

**Where:** [schematron.sch#L229](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L229)

**Problem:** MPD@maxSegmentDuration is converted to seconds by adding
years, months, days, hours, minutes and seconds without multiplying them.

**Effect:** `PT1M` counts as 1 second, so segments longer than 1 s fail.

**Fix:** Use `xs:dayTimeDuration(ancestor::dash:MPD/@maxSegmentDuration) div xs:dayTimeDuration('PT1S')`.

### 7. SCH-57 knows only seven Role values

**Where:** [schematron.sch#L255](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L255)

**Problem:** The rule accepts caption, subtitle, main, alternate, supplementary,
commentary and dub. Later editions of ISO/IEC 23009-1 add description, sign,
metadata, enhanced-audio-intelligibility, emergency, forced-subtitle, easyreader
and karaoke.

**Effect:** False errors for valid Role values such as forced-subtitle.

**Fix:** Extend the list to the current edition.

### 8. SCH-4 has an outdated list of profiles

**Where:** [schematron.sch#L12-L55](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L12-L55)

**Problem:** Among others, the DVB interoperability point profiles
(`urn:dvb:dash:profile:dvb-dash:isoff-ext-live:2014`, `...:isoff-ext-on-demand:2014`),
the DASH-IF low-latency profile and `urn:mpeg:dash:profile:advanced-linear:2025`
are missing. ISO/IEC 23009-1 also allows profiles that a validator does not know.

**Effect:** Errors for valid MPDs.

**Fix:** Extend the list, and report unknown profiles as information.

### 9. SCH-71 fails for a Period without AdaptationSets

**Where:** [schematron.sch#L324](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L324)

**Problem:** `every $AdaptationSet in child::dash:AdaptationSet satisfies ...` is
true for an empty set.

**Effect:** A Period without AdaptationSets (for example one resolved to zero) fails the SRD rule.

**Fix:** Add `child::dash:AdaptationSet and` before `every`.

### 10. SCH-75 and SCH-77 check the SRD bounds wrongly

**Where:** [schematron.sch#L332](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L332)
and [#L336](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L336)

**Problem:** The rules use `some $source_id ... satisfies`, so they pass when
any one source meets the bound instead of every source. The total size is taken
with `tokenize(descendant::...[...]/@value, ',')`: with no descriptor giving the
total size it is NaN and the rule fails, and with two such descriptors for the
source `tokenize()` gets a sequence and raises a type error (XPTY0004).

**Effect:** Missed errors, false errors, and aborted Schematron runs.

**Fix:** Use `every`, and take the total size from one descriptor (`[1]`).

### 11. SCH-90 matches any SupplementalProperty value

**Where:** [schematron.sch#L384](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L384)

**Problem:** Any SupplementalProperty whose @value equals the id of a later
AdaptationSet is taken as an adaptation set switching signal. The scheme
`urn:mpeg:dash:adaptation-set-switching:2016` is not checked, and @value is a
comma-separated list that is compared as a whole.

**Effect:** Unrelated descriptors trigger the rule, and switching with more than one AdaptationSet is missed.

**Fix:** Require the switching scheme and tokenize @value.

### 12. SCH-98 misspells the profile and the attribute

**Where:** [schematron.sch#L400](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L400)

**Problem:** The profile is written `isoff-ondemand:2011` instead of
`isoff-on-demand:2011`, and the attribute `subSegmentAlignment` instead of
`subsegmentAlignment`.

**Effect:** The rule never fires.

**Fix:** Correct both names.

### 13. SCH-100, SCH-111 and SCH-113 compare profile lists as substrings

**Where:** [schematron.sch#L404](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L404),
[#L434](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L434),
[#L444](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L444)

**Problem:** `contains(ancestor::dash:MPD/@profiles, @profiles)` tests whether
the whole lower-level value is a substring of the higher-level one.

**Effect:** A valid subset in a different order, or with different spacing, fails.

**Fix:** Tokenize both values and require every lower-level profile to be in the higher-level list.

### 14. SCH-102 tests a child element instead of the attribute

**Where:** [schematron.sch#L408](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L408)

**Problem:** `subsegmentStartsWithSAP` lacks the `@`, so it selects a child
element that never exists. The value test
`not(... = '1') or not(... = '2')` is also always true.

**Effect:** Every AdaptationSet of an MPD with the IOP on-demand profile fails.

**Fix:** Use `@subsegmentStartsWithSAP = ('1', '2')`.

### 15. SCH-110 raises a type error when @codecs is on several levels

**Where:** [schematron.sch#L430](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L430)

**Problem:** `substring-before(ancestor-or-self::*/@codecs[1], '.')` gets one
@codecs per level when both the AdaptationSet and the Representation have it.
`[1]` applies to each step, not to the sequence.

**Effect:** XPTY0004. SaxonJS evaluates eagerly and stops; Saxon-J happened to skip it.

**Fix:** Use `(ancestor-or-self::*/@codecs)[last()]` for the nearest value.

### 16. A dynamic error in one rule aborts the whole Schematron report

**Where:** [app/Modules/Common/MPD/Schematron.php](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/Common/MPD/Schematron.php)

**Problem:** Saxon-HE 12 stopped with dynamic `tokenize()` cardinality errors on
3 of 110 MPDs (the DASH schema and DASH-IF example MPDs), so no Schematron result
was produced for them. Items 10 and 15 are likely causes.

**Effect:** All Schematron checks are lost for such MPDs.

**Fix:** Fix the rules, and report a failed Schematron run per rule if possible.

## Schematron rules (schematron/dvb-dash.sch)

### 17. Asserts 1 to 3 compare each Representation with itself

**Where:** [dvb-dash.sch#L15-L19](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/dvb-dash.sch#L15-L19)

**Problem:** `@mimeType = ancestor::dash:AdaptationSet/dash:Representation/@mimeType`
is a general comparison with all Representations, including the context node,
so it is true whenever the attribute is present.

**Effect:** Different @mimeType, @codecs or @audioSamplingRate values are never
reported. The rules fail instead when the attribute is only on the AdaptationSet.

**Fix:** Compare with `count(distinct-values(...)) = 1` on the effective values.

### 18. Asserts 9 to 13 never match Representations

**Where:** [dvb-dash.sch#L64](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/dvb-dash.sch#L64)

**Problem:** The rule context has `self::Representation` without the `dash:` prefix.

**Effect:** AC-4 constraints are never checked on Representations.

**Fix:** Use `self::dash:Representation`.

## Schematron rules (schematron/ac4-generic.sch)

### 19. Assert 2 has an inverted @mimeType test

**Where:** [ac4-generic.sch#L30](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/ac4-generic.sch#L30)

**Problem:** `$bs_ver != '00' or @mimeType != ('audio/mp4','video/mp4')` passes
only when @mimeType is not one of the two allowed values.

**Effect:** AC-4 bitstream version 0 with a correct @mimeType fails.

**Fix:** Use `@mimeType = ('audio/mp4','video/mp4')`.

## IOP module

### 20. The IOP live profile check tests misspelled profiles

**Where:** [app/Modules/IOP/MPD/Live.php#L48](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/IOP/MPD/Live.php#L48)
and [#L56](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/IOP/MPD/Live.php#L56)

**Problem:** The profiles are written `urn:mpeg:dash:profile:iofff-live:2011`
and `http://dashif.org/guidlines/dash-if-ondemand`.

**Effect:** The check never fires.

**Fix:** Use `isoff-live` and `guidelines`.

### 21. The AdaptationSet level check always fails for AVC

**Where:** [app/Modules/IOP/Segments/CrossAdaptation.php#L113-L130](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/IOP/Segments/CrossAdaptation.php#L113-L130)

**Problem:** For AVC, the level from @codecs is two hex digits (`1f`), but it is
compared with `"L" . $maxLevel` (`L31`). Without @codecs on the AdaptationSet,
the comparison is with an empty string.

**Effect:** Every AVC AdaptationSet fails, as does any AdaptationSet with @codecs only on Representations.

**Fix:** Convert the hex level, and skip AdaptationSets without @codecs.

## CMAF module

### 22. CompositionTimes requires an edit list without composition offsets

**Where:** [app/Modules/CMAF/Segments/CompositionTimes.php#L72](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/CompositionTimes.php#L72)

**Problem:** With `trun` version 0 an `elst` box is always required.

**Effect:** Video without B-frames (all composition offsets 0) fails.

**Fix:** Require the edit list only when there are composition offsets.

### 23. Durations ignores sample durations in trun

**Where:** [app/Modules/CMAF/Segments/Durations.php#L66](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/Durations.php#L66)

Already reported as [#772](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/issues/772).

### 24. The im1i subtitle check is not implemented and always fails

**Where:** [app/Modules/CMAF/Segments/Subtitles.php#L53](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/Subtitles.php#L53)

**Problem:** The check reports a failure with "not yet been ported".

**Effect:** Every im1i track fails.

**Fix:** Check for a `subs` box in each fragment, or skip the check until it is implemented.

### 25. VideoMediaProfile never checks the frame rate and fails every cud8 track

**Where:** [app/Modules/CMAF/Segments/VideoMediaProfile.php#L184](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/VideoMediaProfile.php#L184),
[#L249](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/VideoMediaProfile.php#L249),
[#L278](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/VideoMediaProfile.php#L278),
[#L315](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/VideoMediaProfile.php#L315),
[#L354](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/VideoMediaProfile.php#L354)

**Problem:** `signalledFrameRate` is always 0. The cud8 brand requires the HEVC
profile "Main8", but the signaled profile is mapped only to "Main", "Main10" or
"Other". Missing colour information is treated as value 1 (BT.709). The cud1
brand allows only BT.709 ([L201-L203](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/VideoMediaProfile.php#L201-L203)), but CMAF table 12 and
CTA-5001-F table 1 also allow BT.2020 (colour primaries 9, transfer
characteristics 14).

**Effect:** Frame rate limits are never checked, every cud8 track fails,
colour checks pass for streams that do not signal colour, and BT.2020 cud1
tracks such as the WAVE cud1 test vectors fail.

**Fix:** Take the frame rate from the VUI or the MPD, use "Main" for cud8,
skip colour checks when colour is not signaled (or treat it as unspecified, 2),
and accept BT.2020 for cud1.

### 26. SubtitleMediaProfile skips CMAF subtitles

**Where:** [app/Modules/CMAF/Segments/SubtitleMediaProfile.php#L46](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/SubtitleMediaProfile.php#L46)

**Problem:** Only AdaptationSets whose @mimeType starts with `text/` are checked,
but CMAF subtitle tracks have `application/mp4`.

**Effect:** The check does not run for CMAF subtitles.

**Fix:** Select AdaptationSets by @contentType text or by the track handler.

## Low Latency module

### 27. LeapSecondInformation is looked up with a misspelled name

**Where:** [app/Modules/LowLatency/MPD/LeapSecond.php#L48](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/MPD/LeapSecond.php#L48)

**Problem:** `getDOMElements('LeapSecondInformat')`.

**Effect:** Always reports that LeapSecondInformation is missing.

**Fix:** Use `LeapSecondInformation` (and see item 1, which also hides it).

### 28. The SegmentTimeline check can never pass

**Where:** [app/Modules/LowLatency/MPD/SegmentURL.php#L109](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/MPD/SegmentURL.php#L109)
and [#L117](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/MPD/SegmentURL.php#L117)

**Problem:** The fallback to the Period assigns `$segmentTemplates` instead of
`$segmentTimelines`, and @media is read from the SegmentTimeline element, which
has no such attribute (it is on SegmentTemplate). The test also requires both
`$Number$` and `$Time$`, which ISO/IEC 23009-1 forbids in one template.

**Effect:** Low-latency streams with SegmentTimeline always fail.

**Fix:** Read @media from the SegmentTemplate that holds the SegmentTimeline and require `$Number$` or `$Time$`.

### 29. Segment and chunked mode are chosen by presence, not value

**Where:** [app/Modules/LowLatency/Segments/SegmentOrChunked.php#L73](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/Segments/SegmentOrChunked.php#L73)

**Problem:** Any @availabilityTimeComplete selects segment mode, so
`availabilityTimeComplete="false"`, which marks a chunked AdaptationSet, selects
segment mode. Without the attribute, chunked mode is selected and then fails
because the attribute is missing.

**Effect:** Chunked low-latency streams are checked with the wrong rules.

**Fix:** Select chunked mode when @availabilityTimeComplete is false.

### 30. Booleans are compared with TRUE and FALSE

**Where:** [app/Modules/LowLatency/Segments/ChunkedAdaptation.php#L134](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/Segments/ChunkedAdaptation.php#L134)
and [ResyncCrossAdaptation.php#L189](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/Segments/ResyncCrossAdaptation.php#L189)

**Problem:** XML Schema booleans are `true`, `false`, `1` or `0`, but the values
are compared with `'FALSE'` and `'TRUE'`.

**Effect:** `availabilityTimeComplete="false"` and `Resync@marker="true"` always fail.

**Fix:** Compare case-sensitively with `false`/`0` and `true`/`1`.

### 31. SegmentAdaptation duration limits are inverted

**Where:** [app/Modules/LowLatency/Segments/SegmentAdaptation.php#L130](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/Segments/SegmentAdaptation.php#L130)
and [#L138](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/Segments/SegmentAdaptation.php#L138)

**Problem:** The checks pass when the segment duration is more than 50% (and 30%)
of the target latency, the opposite of the rule.

**Effect:** Short segments fail and long segments pass.

**Fix:** Use `<=` in both comparisons.

### 32. Placeholder checks that always pass or always fail

**Where:** [ChunkedAdaptation.php#L92-L104](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/Segments/ChunkedAdaptation.php#L92-L104)
and [ChunkedCrossAdaptation.php#L133-L136](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/Segments/ChunkedCrossAdaptation.php#L133-L136)

**Problem:** "Each Segment SHALL conform to a CMAF Fragment" and "EMSG box
constraints" are reported as passed without any check, and the second option of
the cross-AdaptationSet rule is a stub that returns false.

**Effect:** The report shows passes that were never checked, and the cross-AdaptationSet rule depends only on its first option.

**Fix:** Implement the checks or report them as skipped.

## DVB module

### 33. The common audio attribute checks always fail

**Where:** [app/Modules/DVB/MPD/AudioChecks.php#L219](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/AudioChecks.php#L219)
and [#L264](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/AudioChecks.php#L264)

**Problem:** The values are collected into `$values[$attribute]`, but the checks
read `$attributeValues`, which stays empty, so `count(array_unique(...)) == 1` is
`0 == 1`.

**Effect:** "@mimeType, @codecs and @audioSamplingRate SHALL/SHOULD be common"
(DVB 6.1.1) fails for every audio AdaptationSet.

**Fix:** Collect into `$attributeValues`.

### 34. The Dolby channel configuration check never runs for AC-4

**Where:** [app/Modules/DVB/MPD/AudioChecks.php#L145](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/AudioChecks.php#L145)

**Problem:** `str_starts_with('ec-3', $codec)` has its arguments reversed: it
tests whether `ec-3` starts with the codecs string. A full AC-4 codecs string
such as `ac-4.02.01.01` never matches, and the function returns.

**Effect:** AudioChannelConfiguration of AC-4 AdaptationSets is never checked (DVB 6.3.1).

**Fix:** Use `str_starts_with($codec, 'ec-3')` and `str_starts_with($codec, 'ac-4')`.

### 35. An optional @presentationTimeOffset is required

**Where:** [SCTEChecks.php#L47](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/SCTEChecks.php#L47) and [#L122](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/SCTEChecks.php#L122)

**Problem:** The check is described as "The @presentationTimeOffset attribute MAY
be present", but it fails when the attribute is absent.

**Effect:** SCTE-35 EventStreams without @presentationTimeOffset fail (DVB 9.1.9).

**Fix:** Remove the check, or report its absence as information.

### 36. An indefinite SCTE-35 event duration is never detected

**Where:** [SCTEChecks.php#L170](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/SCTEChecks.php#L170)

**Problem:** @duration is compared with the string `0xFFFFFFFF`, but the
attribute is an unsigned integer, written `4294967295`.

**Effect:** Events with indefinite duration pass (DVB 9.1.9).

**Fix:** Compare the numeric value with 4294967295.

### 37. Fractional frame rates are truncated in the timescale check

**Where:** [SCTEChecks.php#L97](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/SCTEChecks.php#L97)

**Problem:** `intval($adaptationSetFrameRate)` turns `30000/1001` into 30000.

**Effect:** The "@timescale is a multiple of the frame rate" check is wrong for fractional frame rates.

**Fix:** Evaluate the fraction.

### 38. AVC profile and level are required of every Representation

**Where:** [LegacyCodecs.php#L107](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/LegacyCodecs.php#L107) and [#L116](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/LegacyCodecs.php#L116)

**Problem:** Every AVC Representation must be exactly High profile (100) at
level 3.0, 3.2, 4.0 or 4.1. ETSI TS 103 285 V1.4.1 clause 5.1.1 instead
requires that an MPD with AVC video *includes* content compatible with the
avc_hd_50_level40 or avc_hd_60_level40 player conformance points, which
ETSI TS 101 154 table L.1 defines as Main and High profile up to level 4.0,
"unless targeting devices that are known to support other capabilities".
Other Representations are not restricted.

**Effect:** The BBC DVB-DASH test card
(`https://rdmedia.bbc.co.uk/testcard/vod/manifests/avc-full.mpd`) fails for its
Main and Constrained Baseline Representations, levels 2.1 and 3.1, and level 4.2
for 1080p50, although it also offers High profile level 4.0.

**Fix:** Per Period, check that at least one AVC Representation is Main or High
profile (or sets constraint_set1_flag, as Constrained Baseline does) at level
4.0 or lower, and report a warning otherwise because of the device exception.

### 39. The stpp subtitle checks skip @codecs with parameters

**Where:** [Subtitle.php#L60](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/Subtitle.php#L60)

**Problem:** The checks run only when @codecs equals `stpp`.

**Effect:** `stpp.ttml.im1t` and similar values are not checked.

**Fix:** Use `str_starts_with($codecs, 'stpp')`.

### 40. The channel count check fails with other channel configuration schemes

**Where:** [CrossAudio.php](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/CrossAudio.php)

**Problem:** The MPD channel count is read only from the
`urn:mpeg:dash:23003:3:audio_channel_configuration:2011` scheme and is empty otherwise.

**Effect:** Audio with another scheme (for example Dolby or MPEG cicp) fails.

**Fix:** Compare only when that scheme is used, or map the other schemes to channel counts.

### 41. The HEVC tier, profile, level and bit depth rules do not match the spec

**Where:** [LegacyCodecs.php#L124-L172](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/LegacyCodecs.php#L124-L172)

**Problem:** Every HEVC Representation must be Main tier, 8 or 10 bits, Main 10
profile (Main only up to 1920x1080) and level 4.1 up to 1080p or 5.1 above.
ETSI TS 103 285 V1.4.1 clause 5.2.3 only requires that the MPD *includes*
content for at least one HEVC player conformance point of ETSI TS 101 154
table L.1, and those allow Main profile up to level 5.1 at 2160p (hevc_uhd),
level 5.2 for high frame rates and level 6.1 at 4320p. The bit depth rule of
TS 101 154 clause L.3.3.2, which applies to every Representation, depends on the
profile: Main profile must be 8-bit, and Main 10 must have equal luma and
chroma bit depths of 8 or 10. The same clause requires progressive frames
(general_progressive_source_flag 1, general_interlaced_source_flag 0,
general_frame_only_constraint_flag 1), which is not checked.

**Effect:** Valid UHD Main profile, HFR and 8K content fails; Main profile
10-bit content and interlaced content pass.

**Fix:** Check the conformance points once per Period (warning, as for AVC),
and check bit depth by profile and the progressive flags per Representation.

### 42. SCH-7 requires a static MPD for the extended live profile

**Where:** [schematron.sch#L86](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L86)

**Problem:** The rule fails every dynamic MPD with
`urn:mpeg:dash:profile:isoff-ext-live:2014`. ISO/IEC 23009-1 clause 8.8 has
no such constraint (unlike the extended on-demand profile in clause 8.9.2.1,
which SCH-8 checks), and the standard's own extended live example MPD with
UTCTiming and MPD update events is dynamic.

**Effect:** False errors for live services using the extended live profile,
which is the profile meant for them.

**Fix:** Remove the assertion.

### 43. The DVB scheme and pssh checks do not match TS 103 285

**Where:** [ContentProtection.php#L62-L66](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/ContentProtection.php#L62-L66)
and [#L135-L143](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/ContentProtection.php#L135-L143)

**Problem:** The scheme check warns unless the 'schm' scheme is `cenc`. ETSI
TS 103 285 V1.3.1 and V1.4.1, clause 8.6.0, say that media data *shall* use `cenc` or `cbcs`, that AdaptationSets using `cbcs` shall not carry the
2014 profile, and that presentations with AVC shall keep an AVC video
AdaptationSet and one AdaptationSet of each other media type that is clear or
`cenc`. None of these is checked. The check "PSSH Boxes SHOULD exist in either
the MPD or the Initialization Segment" has no counterpart in the spec: clause
8.5.0 says the initialization segment and/or media segments may carry 'pssh'.

**Effect:** Valid `cbcs` content gets warnings, while `cbcs` with the 2014
profile, other schemes such as `cens`, and missing `cenc` fallbacks are only
warnings or not reported. Content that delivers 'pssh' in media segments gets
a warning.

**Fix:** Fail schemes other than `cenc` and `cbcs`, fail `cbcs` with the 2014
profile, add the fallback check per Period, and report the pssh case as
information.

### 44. Several DVB MPD update checks can never fail, others fail wrongly

**Where:** [UpdateConstraints.php](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/UpdateConstraints.php)

**Problem:**

- [L243](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/UpdateConstraints.php#L243) reads `AssetIdentifier` as an attribute, but it is an
  element, so both sides are empty and the check always passes (DVB-48).
- [L296](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/UpdateConstraints.php#L296) and [L374](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/UpdateConstraints.php#L374) look for `AudioConfiguration`
  elements; the element is `AudioChannelConfiguration`, so the check always
  passes (DVB-52).
- [L152-L153](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/UpdateConstraints.php#L152-L153) compares `maxSegmentDuration` values as strings,
  so an increase from `PT2S` to `PT10S` is not found (DVB-45).
  [L236](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/UpdateConstraints.php#L236) compares Period@start as strings too.
- AdaptationSets and Representations are paired by @id only, so a changed @id
  is never reported (DVB-49, DVB-50 always pass), while AdaptationSets without
  the optional @id fail with "Unable to retrieve @id"
  ([L259](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/UpdateConstraints.php#L259)).
- `SegmentTemplate@timescale` (TS 103 285 4.8.3 item 8) and the removal of
  AdaptationSets and Representations (4.8.1 item 1) are not checked.

**Effect:** Changes to AssetIdentifier, AudioChannelConfiguration,
maxSegmentDuration, ids and timescale pass, and MPDs without AdaptationSet@id
fail.

**Fix:** Compare the AssetIdentifier and AudioChannelConfiguration elements,
parse durations, pair Representations by @id within a Period and
AdaptationSets through their Representations, and add the missing checks.

### 45. The DVB HEVC signalling checks do not follow TS 103 285 5.2

**Where:** [BitStream.php](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/BitStream.php)

**Problem:**

- The ColourPrimaries, MatrixCoefficients and TransferCharacteristics
  descriptors are compared with the 'colr' box
  ([L238](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/BitStream.php#L238)); TS 103 285 V1.4.1 clause 5.2.5 says they
  shall match the VUI. Without a 'colr' box the check fails even when the VUI
  matches.
- The HLG10 and PQ10 checks (5.2.6, 5.2.7) report information when the
  signalling is missing and pass when it is present
  ([L165](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/BitStream.php#L165), [L227](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/BitStream.php#L227)), so they
  never fail. They also do not look at the bitstream to see whether HLG10 or
  PQ10 is used.
- The @codecs check requires the string derived from the decoder
  configuration to be a prefix of @codecs ([L333](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/BitStream.php#L333)),
  but clause 5.2.2 says the values
  "are not required to indicate the minimum capabilities", so a higher level
  is allowed.
- Only `hev1` and `hvc1` sample entries are checked
  ([L86](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/BitStream.php#L86)), so encrypted HEVC (`encv`) is skipped.

**Effect:** False failures for streams without 'colr' and for @codecs with a
higher level; missing HDR signalling is never reported.

**Fix:** Compare with the VUI colour description, detect PQ10 from
transfer_characteristics 16 and HLG10 from 14 plus the
alternative_transfer_characteristics SEI, fail missing signalling, and accept
@codecs values that signal at least the media's profile, tier and level.

### 46. The DVB SEI location check skips hvc1 and misses prefix SEI after VCL

**Where:** [SEILocations.php#L52](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/SEILocations.php#L52) and
[#L91](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/SEILocations.php#L91)

**Problem:** The HEVC check runs for the sample entries `hev1` and `hevc`;
`hevc` does not exist and `hvc1` is missing. ETSI TS 101 154 L.3.3.8 requires
prefix SEI before the first VCL NAL unit, which is not checked. A suffix SEI
fails if any NAL unit except SEI follows it, also non-VCL units such as end of
sequence or filler data, while the rule is only about VCL NAL units.

**Effect:** hvc1 streams are not checked, prefix SEI between VCL NAL units
pass, and valid suffix SEI before end of sequence fail.

The VVC check ([L116](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/SEILocations.php#L116)) allows only NAL
unit types 16 and 17 (PPS and prefix APS) before prefix SEI, while TS 101 154
5.15.1.8.1 names DCI, VPS, SPS and PPS (13 to 16), and it does not check that
prefix SEI precede the picture header.

**Fix:** Check `hvc1`, fail a prefix SEI after a VCL NAL unit (or, for VVC,
after the picture header), compare suffix SEI positions only with VCL NAL
units, and use the VVC parameter set types 13 to 16.

### 47. The CTA WAVE checks follow the 2018 edition and misread it

**Where:** [AudioProfile.php#L98](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CTAWave/Segments/AudioProfile.php#L98),
[VideoProfile.php#L99](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CTAWave/Segments/VideoProfile.php#L99),
[SubtitleProfile.php#L101](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CTAWave/Segments/SubtitleProfile.php#L101),
[SplicingPoints.php#L126](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CTAWave/Segments/SplicingPoints.php#L126)

**Problem:** The module reports against "WAVE Content Spec 2018Ed".
In CTA-5001-F (2025 edition):

- The AAC Core, AVC HD and IMSC1 Text profiles are required only for
  presentations that conform to the CMFHD presentation profiles (clause 5),
  per selection set. Upstream requires them in every audio, video and
  subtitle AdaptationSet of all WAVE content.
- WAVE programs "MAY conform" to a splice constraint profile (clause 6.1),
  and none is signalled, but upstream fails every splice that breaks the
  Baseline Splice Constraint Profile.
- Within that profile, frame rate, picture aspect ratio and channel
  configuration only have to conform to the media profile (table 5); keeping
  them the same is a recommendation (table 6). Upstream fails any change.
- Only the first Representation of AdaptationSets at the same index in the
  two Periods is compared.

**Effect:** WAVE content with additional AdaptationSets (for example a second
audio language in E-AC-3, or HEVC next to AVC) fails, and multi-Period
content fails for allowed changes.

**Fix:** Check the required profiles only for CMFHD presentation profiles and
per selection set, and report splice constraints as information, with the
table 6 items as recommendations.

### 48. Several DASH-HLS interoperability checks fail valid content

**Where:** [EncryptionScheme.php](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/WaveHLSInterop/Segments/EncryptionScheme.php),
[SegmentEncryption.php](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/WaveHLSInterop/Segments/SegmentEncryption.php),
[AddressableMediaObject.php](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/WaveHLSInterop/Segments/AddressableMediaObject.php)

**Problem:** Compared with CTA-5005-B:

- The encrypt:skip pattern check ([L90](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/WaveHLSInterop/Segments/EncryptionScheme.php#L90)) also
  runs for `cenc` tracks, which have no pattern, so every cenc video and audio
  track fails. For cbcs it accepts any N:9N video and N:0 audio pattern,
  not 1:9 and 10:0.
- `cenc` tracks must have a constant 8-byte IV
  ([L206](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/WaveHLSInterop/Segments/EncryptionScheme.php#L206)). CMAF 8.2.3.1, which clause 4.3.2
  refers to, limits cenc IVs to 8 bytes; cenc normally uses per-sample IVs,
  which this rejects.
- "A single encryption key per segment" is tested by counting the KIDs in all
  pssh boxes of the segment ([L71](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/WaveHLSInterop/Segments/SegmentEncryption.php#L71)), so one
  key listed for two DRM systems fails. "A single IV" is tested by requiring
  no per-sample IVs in senc ([L94](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/WaveHLSInterop/Segments/SegmentEncryption.php#L94)), which
  fails every cenc track.
- The single 'sidx' of clause 4.1.2 applies to CMAF track files delivered to
  clients (byte-range addressing), but it is required in every media segment,
  so SegmentTemplate content fails.
- The check that a cenc component has a cbcs equivalent (4.3.3) is a TODO.

**Effect:** All cenc content and all SegmentTemplate content fail.

**Fix:** Check patterns only for cbcs (1:9 video, 10:0 audio), require 8-byte
cenc IVs, compare the KID and IV of each fragment in a segment (from 'seig' or
'tenc'), and check 'sidx' only for SegmentBase track files.

### 49. The HbbTV codec and AVC rules are outdated

**Where:** [MPD.php](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/HbbTV/MPD.php), [Segments/LegacyCodecs.php#L110](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/HbbTV/Segments/LegacyCodecs.php#L110) and
[#L119](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/HbbTV/Segments/LegacyCodecs.php#L119)

**Problem:** Since ETSI TS 102 796 V1.4.1, annex E defines the HbbTV DASH
profile through DVB-DASH: E.4.1 treats `urn:hbbtv:dash:profile:isoff-live:2012`
as the DVB-DASH 2014 live profile, and E.2.1 only requires the MPD to include
at least one AdaptationSet in a codec of clause 7.3.1, which includes HEVC.
Upstream warns for every Representation whose codec is not AVC, AAC or
E-AC-3 (so also for HEVC), and requires every AVC Representation to be Main
or High profile at level 3.0, 3.1, 3.2 or 4.0, instead of the DVB-DASH
conformance points (bug 38).

**Effect:** HEVC and additional Representations in other codecs are
reported, and AVC ladders with lower levels fail.

**Fix:** Require one AdaptationSet in a 7.3.1 codec and apply the DVB-DASH
rules to HbbTV content.

### 50. The DVB 2018 schema check rejects UTCTiming

**Where:** [resources/xsd/DASH2nd.xsd](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/resources/xsd/DASH2nd.xsd),
used by [app/Modules/Common/MPD/XSDValidation.php#L102](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/Common/MPD/XSDValidation.php#L102)

**Problem:** COMMON-3 validates against the 2nd edition schema without
Amendment 1, which has no UTCTiming element. DVB-DASH requires UTCTiming in
live MPDs (4.7.2), and UTCTiming is defined in ISO/IEC 23009-1:2014/Amd 1.

**Effect:** Every DVB MPD with UTCTiming fails the "DASH 2nd (DVB 2018)" check.

**Fix:** Use the schema of the 2nd edition with Amd 1 (branch `2nd-Ed+AMD1` of
MPEGGroup/DASHSchema).

### 51. The DVB codec lists lack the codecs added since V1.3.1

**Where:** [MPD/Codecs.php#L45](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/MPD/Codecs.php#L45) and
[Segments/LegacyCodecs.php#L177](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/DVB/Segments/LegacyCodecs.php#L177)

**Problem:** Both lists hold AVC, HEVC, AAC, E-AC-3, AC-4, DTS and stpp.
ETSI TS 103 285 V1.4.1 also specifies VVC (`vvc1`, `vvi1`, clause 5.3), AVS3
(`avs3`, 5.4), DTS-UHD (`dtsx`, `dtsy`, 6.4) and MPEG-H Audio (`mhm1`,
`mhm2`, 6.8).

**Effect:** DVB content with these codecs gets "codec not supported" warnings.

**Fix:** Add the codecs of TS 103 285 V1.4.1 clauses 5 to 7.

### 52. Durations compares the first tfdt of a segment with every fragment

**Where:** [app/Modules/CMAF/Segments/Durations.php#L61](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/Durations.php#L61)

**Problem:** The loop over the fragments of a segment compares `$tfdtBoxes[0]`,
the first fragment's baseMediaDecodeTime, with the running offset, instead of
`$tfdtBoxes[$index]`. ISO/IEC 23000-19 clauses 7.3.2.2 c) and 7.3.2.3 g) require
each CMAF fragment and chunk to start at the previous one's baseMediaDecodeTime
plus its sample durations.

**Effect:** Every valid segment with more than one fragment or chunk, as in
low-latency streams, fails, and wrong baseMediaDecodeTime values in later
fragments are never found.

**Fix:** Compare `$tfdtBoxes[$index]->decodeTime`, with the durations of bug 23
fixed.

### 53. @maxWidth and @maxHeight are compared with every track

**Where:** [DASHProfile.php#L153-L166](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/Segments/DASHProfile.php#L153-L166)

**Problem:** Each Representation's track width and height are compared with the
AdaptationSet @maxWidth and @maxHeight. ISO/IEC 23009-1 8.12.4.3 requires
"@maxWidth or @width" to be the width in the TrackHeaderBox of the CMAF
Principal Header, which is one header for the whole Switching Set. Without
@maxWidth, the track width is compared with an empty string. The checks are
warnings, but the clause says "shall".

**Effect:** In an AdaptationSet with several resolutions, every Representation
but the largest fails, and an AdaptationSet with @width but no @maxWidth always
fails (LL-21, LL-22).

**Fix:** Compare @maxWidth and @maxHeight once per AdaptationSet with the tkhd
size of the CMAF Principal Header (@initializationPrincipal, or else the
largest track), and without them compare @width and @height. Report failures as
errors.

### 54. The fragment bitrate check applies the on-demand rule to any segments

**Where:** [Bitrate.php#L46-L106](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/WaveHLSInterop/Segments/Bitrate.php#L46-L106)

**Problem:** CTA-5005-B 4.1.2 has two rules. On demand, each CMAF fragment
should be within 10% of the average bitrate of the whole track. Live, each
fragment of a 1-hour slice, or of the whole track if shorter, should be at
most 110% of the slice average. The check never looks at MPD@type, although
its skip reason says "MPD is not signalled as on-demand". It applies the
on-demand bounds to whichever segments were downloaded, and includes the last
fragment, which may be short. A SHOULD is reported as FAIL
([L102](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/WaveHLSInterop/Segments/Bitrate.php#L102)), and `$atLeastOneFailed |=` collects the results of
passing checks, so the calculated bitrates are logged when a segment passes.

**Effect:** Live streams fail when one fragment is below 90% of the
average. On demand, the average comes from the downloaded segments rather than
the track, and a short last fragment fails (WAVEHLS-2).

**Fix:** Use the on-demand rule only for static MPDs and over all fragments of
the track, excluding a short last fragment. For dynamic MPDs, apply only the
110% upper bound over an hour of the track. Report a warning, and collect
failures with `!pathAdd(...)`.

### 55. LL-24 compares KIDs in different formats and reads cenc:pssh as an attribute

**Where:** [DASHProfile.php#L181-L240](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/LowLatency/Segments/DASHProfile.php#L181-L240),
with the KID from [MP4BoxRepresentation.php#L318](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Services/Validators/MP4BoxRepresentation.php#L318)

**Problem:** The tenc KID comes from the MP4Box dump as `0x` and 32 upper-case
hex digits (`KID="0x9EB4050D..."`, checked with GPAC 26.07), and is compared as
a string with cenc:default_KID, which ISO/IEC 23001-7 11.2 writes as a UUID
(`9eb4050d-e44b-...`), so the two never match. `getAttribute('cenc:pssh')`
reads an element as an attribute and is always empty; read as an element, the
base64 pssh box would be compared with the SystemID of the first pssh box in the
initialization segment. The check passes when any one descriptor matches, so a
DRM-specific descriptor without cenc:default_KID passes it without the
mp4protection descriptor that ISO/IEC 23001-7 11.3.2 requires, and the
mp4protection @value is not compared with the schm scheme type (ISO/IEC
23009-1 5.8.5.2.2).

**Effect:** Encrypted low-latency content fails LL-24 when every
ContentProtection descriptor has cenc:default_KID, even the right one, and
passes when any descriptor has none, whatever its scheme and the other KIDs.

**Fix:** Compare KIDs as 16 bytes, for example as hex digits in one letter
case. Require the mp4protection descriptor for the ISO/IEC 23001-7 schemes,
compare its @value with the schm scheme type, and check that the
cenc:default_KID of every descriptor lists the tenc KID. Read cenc:pssh as a
child element in the urn:mpeg:cenc:2013 namespace and compare the SystemID of
the decoded box with the UUID of the descriptor.

### 56. ContentProtection @value is checked only by its length

**Where:** [schematron.sch#L243-L247](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/schematron/schematron.sch#L243-L247)

**Problem:** SCH-55 requires `string-length(@value) = 4` for the mp4protection
descriptor, but ISO/IEC 23009-1 5.8.5.2.2 lets the 4CC be percent-encoded and
followed by ":" and a version of up to 8 hex digits (`cenc:00010000`). A
descriptor with @ref (5.8.4.1.3) inherits @value from its source and need not
have its own. SCH-56 also checks only the length, while 5.8.5.2.3 requires 4
lower-case hex digits for the CA_system_ID.

**Effect:** Valid versioned or percent-encoded values and referencing
descriptors fail SCH-55, and CA_descriptor values such as `0B00` or `xyz!`
pass SCH-56.

**Fix:** Match the ABNF of 5.8.5.2.2 (allowing digits in the 4CC, as in
`cbc1`) on the @value after resolving @ref, and `[0-9a-f]{4}` for the
CA_descriptor.

### 57. CMAF-15 counts senc boxes against moof boxes

**Where:** [LegacyInitialization.php#L50-L81](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance/blob/e311ec6e12b21394355fd1749101590400e31cbb/app/Modules/CMAF/Segments/LegacyInitialization.php#L50-L81)

**Problem:** ISO/IEC 23000-19 7.3.2.4 b requires each CMAF fragment, with its
CMAF header and keys, to carry enough metadata to be decrypted on its own. The
check instead requires as many senc boxes as moof boxes, whenever the MPD has a
ContentProtection element, and only with the dash264 profile. But 8.2.2.1 makes
senc a SHOULD: the sample auxiliary information may be in another box, such as
a PIFF uuid box, and with constant IVs and no subsample encryption there is
none, so senc "should be omitted". The clear fragments of an encrypted track
(a clear lead), which use a sample entry without sinf or a seig sample group
with isProtected 0, need none either.

**Effect:** cbcs audio with constant IVs and clear-lead segments fail, and
DASH-IF IOP content without the dash264 profile is not checked (CMAF-15).

**Fix:** For each fragment, find whether its samples are encrypted from the
sample entry that its sample description index selects and from its seig
sample group, and require sample auxiliary information (senc, a uuid box or
saio) when decrypting needs it: for subsample-encrypted video, or per-sample
IVs.

## Other projects

Found during the same work in projects that dashcheck builds on or tests with.

| Project | Problem |
|---|---|
| [mp4ff](https://github.com/Eyevinn/mp4ff) | `IsSyncSampleFlags` requires `sample_depends_on` = 2, so sync samples with `sample_depends_on` = 0 (unknown) and `sample_is_non_sync_sample` = 0 are not sync. ISO/IEC 14496-12 lets the non-sync flag alone decide. |
| [dash-mpd](https://github.com/Eyevinn/dash-mpd) | Its XML parser accepts malformed XML (a missing space between attributes). |
| dash-mpd | xs:duration values with years or months are rejected, so such MPDs cannot be parsed although they are schema-valid. |
| dash-mpd | An absent `@indexRangeExact` cannot be told from `false`. |
| [jacoelho/xsd](https://github.com/jacoelho/xsd) | Up to v0.6.4, built-in `xlink:type`, `href`, `role`, `arcrole`, `title`, `show` and `actuate` attributes, typed `xs:string` or `xs:anyURI`, shadow the declarations of an imported xlink.xsd (`internal/schema/builtin.go`; `AddGlobalAttributeComponent` skips redeclarations of built-ins). So `xlink:actuate="bogus"` validates, `xlink:role=""` passes its minLength, and `xlink:type=" simple "` fails `fixed="simple"` because it is not whitespace-collapsed. Fix: treat the XLink built-ins as fallbacks that a loaded schema declaration replaces. Reported as [#137](https://github.com/jacoelho/xsd/issues/137). |
| [livesim2](https://github.com/Dash-Industry-Forum/livesim2) | In SegmentTimeline mode, audio segments of up to 2.005333 s exceed `maxSegmentDuration="PT2S"`. |
| livesim2 | The test asset `cmd/livesim2/app/testdata/assets/testpic_2s/Manifest.mpd` has `DASH-MPD.xsd"bprofiles=` (malformed XML) since commit d6fd0a2. |
| livesim2 | The low-latency `testpic4_8s` video has trun version 1 and an offset edit list (media_time 1024). ISO/IEC 23000-19 9.2.5 forbids both together, and DASH-IF low-latency 9.X.4.5 requires CMAF fragments. |
| livesim2 | `Manifest300.mpd` keeps `maxWidth="1280" maxHeight="720"` from the full ladder, but its only video Representation is 640x360. ISO/IEC 23009-1 5.3.3.2 defines @maxWidth as the largest @width of the AdaptationSet. |
| livesim2 | With Segment Sequences (`ssras_2,1/chunkdurssr_2,0.5`), a Partial Segment of the sequence in progress gets 425 until the whole sequence has ended, although ISO/IEC 23009-1 5.3.9.7.1 derives its availability start time from its own MPD duration (the sequence duration divided by @k). |
| livesim2 | With Segment Sequences, the SSR AdaptationSet always has `<SegmentSequenceProperties sapType="1" cadence="1">`, whatever the Partial Segment duration. `testpic_2s_low_delay` has an IDR picture every second, so with `chunkdurssr_2,0.5` Partial Segments 1 and 3 of each sequence start with a non-IDR picture, contrary to the signalled SAP type (ISO/IEC 23009-1 5.3.5.7). |
| livesim2 | With `continuous_1`, the period-continuity descriptors have `value="1"` instead of the @id of the earlier Period, and the AdaptationSets have no @id (ISO/IEC 23009-1 5.3.2.4 needs both). The first Period also carries the descriptor. |
| CTA WAVE test content | The cenc vectors (for example `clg1_sets/12.5_25_50/t41-cenc/2025-01-15`) use 16-byte IVs; ISO/IEC 23000-19 8.2.3.1 (DIS text) limits cenc IVs to 8 bytes. |
| Shaka demo assets | `angel-one-widevine` has the CMAF brand cmfc, but its initialization segments have a clear sample entry (avc1, mp4a) after the protected one, and the clear-lead fragments select it with their sample description index. ISO/IEC 23000-19 7.5.10 requires every sample entry of an encrypted CMAF track to be protected, and 8.2.3.2 signals clear samples with sample groups. |
| BBC test card | The live test card's subtitle initialization segments (`testcard2020/*-subs/IS.mp4`) have an 8-byte 'sthd' box without the FullBox version and flags, so mp4ff cannot parse them. |
