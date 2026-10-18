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
| 25 | CMAF VideoMediaProfile | Frame rate never checked, cud8 always fails | Verified in code |
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
"Other". Missing colour information is treated as value 1 (BT.709).

**Effect:** Frame rate limits are never checked, every cud8 track fails, and
colour checks pass for streams that do not signal colour.

**Fix:** Take the frame rate from the VUI or the MPD, use "Main" for cud8, and
skip colour checks when colour is not signaled (or treat it as unspecified, 2).

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

## Other projects

Found during the same work in projects that dashcheck builds on.

| Project | Problem |
|---|---|
| [mp4ff](https://github.com/Eyevinn/mp4ff) | `IsSyncSampleFlags` requires `sample_depends_on` = 2, so sync samples with `sample_depends_on` = 0 (unknown) and `sample_is_non_sync_sample` = 0 are not sync. ISO/IEC 14496-12 lets the non-sync flag alone decide. |
| [dash-mpd](https://github.com/Eyevinn/dash-mpd) | Its XML parser accepts malformed XML (a missing space between attributes). |
| dash-mpd | xs:duration values with years or months are rejected, so such MPDs cannot be parsed although they are schema-valid. |
| dash-mpd | An absent `@indexRangeExact` cannot be told from `false`. |
| [livesim2](https://github.com/Dash-Industry-Forum/livesim2) | In SegmentTimeline mode, audio segments of up to 2.005333 s exceed `maxSegmentDuration="PT2S"`. |
| livesim2 | The test asset `cmd/livesim2/app/testdata/assets/testpic_2s/Manifest.mpd` has `DASH-MPD.xsd"bprofiles=` (malformed XML) since commit d6fd0a2. |
