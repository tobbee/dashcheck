# DASH-IF Conformance tool coverage

<!-- Generated from dashif-checks.json by `go test ./internal/coverage -update`. Do not edit. -->

Checks of the [DASH-IF Conformance tool](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance) at commit e311ec6 (2026-10-16), and the dashcheck rules that cover them. Statuses: **done** is covered by a dashcheck rule, **partial** is partly covered, **todo** is not ported yet, and **n/a** will not be ported (see the note).

| Group | Checks | Done | Partial | Todo | n/a |
|---|---:|---:|---:|---:|---:|
| Schematron: schematron.sch | 123 | 121 | 0 | 0 | 2 |
| Schematron: dvb-dash.sch | 13 | 13 | 0 | 0 | 0 |
| Schematron: ac4-generic.sch | 17 | 17 | 0 | 0 | 0 |
| Module: Common | 5 | 4 | 0 | 0 | 1 |
| Module: IOP | 15 | 14 | 0 | 0 | 1 |
| Module: CMAF | 23 | 23 | 0 | 0 | 0 |
| Module: Low Latency | 54 | 53 | 0 | 0 | 1 |
| Module: DVB | 103 | 95 | 1 | 5 | 2 |
| Module: HbbTV | 10 | 5 | 0 | 0 | 5 |
| Module: CTA WAVE | 21 | 16 | 0 | 0 | 5 |
| Module: WAVE HLS interop | 19 | 19 | 0 | 0 | 0 |
| Module: Dolby | 5 | 5 | 0 | 0 | 0 |
| **Total** | **408** | **385** | **1** | **5** | **17** |

## Schematron: schematron.sch

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| SCH-1 | `dash:MPD`: If MPD is of type "dynamic" availabilityStartTime shall be defined. | done | MPD-DYN-AST |  |
| SCH-2 | `dash:MPD`: If MPD is of type "dynamic" publishTime shall be defined. | done | MPD-DYN-PUBLISHTIME |  |
| SCH-3 | `dash:MPD`: If mediaPresentationDuration is not defined for the MPD minimumUpdatePeriod shall be defined or vice versa. | done | MPD-DURATION-MISSING | upstream ignores the last Period@duration; dashcheck uses the condition of SCH-6 |
| SCH-4 | `dash:MPD`: At least one unknown profile string found. Known list in schematron file. | done | MPD-PROFILE-UNKNOWN | dashcheck reports info, upstream an error; dashcheck also knows DVB interop, CMAF, low-latency and newer MPEG profiles |
| SCH-5 | `dash:MPD`: For On-Demand profile, the MPD @type shall be "static". | done | MPD-PROFILE-TYPE |  |
| SCH-6 | `dash:MPD`: If minimumUpdatePeriod is not present and the last period does not include the duration attribute the mediaPresentationDuration must be present. | done | MPD-DURATION-MISSING |  |
| SCH-7 | `dash:MPD`: For ISO BMFF extended live profile, MPD@type shall be static. | n/a |  | not a requirement: ISO/IEC 23009-1 8.8 does not restrict the extended live profile to static MPDs, and its own extended live example is dynamic (bug 42) |
| SCH-8 | `dash:MPD`: For ISO BMFF extended on-demand profile, MPD@type shall be static. | done | MPD-PROFILE-TYPE |  |
| SCH-9 | `dash:Period`: If bitstreamSwitching is set to true all bitstreamSwitching declarations for AdaptationSet within this Period shall not be set to false. | done | MPD-BITSTREAMSWITCHING |  |
| SCH-10 | `dash:Period`: The id of each Period shall be unique. | done | MPD-PERIOD-ID |  |
| SCH-11 | `dash:Period`: At most one of SegmentBase, SegmentTemplate and SegmentList shall be defined in Period. | done | MPD-SEGINFO-MULTIPLE |  |
| SCH-12 | `dash:Period`: If the MPD is dynamic the Period element shall have an id. | done | MPD-PERIOD-ID |  |
| SCH-13 | `dash:Period`: At least one BaseURL, SegmentTemplate or SegmentList shall be defined in Period, AdaptationSet or Representation. | done | MPD-SEGINFO-NONE |  |
| SCH-14 | `dash:Period`: If the duration attribute is set to zero, there should only be a single AdaptationSet present. | done | MPD-PERIOD-ZERO | upstream fails for any Period with @duration 0 and an AdaptationSet; dashcheck follows the message and warns when there is more than one |
| SCH-15 | `dash:Period`: Neither the Period.SegmentList element nor the Period.SegmentTemplate element shall be present for On-Demand profile, violated here. | done | MPD-PROFILE-SEGINFO |  |
| SCH-16 | `dash:Period`: For ISO BMFF extended live profile, Period.SegmentList shall not be present. | done | MPD-PROFILE-SEGINFO | Period@profiles is not in the 6th edition schema, so only MPD@profiles is used |
| SCH-17 | `dash:Period`: For ISO BMFF extended live profile: "If a Period contains multiple Adaptation Sets with value of the @contentType="video" then at least one Adaptation Set shall contain a Role el-ement $&lt;$Role scheme="urn:mpeg:dash:role:2011" value="main"&gt;" | done | MPD-PROFILE-MAINROLE |  |
| SCH-18 | `dash:Period`: For ISO BMFF extended on-demand profile, neither the Period.SegmentList element nor the Period.SegmentTemplate element shall be present for On-Demand profile, violated here. | done | MPD-PROFILE-SEGINFO |  |
| SCH-19 | `dash:Period`: For ISO BMFF extended on-demand profile: "If a Period contains multiple Adaptation Sets with value of the @contentType="video" then at least one Adaptation Set shall contain a Role el-ement $&lt;$Role scheme="urn:mpeg:dash:role:2011" value="main"&gt;" | done | MPD-PROFILE-MAINROLE |  |
| SCH-20 | `dash:AdaptationSet`: The id of each AdaptationSet within a Period shall be unique. | done | MPD-AS-ID |  |
| SCH-21 | `dash:AdaptationSet`: Attributes from the AdaptationSet shall not be repeated in the descendanding ContentComponent elements. | done | MPD-CONTENTCOMPONENT |  |
| SCH-22 | `dash:AdaptationSet`: Common attributes for AdaptationSet and Representation shall either be in one of the elements but not in both. | done | MPD-COMMON-BOTH | dashcheck reports information, since ISO/IEC 23009-1 5.3.7.1 allows common attributes on both levels (upstream bug 58) |
| SCH-23 | `dash:AdaptationSet`: Each minimum value (minWidth, minHeight, minBandwidth) shall be larger than the maximum value. | done | MPD-AS-MINMAX |  |
| SCH-24 | `dash:AdaptationSet`: The value of the bandwidth attribute shall be in the range defined by the AdaptationSet. | done | MPD-AS-RANGE |  |
| SCH-25 | `dash:AdaptationSet`: The value of the width attribute shall be in the range defined by the AdaptationSet. | done | MPD-AS-RANGE | dashcheck also checks @minWidth |
| SCH-26 | `dash:AdaptationSet`: The value of the height attribute shall be in the range defined by the AdaptationSet. | done | MPD-AS-RANGE | dashcheck also checks @minHeight |
| SCH-27 | `dash:AdaptationSet`: An AdaptationSet shall have at least one Representation element. | done | MPD-AS-NOREP |  |
| SCH-28 | `dash:AdaptationSet`: At most one of SegmentBase, SegmentTemplate and SegmentList shall be defined in AdaptationSet. | done | MPD-SEGINFO-MULTIPLE |  |
| SCH-29 | `dash:AdaptationSet`: ISO/IEC 23009-1:2019, 5.3.3.1: The value of the frameRate attribute shall be in the range defined by the AdaptationSet. | done | MPD-AS-RANGE |  |
| SCH-30 | `dash:AdaptationSet`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - The MPD contains an attribute that is not part of the HbbTV profile', i.e., found 'subsegmentAlignment' as true | done | MPD-HBBTV |  |
| SCH-31 | `dash:AdaptationSet`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - The MPD contains an attribute that is not part of the HbbTV profile', i.e., found 'subsegmentStartsWithSAP' as 1 or 2 | done | MPD-HBBTV |  |
| SCH-32 | `dash:AdaptationSet`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - The MPD contains an attribute that is not part of the HbbTV profile', i.e., found 'subsegmentStartsWithSAP' as 3 while not containing more than one Representation | done | MPD-HBBTV |  |
| SCH-33 | `dash:ContentComponent`: The id of each ContentComponent within an AdaptationSet shall be unique. | done | MPD-CONTENTCOMPONENT |  |
| SCH-34 | `dash:Representation`: Either the Representation or the containing AdaptationSet shall have the mimeType attribute. | done | MPD-MIMETYPE |  |
| SCH-35 | `dash:Representation`: For live profile, the SegmentTemplate element shall be present on at least one of the three levels, the Period level containing the Representation, the Adaptation Set containing the Representation, or on Representation level itself. | done | MPD-PROFILE-SEGINFO | uses the effective @profiles of the Representation |
| SCH-36 | `dash:Representation`: At most one of SegmentBase, SegmentTemplate and SegmentList shall be defined in Representation. | done | MPD-SEGINFO-MULTIPLE |  |
| SCH-37 | `dash:Representation`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - The MPD contains an element that is not part of the HbbTV profile', i.e., found 'BaseURL' element | done | MPD-HBBTV |  |
| SCH-38 | `dash:Representation`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - The MPD contains an attribute that is not part of the HbbTV profile', i.e., found 'subsegmentStartsWithSAP' as 3 with same value of mediaStreamStructureId in more than one Representation | done | MPD-HBBTV |  |
| SCH-39 | `dash:Representation`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - For HbbTV profile, the SegmentTemplate element shall be present on at least one of the three levels, the Period level containing the Representation, the Adaptation Set containing the Representation, or on Representation level itself as it is based on live profile | done | MPD-HBBTV |  |
| SCH-40 | `dash:SubRepresentation`: If the level attribute is defined for a SubRepresentation also the bandwidth attribute shall be defined. | done | MPD-SUBREP-BANDWIDTH |  |
| SCH-41 | `dash:SegmentTemplate`: If more than one Media Segment is present the duration attribute or SegmentTimeline element shall be present. | done | ADDR-TEMPLATE | checked on the inherited SegmentTemplate of each Representation, so templates split over levels are not flagged as upstream does |
| SCH-42 | `dash:SegmentTemplate`: Either the duration attribute or SegmentTimeline element shall be present but not both. | done | ADDR-TEMPLATE |  |
| SCH-43 | `dash:SegmentTemplate`: If indexRange is not present indexRangeExact shall not be present. | done | MPD-INDEXRANGEEXACT | checked on the attributes as written |
| SCH-44 | `dash:SegmentTemplate`: Neither $Number$ nor the $Time$ identifier shall be included in the initialization attribute. | done | MPD-TEMPLATE-INIT |  |
| SCH-45 | `dash:SegmentTemplate`: Neither $Number$ nor the $Time$ identifier shall be included in the bitstreamSwitching attribute. | done | MPD-TEMPLATE-INIT |  |
| SCH-46 | `dash:SegmentTemplate`: Only identifiers such as $Bandwidth$, $Time$, $RepresentationID$, or $Number$ shall be used. | done | ADDR-TEMPLATE | checked when @media is expanded |
| SCH-47 | `dash:SegmentTemplate`: $RepresentationID$ shall not have a format tag. | done | ADDR-TEMPLATE |  |
| SCH-48 | `dash:SegmentList`: If more than one Media Segment is present the duration attribute or SegmentTimeline element shall be present. | done | ADDR-TEMPLATE |  |
| SCH-49 | `dash:SegmentList`: Either the duration attribute or SegmentTimeline element shall be present but not both. | done | ADDR-TEMPLATE |  |
| SCH-50 | `dash:SegmentList`: If indexRange is not present indexRangeExact shall not be present. | done | MPD-INDEXRANGEEXACT | checked on the attributes as written |
| SCH-51 | `dash:SegmentBase`: If indexRange is not present indexRangeExact shall not be present. | done | MPD-INDEXRANGEEXACT | checked on the attributes as written |
| SCH-52 | `dash:SegmentBase`: The timeShiftBufferDepth shall not be smaller than timeShiftBufferDepth specified in the MPD element | done | MPD-SEGBASE-TSBD | upstream tests @timeShiftbuffer (a typo), so its rule never fires |
| SCH-53 | `dash:SegmentTimeline`: The d attribute of a SegmentTimeline shall not exceed the value give by the MPD maxSegmentDuration attribute. | done | MPD-MAXSEGDUR | upstream adds the duration components without unit conversion, so PT1M counts as 1 s |
| SCH-54 | `dash:ProgramInformation`: If more than one ProgramInformation element is given each ProgramInformation element shall have a lang attribute. | done | MPD-PROGINFO-LANG |  |
| SCH-55 | `dash:ContentProtection`: The value of ContentProtection shall be the 4CC contained in the Scheme Type Box | done | MPD-CP-VALUE | dashcheck accepts the percent-encoded 4CC and :version of ISO/IEC 23009-1 5.8.5.2.2 and a @value inherited through @ref; upstream requires exactly 4 characters (bug 56) |
| SCH-56 | `dash:ContentProtection`: The value of ContentProtection shall be the 4-digit lower-case hexadecimal Representation. | done | MPD-CP-VALUE | dashcheck requires 4 lower-case hex digits (ISO/IEC 23009-1 5.8.5.2.3); upstream checks only the length (bug 56) |
| SCH-57 | `dash:Role`: The value of Role (role) shall be caption, subtitle, main, alternate, supplementary, commentary or dub. | done | MPD-ROLE-VALUE | dashcheck accepts all role values of the 6th edition, upstream only the first seven |
| SCH-58 | `dash:Role`: The value of Role (stereoid) shall start with 'l' or 'r'. | done | MPD-ROLE-VALUE |  |
| SCH-59 | `dash:FramePacking`: The URI urn:mpeg:dash:14496:10:frame_packing_arrangement_type:2011 is used for Adaptation Sets or Representations that contain a video component that conforms to ISO/IEC 14496-10. | done | MPD-FRAMEPACKING | uses the effective @codecs of the Representation |
| SCH-60 | `dash:FramePacking`: The URI urn:mpeg:dash:13818:1:stereo_video_format_type:2011 is used for Adaptation Sets or Representations that contain a video component that conforms to ISO/IEC 13818-1. | done | MPD-FRAMEPACKING |  |
| SCH-61 | `dash:FramePacking`: schemeIdUri for FramePacking descriptor shall be urn:mpeg:dash:14496:10:frame_packing_arrangement_type:2011 or urn:mpeg:dash:13818:1:stereo_video_format_type:2011. | done | MPD-FRAMEPACKING |  |
| SCH-62 | `dash:FramePacking`: The value of FramePacking shall be 0 to 6 as defined in ISO/IEC 23091(all parts). | done | MPD-FRAMEPACKING |  |
| SCH-63 | `dash:AudioChannelConfiguration`: If URI urn:mpeg:dash:outputChannelPositionList:2012 is used the value attribute shall be a space-delimited list as defined in ISO/IEC 23091(all parts). | done | MPD-CHANNEL-POSITIONS |  |
| SCH-64 | `dash:EventStream`: If href is not present actuate shall not be present. | done | MPD-EVENTSTREAM |  |
| SCH-65 | `dash:EventStream`: schemeIdUri shall be present. | done | MPD-EVENTSTREAM |  |
| SCH-66 | `dash:Subset`: The id of each Subset shall be unique. | done | MPD-SUBSET-ID |  |
| SCH-67 | `dash:UTCTiming`: @schemeIdUri for UTCTiming is not one of the 7 different types specified. | done | MPD-UTCTIMING-SCHEME | dashcheck also warns for schemes that browsers cannot use |
| SCH-68 | `dash:SupplementalProperty`: If schemeIdUri urn:mpeg:dash:chaining:2016 is used, then value attribute shall be composed of the comma separated parameters (no comma needed if only first parameter is present). | done | MPD-CHAINING |  |
| SCH-69 | `dash:SupplementalProperty`: MPD fallback chaining shall be signaled by Supplemental Descriptor on MPD level with schemeIdUri urn:mpeg:dash:fallback:2016. | done | MPD-CHAINING |  |
| SCH-70 | `dash:SupplementalProperty`: If schemeIdUri urn:mpeg:dash:fallback:2016 is used, then value attribute shall be composed of one URL or whitespace separated URLs. | done | MPD-CHAINING |  |
| SCH-71 | `dash:Period`: When every Adaptation Set in a MPD has a SRD descriptor, at least one of this descriptor shall be a SupplementalProperty. | done | MPD-SRD-ESSENTIAL | upstream also fails for Periods without AdaptationSets |
| SCH-72 | `dash:Period`: For a given source_id of the @value attribute, at least one of the EssentialProperty or SupplementalProperty in the containing Period shall specify the optional parameters W and H. | done | MPD-SRD-TOTAL |  |
| SCH-73 | `dash:Period`: For a given source_id of the @value attribute, if two SRD elements (indistinctively EssentialProperty or SupplementalProperty) explicitly specify a different pair of values for the optional parameters (W,H) then all the remaining SRD element shall explicitly specify a pair of values for (W,H) too. | done | MPD-SRD-TOTAL |  |
| SCH-74 | `dash:Period`: For a given source_id of the @value attribute, the values of x, w and W shall be such that, for each descriptor, the sum of x and w is smaller or equal to W. | done | MPD-SRD-BOUNDS |  |
| SCH-75 | `dash:Period`: For a given source_id of the @value attribute, the values of x, w and W shall be such that, for each descriptor, the sum of x and w is smaller or equal to W. | done | MPD-SRD-BOUNDS | upstream passes if any source meets the bound and fails for sources without a total size; dashcheck checks every source |
| SCH-76 | `dash:Period`: For a given source_id of the @value attribute, the values of y, h and H shall be such that, for each descriptor, the sum of y and h is smaller or equal to H. | done | MPD-SRD-BOUNDS |  |
| SCH-77 | `dash:Period`: For a given source_id of the @value attribute, the values of y, h and H shall be such that, for each descriptor, the sum of y and h is smaller or equal to H. | done | MPD-SRD-BOUNDS | upstream passes if any source meets the bound and fails for sources without a total size; dashcheck checks every source |
| SCH-78 | `dash:SupplementalProperty \| dash:EssentialProperty`: An EssentialProperty or a SupplementalProperty descriptor with @schemeIdUri equal to “urn:mpeg:dash:srd:2014” shall be the child element of an AdaptationSet or a SubRepresentation element. | done | MPD-SRD-PLACEMENT |  |
| SCH-79 | `dash:SupplementalProperty \| dash:EssentialProperty`: If an EssentialProperty or a SupplementalProperty descriptor with @schemeIdUri equal to “urn:mpeg:dash:srd:2014” is present, then the @value attribute must be present. | done | MPD-SRD-VALUE |  |
| SCH-80 | `dash:SupplementalProperty \| dash:EssentialProperty`: If an EssentialProperty or a SupplementalProperty descriptor with @schemeIdUri equal to “urn:mpeg:dash:srd:2014” is present, then the @value attribute must contain at least the mandatory comma separated parameters, i.e. source_id, x, y, w, h. | done | MPD-SRD-VALUE |  |
| SCH-81 | `dash:SupplementalProperty \| dash:EssentialProperty`: If an EssentialProperty or a SupplementalProperty descriptor with @schemeIdUri equal to “urn:mpeg:dash:srd:2014” is present, then each parameter value has to match the expected type format i.e. non-negative integer in decimal representation. | done | MPD-SRD-VALUE |  |
| SCH-82 | `dash:SupplementalProperty \| dash:EssentialProperty`: If an EssentialProperty or a SupplementalProperty descriptor with @schemeIdUri equal to “urn:mpeg:dash:srd:2014” is present and the @value attribute contains the optional parameter W then the optional parameter H shall be present too. In addition, if the optional parameter spatial_set_id is present, then the optional parameters W and H shall be present. | done | MPD-SRD-VALUE |  |
| SCH-83 | `dash:MPD`: If MPD is of type "static" and if the profile contains a DASH-IF IOP profile, then the timeShiftBufferDepth shall not be defined. | done | MPD-IOP-STATIC |  |
| SCH-84 | `dash:MPD`: If MPD is of type "static" and if the profile contains a DASH-IF IOP profile, then the minimumUpdatePeriod shall not be defined. | done | MPD-IOP-STATIC |  |
| SCH-85 | `dash:MPD`: DASH-IF IOP Section 3.2.2.2: For dynamic MPD, the @profile shall include urn:mpeg:dash:profile:isoff-live:2011. | done | MPD-IOP-LIVE |  |
| SCH-86 | `dash:MPD`: DASH-IF IOP Section 3.10.2/3.10.3: "MPD@minBufferTime shall be present", not found. | done | MPD-IOP-MINBUFFER | the schema also requires @minBufferTime |
| SCH-87 | `dash:Period`: DASH-IF IOP Section 3.2.2: "the Period.SegmentList element shall not be present" violated here | done | MPD-PROFILE-SEGINFO |  |
| SCH-88 | `dash:Period`: DASH-IF IOP Section 3.2.2: "If a Period contains multiple Adaptation Sets with value of the @contentType="video" then at least one Adaptation Set shall contain a Role el-ement $&lt;$Role scheme="urn:mpeg:dash:role:2011" value="main"&gt;" violated here | done | MPD-PROFILE-MAINROLE |  |
| SCH-89 | `dash:Period`: DASH-IF IOP Section 3.10.4: "All Representations in one Period shall either conform to a DASH Live profile or DASH On-Demand profile", both Live and On-Demand profiles are signaled for the Period. | n/a |  | Period@profiles is not in the 6th edition schema, which rejects it |
| SCH-90 | `dash:AdaptationSet`: If the content author signals the ability of Adaptation Set switching and as @segmentAlignment or @subsegmentAlignment are set to TRUE for one Adaptation Set, the (Sub)Segment alignment shall hold for all Representations in all Adaptation Sets for which the @id value is included in the @value attribute of the Supplemental descriptor. | done | MPD-IOP-SWITCHING | dashcheck requires the adaptation-set-switching scheme; upstream matches any SupplementalProperty value |
| SCH-91 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with value of the @contentType="video" the following attributes shall be present: @par" violated here | done | MPD-IOP-VIDEO |  |
| SCH-92 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For Adaptation Set or for any Representation within an Adaptation Set with value of the @contentType="video" the attribute @scanType shall either not be present or shall be set to 'progressive' ", violated here | done | MPD-IOP-SCANTYPE |  |
| SCH-93 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with value of the @contentType="audio" the following attributes shall be present: @lang" violated here | done | MPD-IOP-AUDIO |  |
| SCH-94 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with @contentType="video" the following attributes shall be present: @maxWidth or @width" violated here | done | MPD-IOP-VIDEO |  |
| SCH-95 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with @contentType="video" the following attributes shall be present: @maxHeight or @height" violated here | done | MPD-IOP-VIDEO |  |
| SCH-96 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with @contentType="video" the following attributes shall be present: @maxFrameRate or @frameRate" violated here | done | MPD-IOP-VIDEO |  |
| SCH-97 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.2.2: For Live Profile @segmentAlignment shall be set to true for all Adaptation Sets | done | MPD-IOP-LIVE |  |
| SCH-98 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.2.2: For On-Demand Profile @subSegmentAlignment shall be set to true for all Adaptation Sets | done | MPD-IOP-ONDEMAND | upstream misspells the profile (isoff-ondemand) and the attribute (subSegmentAlignment), so its rule never fires |
| SCH-99 | `dash:AdaptationSet`: If a DASH-IF profile identifier is present, for the Adaptation Sets the mimeType shall be one of the six following type: "video/mp4", "audio/mp4", "application/mp4", "application/ttml+xml", "text/vtt" or "image/jpeg" | done | MPD-IOP-MIMETYPE |  |
| SCH-100 | `dash:AdaptationSet`: The value of @profiles shall be a subset of the respective value in any higher level of the document hierarchy | done | MPD-PROFILE-SUBSET | dashcheck compares individual profiles; upstream tests whether the whole value is a substring |
| SCH-101 | `dash:AdaptationSet`: DASH-IF IOP Section 3.10.3: "AdaptationSet@subsegmentAlignment SHALL be present and set to true", not set appropriately. | done | MPD-IOP-ONDEMAND |  |
| SCH-102 | `dash:AdaptationSet`: DASH-IF IOP Section 3.10.3: "AdaptationSet@subsegmentStartsWithSAP SHALL be present and set to 1 or 2", not set appropriately. | done | MPD-IOP-ONDEMAND | upstream tests a child element instead of the attribute, so it fails for every AdaptationSet |
| SCH-103 | `dash:AdaptationSet`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - For HbbTV profile, @segmentAlignment shall be set to true for all Adaptation Sets as it is based on live profile | done | MPD-HBBTV |  |
| SCH-104 | `dash:Representation`: DASH-IF IOP (v3.3), Section 3.2.13 : "In contrast to MPEG-DASH which does not prohibit the use of multiplexed Representations, in the DASH-IF IOPs one Adaptation Set always contains exactly a single media type.". | done | MPD-IOP-MIMETYPE |  |
| SCH-105 | `dash:Representation`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with value of the @contentType="video" the following attributes shall be present: @maxWidth (or @width if all Representations have the same width), @maxHeight (or @height if all Representations have the same width), @maxFrameRate (or @frameRate if all Representations have the same width)" violated here | done | MPD-IOP-VIDEO |  |
| SCH-106 | `dash:Representation`: DASH-IF IOP Section 3.2.4: "For any Representation within an Adaptation Set with value of the @contentType="video" the following attributes shall be present: @width, if not present in AdaptationSet element; @height, if not present in AdaptationSet element; @frameRate, if not present in AdaptationSet element; @sar" violated here | done | MPD-IOP-VIDEO | dashcheck also accepts @sar on the AdaptationSet |
| SCH-107 | `dash:Representation`: DASH-IF IOP Section 3.2.4: "For Adaptation Set or for any Representation within an Adaptation Set with value of the @contentType="video" the attribute @scanType shall either not be present or shall be set to 'progressive' ", violated here | done | MPD-IOP-SCANTYPE |  |
| SCH-108 | `dash:Representation`: DASH-IF IOP Section 3.2.4: "For any Representation within an Adaptation Set with value of the @contentType="audio" the following elements and attributes shall be present: @audioSamplingRate, if not present in AdaptationSet element; AudioChannelConfiguration, if not present in AdaptationSet element" violated here | done | MPD-IOP-AUDIO |  |
| SCH-109 | `dash:Representation`: If profiles contain dash-if profile identifier, "codecs" attribute on AdaptationSet level OR Representation level shall not contain more than one identifiers as a comma separated list | done | MPD-IOP-CODECS |  |
| SCH-110 | `dash:Representation`: The @dependencyId attribute on the Enhancement Layer Representation shall refer to the Base Layer Representation @id attribute. | done | MPD-IOP-DEPENDENCY |  |
| SCH-111 | `dash:Representation`: The value of @profiles shall be a subset of the respective value in any higher level of the document hierarchy. | done | MPD-PROFILE-SUBSET | dashcheck compares individual profiles; upstream tests whether the whole value is a substring |
| SCH-112 | `dash:Representation`: DASH-IF IOP 4.3 Section 3.2.1 - "For on-demand profiles, @indexRange attribute shall be present. | done | MPD-IOP-ONDEMAND |  |
| SCH-113 | `dash:SubRepresentation`: The value of @profiles shall be a subset of the respective value in any higher level of the document hierarchy | done | MPD-PROFILE-SUBSET | dashcheck compares individual profiles; upstream tests whether the whole value is a substring |
| SCH-114 | `dash:SegmentBase`: DASH-IF IOP 4.3 Section 3.2.1 - "For on-demand profiles, @indexRange attribute shall be present. | done | MPD-IOP-ONDEMAND |  |
| SCH-115 | `dash:ContentProtection`: The ContentProtection descriptors shall always be present in the AdaptationSet element and apply to all contained Representations. | done | MPD-IOP-CP |  |
| SCH-116 | `dash:ContentProtection`: The ContentProtection descriptor for the mp4 protection scheme with @schemeIdUri 'urn:mpeg:dash:mp4protection:2011' and @value 'cenc' shall be present in the AdaptationSet element. | done | MPD-IOP-CP |  |
| SCH-117 | `dash:ContentProtection`: The ContentProtection Descriptor for the mp4protection scheme shall contain the attribute @cenc:default_KID. | done | MPD-IOP-CP |  |
| SCH-118 | `dash:AudioChannelConfiguration`: If profile http://dashif.org/guidelines/dashif#ac-4 is used, then schemeIdUri attribute shall be tag:dolby.com,2014:dash:audio_channel_configuration:2011. | done | MPD-AUDIO-CHANNELCONFIG |  |
| SCH-119 | `dash:AudioChannelConfiguration`: If profile http://dashif.org/guidelines/dashif#mpeg-h-3da is used, then schemeIdUri attribute shall be urn:mpeg:mpegB:cicp:ChannelConfiguration. | done | MPD-AUDIO-CHANNELCONFIG |  |
| SCH-120 | `dash:EssentialProperty`: If EssentialProperty descriptor with @schemeIdUri set to http://dashif.org/guidelines/thumbnail_tile is present, then value shall provide horizontal and vertical number of tiles separated by an 'x'. | done | MPD-THUMBNAIL |  |
| SCH-121 | `dash:EssentialProperty`: If EssentialProperty descriptor with @schemeIdUri set to http://dashif.org/guidelines/thumbnail_tile is present, then bandwidth shall be used to describe the tiling.'. | done | MPD-THUMBNAIL |  |
| SCH-122 | `dash:EssentialProperty`: If EssentialProperty descriptor with @schemeIdUri set to http://dashif.org/guidelines/thumbnail_tile is present, then height shall be used to describe the tiling. | done | MPD-THUMBNAIL |  |
| SCH-123 | `dash:EssentialProperty`: If EssentialProperty descriptor with @schemeIdUri set to http://dashif.org/guidelines/thumbnail_tile is present, then width shall be used to describe the tiling. | done | MPD-THUMBNAIL |  |

## Schematron: dvb-dash.sch

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| SCH-DVB-1 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:AdaptationSet[dlb:isAdaptationSetAudio(.)][not(dlb:isAuxiliaryStream(.))]/dash:Representation`: @mimeType shall be common between all Representations in an Adaptation Set | done | MPD-DVB-AUDIO-COMMON | upstream compares each Representation with all of them, itself included, so it only fails when a Representation lacks @mimeType |
| SCH-DVB-2 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:AdaptationSet[dlb:isAdaptationSetAudio(.)][not(dlb:isAuxiliaryStream(.))]/dash:Representation`: @codecs should be common between all Representations in an Adaptation Set | done | MPD-DVB-AUDIO-SIMILAR | upstream compares each Representation with all of them, itself included, so it only fails when a Representation lacks @codecs |
| SCH-DVB-3 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:AdaptationSet[dlb:isAdaptationSetAudio(.)][not(dlb:isAuxiliaryStream(.))]/dash:Representation`: @audioSamplingRate should be common between all Representations in an Adaptation Set | done | MPD-DVB-AUDIO-SIMILAR | upstream compares each Representation with all of them, itself included, so it only fails when a Representation lacks @audioSamplingRate |
| SCH-DVB-4 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:AdaptationSet[dlb:isAdaptationSetAudio(.)][not(dlb:isAuxiliaryStream(.))]/dash:Representation/dash:AudioChannelConfiguration`: audioChannelConfiguration should be common between all Representations in an Adaptation Set | done | MPD-DVB-AUDIO-SIMILAR |  |
| SCH-DVB-5 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:AdaptationSet[dlb:isAdaptationSetAudio(.)][dlb:isAuxiliaryStream(.)]`: All Adaptation Sets that refer to Auxiliary Audio streams may not contain the @lang attribute and Role, Accessibility, AudioChannelConfiguration descriptors | done | MPD-DVB-AUXILIARY |  |
| SCH-DVB-6 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:Preselection[dlb:isAdaptationSetAudio(.)]`: Every AC-4 or MPEG-H Audio Preselection element shall include at least one Role element using the scheme "urn:mpeg:dash:role:2011" as defined in ISO/IEC 23009-1:2019 [1]. | done | MPD-DVB-PRESELECTION |  |
| SCH-DVB-7 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:Preselection[dlb:isAdaptationSetAudio(.)]`: If there is more than one audio Preselection associated with an audio bundle, at least one of the Preselection elements shall be tagged with an @value set to "main". | done | MPD-DVB-PRESELECTION |  |
| SCH-DVB-8 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:Preselection[dlb:isAdaptationSetAudio(.)]`: @preselectionComponents specifies the ids of the contained Adaptation Sets or Content Components that belong to this Preselection as white space separated list in processing order. | done | MPD-DVB-PRESELECTION |  |
| SCH-DVB-9 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]//*[self::dash:AdaptationSet or self::dash:Preselection or self::Representation][dlb:isAdaptationSetAC4(.)]`: AC-4 audio should be encoded using bitstream_version = 2 (is ). | done | MPD-DVB-AC4-VERSION | upstream's rule context lacks the dash: prefix for Representation, so Representations are never checked |
| SCH-DVB-10 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]//*[self::dash:AdaptationSet or self::dash:Preselection or self::Representation][dlb:isAdaptationSetAC4(.)]`: The presentation_version field according to clause 6.2.1.3 of ETSI TS 103 190-2 [46] shall be set to the value 1. | done | MPD-DVB-AC4 | upstream's rule context lacks the dash: prefix for Representation, so Representations are never checked |
| SCH-DVB-11 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]//*[self::dash:AdaptationSet or self::dash:Preselection or self::Representation][dlb:isAdaptationSetAC4(.)]`: The md_compat field as defined by clause 6.3.2.2.3 of ETSI TS 103 190-2 [46] shall be less than or equal to three. | done | MPD-DVB-AC4 | upstream's rule context lacks the dash: prefix for Representation, so Representations are never checked |
| SCH-DVB-12 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]//*[self::dash:AdaptationSet or self::dash:Preselection or self::Representation][dlb:isAdaptationSetAC4(.)]`: An AC-4 elementary stream shall be encoded with a sampling rate of 48 kHz, 96 kHz or 192 kHz. | done | MPD-DVB-AC4 | upstream's rule context lacks the dash: prefix for Representation, so Representations are never checked |
| SCH-DVB-13 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]//*[self::dash:AdaptationSet or self::dash:Preselection or self::Representation][dlb:isAdaptationSetAC4(.)]`: An Adaptation Set shall not contain audio with a frame rate &gt; 60 Hz unless all video adaptationSets in the Period contain only video with a frame rate &gt; 60 Hz. | done | MPD-DVB-AC4 |  |

## Schematron: ac4-generic.sch

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| SCH-AC4-1 | `*[self::dash:AdaptationSet or self::dash:Preselection][dlb:isAdaptationSetAC4(.)]`: The @codecs attribute shall conform to the syntax described in IETF RFC 6381. The value of the parameter shall be set to a dot-separated list of four parts of which the last three are two-digit hexadecimal numbers. | done | MPD-AC4-CODECS |  |
| SCH-AC4-2 | `*[self::dash:AdaptationSet or self::dash:Preselection][dlb:isAdaptationSetAC4(.)]`: The value of the mimeType attribute shall be set to 'audio/mp4' or 'video/mp4'. | done | MPD-AC4-ATTRS | upstream's test is inverted and fails for audio/mp4 and video/mp4 |
| SCH-AC4-3 | `*[self::dash:AdaptationSet or self::dash:Preselection][dlb:isAdaptationSetAC4(.)]`: @audioSamplingRate shall be set to the sampling frequency derived from the parameters fs_index and dsi_sf_multiplier, contained in ac4_dsi_v1. | done | MPD-AC4-ATTRS |  |
| SCH-AC4-4 | `*[self::dash:AdaptationSet or self::dash:Preselection][dlb:isAdaptationSetAC4(.)]`: The value of the mimeType attribute shall be set to 'audio/mp4'. | done | MPD-AC4-ATTRS |  |
| SCH-AC4-5 | `*[self::dash:AdaptationSet or self::dash:Preselection][dlb:isAdaptationSetAC4(.)]`: The @startWithSAP value shall be set to '1'. | done | MPD-AC4-ATTRS |  |
| SCH-AC4-6 | `dash:Representation[matches(dlb:getNearestCodecString(.), 'ac-4\.00','i')]`: The representation DASH element shall include an AudioChannelConfiguration DASH descriptor. | done | MPD-AC4-CHANNELCONFIG | dashcheck also accepts an AudioChannelConfiguration inherited from the AdaptationSet |
| SCH-AC4-7 | `dash:Role[matches(dlb:getNearestCodecString(.), 'ac-4\.00')]`: The value of Role (role) shall be main, alternate, commentary. | done | MPD-AC4-ROLE |  |
| SCH-AC4-8 | `dash:SupplementalProperty[@schemeIdUri = 'tag:dolby.com,2016:dash:virtualized_content:2016'][matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')]`: The @value attribute of the immersive audio for headphones descriptor shall be 1 | done | MPD-AC4-VIRTUALIZED |  |
| SCH-AC4-9 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.00')][@schemeIdUri eq $NSDLB_acc2014]`: It is recommended to use exactly one AudioChannelConfiguration element with a schemeIdURI of or | done | MPD-AC4-CHANNELCONFIG-COUNT |  |
| SCH-AC4-10 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.00')][@schemeIdUri eq $NSDLB_acc2014]`: The value element shall contain a four-digit hexadecimal representation of the 16-bit field which describes the channel assignment of the referenced AC-4 elementary stream | done | MPD-AC4-CHANNELCONFIG |  |
| SCH-AC4-11 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.00')][@schemeIdUri eq $NSDLB_acc2014]`: Use &lt; schemeIdUri="" value=""/&gt; | done | MPD-AC4-CHANNELCONFIG |  |
| SCH-AC4-12 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][not(@schemeIdUri = ($NSDLB_acc2015,$NSMPEG_acc))]`: Unspecified schemeIdUri in AudioChannelConfiguration element | done | MPD-AC4-CHANNELCONFIG-COUNT |  |
| SCH-AC4-13 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][@schemeIdUri eq $NSDLB_acc2015]`: It is recommended to use exactly one AudioChannelConfiguration element with a schemeIdURI of or | done | MPD-AC4-CHANNELCONFIG-COUNT |  |
| SCH-AC4-14 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][@schemeIdUri eq $NSDLB_acc2015]`: The value element shall contain a six-digit hexadecimal representation of the 24-bit field which describes the channel assignment of the referenced AC-4 elementary stream | done | MPD-AC4-CHANNELCONFIG |  |
| SCH-AC4-15 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][@schemeIdUri eq $NSDLB_acc2015]`: For all AC-4 channel configurations that are mappable to the MPEG channel configuration scheme, the scheme described by @schemeIdUri="urn:mpeg:mpegB:cicp:ChannelConfiguration" shall be used | done | MPD-AC4-CHANNELCONFIG |  |
| SCH-AC4-16 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][@schemeIdUri eq $NSMPEG_acc]`: It is recommended to use exactly one AudioChannelConfiguration element with a schemeIdURI of or | done | MPD-AC4-CHANNELCONFIG-COUNT |  |
| SCH-AC4-17 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][@schemeIdUri eq $NSMPEG_acc]`: Valid values are 1-7, 9-12, 14, 16-17, and 19 | done | MPD-AC4-CHANNELCONFIG |  |

## Module: Common

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| COMMON-1 | `Conformance Tool`: Schematron shall be able to run | n/a |  | checks that the upstream Schematron engine runs |
| COMMON-2 | `DASH 5th`: MPD SHALL be valid | done | MPD-XSD, MPD-XML | dashcheck uses the 6th edition schema |
| COMMON-3 | `DASH 2nd (DVB 2018)`: MPD SHALL be valid | done | MPD-DVB-XSD-2014 | information, for DVB MPDs: the 2nd edition schema with Amd 1, since DVB-DASH requires UTCTiming; upstream uses the schema without Amd 1 (bug 50) |
| COMMON-4 | `DASH 4th-amd1 (DVB 2019)`: MPD SHALL be valid | done | MPD-DVB-XSD | a warning for DVB MPDs: the 4th edition with Amd 1, which DVB-DASH V1.3.1 and V1.4.1 reference; only the first error is reported |
| COMMON-5 | Segment analysis should not result in errors | done | INIT-PARSE, SEG-PARSE |  |

## Module: IOP

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| IOP-1 | `3.2.7.4`: MPD fields having datatype xs:duration shall not use year or month units | done | MPD-DURATION-YM |  |
| IOP-2 | `3.10.3`: SegmentTemplate@media attribute SHALL be present | done | ADDR-TEMPLATE |  |
| IOP-3 | `3.10.4`: @profiles signalling SHALL be present in each Period | n/a |  | Period@profiles is not in the 6th edition schema, which rejects it |
| IOP-4 | `3.10.3`: SegmentTemplate@indexRange attribute SHALL be present | done | MPD-IOP-ONDEMAND |  |
| IOP-5 | `6.2.5.2`: All AVC representations SHALL be encoded using avc3 | done | SEG-IOP-BITSTREAM | dashcheck checks the sample entry and requires @bitstreamSwitching true; upstream checks @codecs whenever @bitstreamSwitching is present |
| IOP-6 | `6.2.5.2`: All AVC representations SHALL include an 'avcC' box containing SPS and PPS NALs | done | SEG-IOP-PARAMSETS |  |
| IOP-7 | `6.2.5.2`: The level of the adaptation set SHALL match the maximum of the corresponding representations | done | SEG-IOP-ASCODECS | upstream compares the AVC level in @codecs (hex) with L plus the decimal level, so it always fails for AVC, and it fails when the AdaptationSet has no @codecs |
| IOP-8 | `6.2.5.2`: The profile of the adaptation set SHALL match the maximum of the corresponding representations | done | SEG-IOP-ASCODECS |  |
| IOP-9 | `6.2.5.2`: The edit list, if present in one, SHALL be present in all representations | done | SEG-IOP-EDITLIST |  |
| IOP-10 | `6.2.5.2`: All HEVC representations SHALL be encoded using hev1 | done | SEG-IOP-BITSTREAM | dashcheck checks the sample entry and requires @bitstreamSwitching true |
| IOP-11 | `6.2.5.2`: All HEVC representations SHALL include an 'hvcC' box containing SPS, PPS and VPS NALs | done | SEG-IOP-PARAMSETS |  |
| IOP-12 | `3.10.3.2`: Only a single 'sidx' box shall be present | done | SEG-IOP-SIDX | dashcheck counts the sidx boxes in @indexRange |
| IOP-13 | `3.10.3.2`: Each representation SHALL have on segment | done | MPD-IOP-ONDEMAND | dashcheck requires SegmentBase addressing, which means a single segment |
| IOP-14 | `6.2.5.2`: Edit lists SHALL NOT be present unless On-Demand profile is signalled | done | SEG-IOP-EDITLIST |  |
| IOP-15 | `6.2.5.2`: The composition time and decoded time for the first sample SHALL match | done | SEG-IOP-FIRSTCT |  |

## Module: CMAF

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| CMAF-1 | `Section A.3`: The maximum encoding parameters [..as per the table..] SHALL not be exceeded | done | SEG-CMAF-MEDIAPROFILE |  |
| CMAF-2 | `Section A.1.2`: Audio adaptation sets SHALL include at least one 'caac' representation | done | SEG-CMAF-PRESENTATIONPROFILE |  |
| CMAF-3 | `Section 9.2.1`: Video tracks SHALL contain either 'trun(v1)' OR 'elst' combined with 'trun(v0)', but not both | done | SEG-CMAF-COMPTIME | upstream requires an edit list with trun version 0 even when there are no composition offsets |
| CMAF-4 | `Section 7.3.5`: All media samples in a CMAF Fragment SHALL be addressed by byte offsets in the 'trun' box | done | SEG-CMAF-DATAOFFSET, SEG-PARSE | upstream compares trun sample sizes with parsed NAL samples; dashcheck requires moof-relative data offsets, and mp4ff checks that samples lie within mdat |
| CMAF-5 | `Section 7.3.2.2`: baseMediaDecodeTime SHALL be equal to the sum of all prior segments | done | SEG-CONTINUITY, SEG-CONTINUITY-FRAGMENT | dashcheck checks the decode time between consecutive checked segments and between the chunks and fragments of each segment, with the sample durations from trun, tfhd or trex; upstream compares the first tfdt of a segment with every fragment (bug 52) and uses only the tfhd default duration (bug 23) |
| CMAF-6 | `Section A.1.2`: Tracks SHALL NOT contain a track encryption box | done | SEG-CMAF-ENCRYPTION |  |
| CMAF-7 | `Section A.1.3`: Tracks SHALL be available in 'cenc' encryption | done | SEG-CMAF-ENCRYPTION |  |
| CMAF-8 | `Section A.1.4`: Tracks SHALL be available in 'cbcs' encryption | done | SEG-CMAF-ENCRYPTION |  |
| CMAF-9 | `Section B.2.4`: CMAF Switching sets SHALL be constrained to include identical SPS information | done | SEG-CMAF-HEVC-SWITCHING |  |
| CMAF-10 | `Section B.2.4`: CMAF Switching sets SHALL be constrained to include identical SEI nals | done | SEG-CMAF-HEVC-SWITCHING | dashcheck compares the SEI NAL units in hvcC; upstream compares NAL header fields |
| CMAF-11 | `Section 7.3.5`: Each AVC CMAF Fragment SHALL be independently accessible | done | SEG-CMAF-INIT | like upstream, checks that the decoder configuration and size are present |
| CMAF-12 | `Section 7.3.5`: Each HEVC CMAF Fragment SHALL be independently accessible | done | SEG-CMAF-INIT | like upstream, checks that the decoder configuration and size are present |
| CMAF-13 | `Section 7.3.5`: Each Audio CMAF Fragment SHALL be independently accessible | done | SEG-CMAF-INIT | like upstream, checks that sample rate and channel count are present |
| CMAF-14 | `Section 7.3.2.4`: The HEVCSampleEntry SHALL contain a 'colr' box with type 'nclx' | done | SEG-CMAF-COLR | dashcheck checks hvc1 and hev1, upstream only hev1 |
| CMAF-15 | `Section 7.3.2.4`: Each encrypted Fragment SHALL be independently decryptable | done | SEG-CMAF-SENC | dashcheck checks every encrypted CMAF track, fragment by fragment: a fragment is clear when its sample entry has no sinf or a seig group marks it unprotected, and needs sample auxiliary information (senc, a PIFF uuid box or saio) only for subsample-encrypted video or per-sample IVs (ISO/IEC 23000-19 8.2.2.1); upstream checks only with the dash264 profile and counts senc boxes against moof boxes (bug 57) |
| CMAF-16 | `Section 7.5.2`: When metadata is carried in a 'meta' box, it SHALL NOT occur at the file level | done | SEG-CMAF-TOPLEVEL |  |
| CMAF-17 | `Section 7.5.2`: When metadata is carried in a 'udta' box, it SHALL NOT occur at the file level | done | SEG-CMAF-TOPLEVEL |  |
| CMAF-18 | `Section 7.3.9.1`: A CMAF switching set SHALL have only media type | done | SEG-CMAF-SWITCHINGSET |  |
| CMAF-19 | `Section 7.3.3.3`: Each subsegment referenced in the 'sidx' box SHALL be a single fragment | done | SEG-CMAF-SIDX |  |
| CMAF-20 | `Section 7.3.4.1`: All CMAF subtitle tracks in a CMAF Switching Set SHALL conform to one CMAF Media Profile | done | SEG-CMAF-SUBTITLEPROFILE | upstream only checks AdaptationSets whose @mimeType starts with text/, which excludes CMAF subtitles in application/mp4 |
| CMAF-21 | `Section 7.5.20`: All CMAF fragments in a 'im1i' track SHALL contain a 'subs' box | done | SEG-CMAF-SUBS | not implemented upstream, where the check always fails |
| CMAF-22 | `Section A.2 (AVC) / B.6 (HEVC)`: The maximum encoding parameters [..as per the table..] SHALL not be exceeded | done | SEG-CMAF-MEDIAPROFILE | upstream never checks the frame rate (always 0), treats missing colour information as BT.709, and asks for an HEVC profile named Main8 for cud8, which always fails; dashcheck checks the frame rate, skips unsignaled colour information and uses Main for cud8 |
| CMAF-23 | `Section A.1.2`: Video adaptation sets SHALL include at least one 'cfhd' representation | done | SEG-CMAF-PRESENTATIONPROFILE |  |

## Module: Low Latency

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| LL-1 | `9.X.4.1`: Media presentation offering Low Latency SHOULD be signalled with … | done | MPD-LL-PROFILE | dashcheck detects low latency from @availabilityTimeOffset and @availabilityTimeComplete |
| LL-2 | `9.X.4.2`: Inband Event streams carrying MPD validity expiration events SHOULD be present | done | MPD-LL-INBAND |  |
| LL-3 | `9.X.4.2`: If Inband Event streams carrying MPD validity expiration events are used, @value SHALL be set to 1 | done | MPD-LL-INBAND-VALUE |  |
| LL-4 | `9.X.4.2`: LeapSecondInformation SHOULD be provided | done | MPD-LL-LEAPSECOND | upstream looks up a misspelled element name and always fails |
| LL-5 | `9.X.4.3`: At least one ProducerTimeReference SHALL be present | done | MPD-LL-PRT | an absent @type means encoder; dash-mpd cannot tell an absent @presentationTime from 0 |
| LL-6 | `9.X.4.3`: Each ProducerTimeReference SHALL have a unique @id | done | MPD-LL-PRT | an @id identifies one reference in the whole MPD (CR r8 9.X.6.2.3), so dashcheck allows it to repeat only for an identical reference (livesim2 copies the PRT into each AdaptationSet); upstream fails every repeated @id |
| LL-7 | `9.X.4.3`: Either (constrained) SegmentTemplate or SegmentTimeline shall be present | done | MPD-LL-ADDRESSING |  |
| LL-8 | `9.X.4.3`: SegmentTemplate will have @duration set and have @media containing $Number$ | done | MPD-LL-ADDRESSING |  |
| LL-9 | `9.X.4.3`: SegmentTimeline will have @media containing $Number$ and $Time$ | done | MPD-LL-ADDRESSING | upstream reads @media from SegmentTimeline and requires both $Number$ and $Time$, so it never passes |
| LL-10 | `9.X.4.2`: At least one valid ServiceDescription element SHALL be present | done | MPD-LL-SERVICEDESC, MPD-LL-SERVICEDESC-OPTIONAL |  |
| LL-11 | `9.X.4.2`: At least one UTC timing description element SHALL be present with a correct @schemIdUri | done | MPD-LL-UTCTIMING |  |
| LL-12 | `9.X.4.5`: Each Segment SHALL Conform to a CMAF Fragment | done | SEG-STRUCTURE, SEG-CMAF-CHUNK, SEG-CMAF-SAP, SEG-CMAF-DATAOFFSET, SEG-CONTINUITY-FRAGMENT | dashcheck applies the CMAF chunk and fragment rules to the segments of low-latency chunked Representations whatever their brands: one traf with one trun per moof, data offsets, decode time continuity, and a sync sample first in video; upstream always passes (bug 32) |
| LL-13 | `9.X.4.5`: EMSG box constraints | done | SEG-EMSG-REPEAT | item 6 of 9.X.4.5: an emsg between moof and mdat fails parsing (SEG-PARSE); an emsg after the first moof must be repeated before the first moof of the next segment; upstream always passes |
| LL-14 | `9.X.4.5`: Each Segment MAY (and typically SHOULD) contain more than one CMAF chunk | done | SEG-LL-CHUNKS |  |
| LL-15 | `9.X.4.5`: A Resync element SHOULD be assigned to each Representation | done | MPD-LL-RESYNC |  |
| LL-16 | `9.X.4.5`: @availabilityTimeOffset SHALL be present for each represention | done | MPD-LL-CHUNKED | without the low-latency profile, dashcheck applies the low-latency rules only when a Representation has `@availabilityTimeComplete` false, since `@availabilityTimeOffset` alone makes complete Segments available earlier (ISO/IEC 23009-1 5.3.9.5.3) |
| LL-17 | `9.X.4.5`: @availabilityTimeComple SHALL be present and set to 'FALSE' for each representation | done | MPD-LL-CHUNKED | upstream compares with FALSE and always fails |
| LL-18 | `9.X.4.5`: One of the following should hold | done | SEG-LL-DURATION-RECOMMENDED | upstream's second option is a stub that returns false; dashcheck warns when a chunked segment exceeds the target latency |
| LL-19 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @contentType shall correspond with the hdlr type | done | XCHK-HANDLER |  |
| LL-20 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @mimeType SHALL be either '&lt;@contentType&gt;/mp4' or &lt;#contentType&gt;/mp4, profiles='cmfc' | done | MPD-LL-MIMETYPE | dashcheck also accepts `application/mp4` for `@contentType` text: it is the media type of all CMAF text tracks (ISO/IEC 23000-19 Table 2), and the one ISO/IEC 23009-1 8.12.4 gives for TTML, whose handler is subt |
| LL-21 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @maxWidth SHOULD correspond to the width in the 'tkhd' box | done | XCHK-CMAF-SIZE | dashcheck compares @maxWidth and @maxHeight with the largest tkhd size of the AdaptationSet, which the CMAF principal header has, and without them each @width and @height with its own tkhd; @width may also be on the @sar grid. The DASH profile for CMAF content applies with the CMAF profile or to low-latency chunked Representations. ISO/IEC 23009-1 8.12.4.3 says shall, so an error; upstream warns and compares @maxWidth with every track (bug 53) |
| LL-22 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @maxHeight SHOULD correspond to the height in the 'tkhd' box | done | XCHK-CMAF-SIZE | dashcheck compares @maxWidth and @maxHeight with the largest tkhd size of the AdaptationSet, which the CMAF principal header has, and without them each @width and @height with its own tkhd; @width may also be on the @sar grid. The DASH profile for CMAF content applies with the CMAF profile or to low-latency chunked Representations. ISO/IEC 23009-1 8.12.4.3 says shall, so an error; upstream warns and compares @maxWidth with every track (bug 53) |
| LL-23 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @codecs shall correspond with the sample descriptor type | done | XCHK-CODECS, XCHK-CODECS-DETAIL |  |
| LL-24 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: If the content is protected, a ContentProtection element SHALL be present and set appropriately | done | XCHK-CP-MISSING, XCHK-CP-SCHEME, XCHK-CP-KID | dashcheck checks all encrypted tracks, not only low latency: the mp4protection descriptor for Common Encryption, its @value against the schm scheme type, and the cenc:default_KID of every descriptor against the tenc KID; upstream compares the KIDs in different formats and passes when any one descriptor matches (bug 55) |
| LL-25 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @timescale SHALL correspond to the timescale in the 'mdhd' box | done | SEG-LL-TIMESCALE |  |
| LL-26 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: Either segmentAlignment or subsegmentAlignment SHALL be set | done | MPD-LL-ALIGNMENT |  |
| LL-27 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: Event message streams MAY be signalled with InbandEventStream elements | n/a |  | a permissive (MAY) statement |
| LL-28 | `9.X.4.5`: All 'emsg' boxes inserted [..] after the start of the first CMAF chunk … | done | SEG-EMSG-REPEAT | compared between consecutive checked segments; upstream always passes |
| LL-29 | `9.X.4.5`: Resync element SHALL/SHOULD be present (conditional) | done | MPD-LL-RESYNC |  |
| LL-30 | `9.X.4.5`: Resync element type SHALL be valid | done | MPD-LL-RESYNC-VALUE |  |
| LL-31 | `9.X.4.5`: Resync element @marker SHALL be 'true' | done | MPD-LL-RESYNC-VALUE | upstream compares with TRUE and always fails |
| LL-32 | `9.X.4.5`: Resync element @dT SHOULD be lower than half the latency, and SHALL not exceed the target latency | done | MPD-LL-RESYNC, MPD-LL-RESYNC-VALUE | dashcheck converts @dT from the Representation timescale to milliseconds; upstream compares the raw value |
| LL-33 | `9.X.4.5`: @qualityRanking should be used | done | MPD-LL-RESYNC |  |
| LL-34 | `9.X.4.4`: Each Segment SHOULD include only a single 'moof' box | done | SEG-LL-SINGLEMOOF |  |
| LL-35 | `9.X.4.4`: A segment with a single 'moof' box may carry an 'smds' brand | done | SEG-LL-SMDS |  |
| LL-36 | `9.X.4.4`: A segment SHALL NOT exceed 50% and SHOULD not exceed 30% of the tareget latency | done | SEG-LL-DURATION, SEG-LL-DURATION-RECOMMENDED | upstream inverts both limits |
| LL-37 | `9.X.4.3`: A Low Latency Adaptation Set SHALL either be a Segment or a Chunked Set | done | MPD-LL-MODE |  |
| LL-38 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The tolerance for the EPT relative to the first stegment SHALL NOT exceed 50% | done | SEG-EPT | for all number-based streams, not only low latency |
| LL-39 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: If an 'S' elements and it predecessor share the same duration, they SHOULD be combined | done | MPD-LL-TIMELINE-REPEAT | S elements with @t start a new run and are not flagged |
| LL-40 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: If each chunk is an addressable object, '@k SHALL be present in the SegmentTimeline | done | MPD-LL-K | chunks are addressable when the SegmentTemplate@media has $SubNumber$; @k may be on S or SegmentTemplate |
| LL-41 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 't' attribute SHALL correspond to the EPT of the corresponding segment | done | SEG-EPT, SEG-EPT-CTO | for all SegmentTimeline streams, not only low latency |
| LL-42 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'd' attribute SHALL correspond to the duration of the corresponding segment | done | SEG-DURATION | for all SegmentTimeline streams, not only low latency |
| LL-43 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: Exactly one 'sidx' box shall be used | done | SEG-LL-SIDX |  |
| LL-44 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box shall be placed before any 'moof' boxes | done | SEG-LL-SIDX | with SegmentBase the index range precedes the media by construction |
| LL-45 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box reference_ID SHALL be equal to the track id | done | SEG-LL-SIDX |  |
| LL-46 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box timescale SHALL be identical to the one in the 'mdhd' box | done | SEG-LL-SIDX |  |
| LL-47 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box earliest presentation time SHALL match the segment EPT | done | SEG-LL-SIDX |  |
| LL-48 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box reference count SHALL match the number of 'mmof' boxes | done | SEG-CMAF-SIDX, SEG-SIDX-SPAN | dashcheck checks that each fetched subsegment of a CMAF track is one fragment (every subsegment with -n -1), and reads the box headers after the sidx box and after the last subsegment to find movie fragments that the sidx does not index; this applies to all indexed segments (ISO/IEC 23009-1 6.3.5.4) |
| LL-49 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box reference_type shall be set to 0 | done | SEG-LL-SIDX |  |
| LL-50 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box startwithsap shall be set to 1 | done | SEG-LL-SIDX |  |
| LL-51 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box sapType shall be set to 1 or 2 | done | SEG-LL-SIDX |  |
| LL-52 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box sapDeltaTime shall be set to 0 | done | SEG-LL-SIDX |  |
| LL-53 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The baseMediaDecodeTime shall be 0 | done | SEG-LL-SIDX |  |
| LL-54 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The list of compatible brands SHALL contain 'dash' | done | SEG-LL-SIDX |  |

## Module: DVB

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| DVB-1 | `Unknown`: MPD@minimumUpdatePeriod SHOULD have a value of 1 second or higher | done | MPD-DVB-MUP |  |
| DVB-2 | `Section 7.2.1.1`: A fontdownload descriptor SHALL only be placed in AdaptationSets containing subtitles | done | MPD-DVB-FONT |  |
| DVB-3 | `Section 6.3.1`: The AudioChannelConfiguration shall be compliant with the dolby scheme | done | MPD-DVB-AUDIO | dashcheck requires the Dolby 2014 scheme for E-AC-3 only; AC-4 schemes are checked by the ac4-generic.sch rules; upstream never checks AC-4 because of reversed arguments |
| DVB-4 | `Section 6.1.2`: Each audio AdaptationSet shall include at least one Role Element | done | MPD-DVB-AUDIO |  |
| DVB-5 | `Section 6.1.1`: All audio Representations SHALL have the attributes and elements in Table 4 | done | MPD-DVB-AUDIO |  |
| DVB-6 | `Section 6.1.1`: @mimeType SHALL be common for all audio Representations within an AdaptationSet | done | MPD-DVB-AUDIO-COMMON | upstream tests an empty array and always fails |
| DVB-7 | `Section 6.1.1`: @codec SHOULD be common for all audio Representations within an Adapation Set | done | MPD-DVB-AUDIO-SIMILAR | upstream tests an empty array and always fails |
| DVB-8 | `Section 6.1.1`: @audioSamplingRate SHOULD be common for all audio Representations within an Adapation Set | done | MPD-DVB-AUDIO-SIMILAR | upstream tests an empty array and always fails |
| DVB-9 | `Section 6.1.1`: AudioChannelConfiguration SHOULD be common for all audio Representations within an Adapation Set | done | MPD-DVB-AUDIO-SIMILAR |  |
| DVB-10 | `Section 6.6.3`: An audio fallback set shall have @value equal to the @id of the base set | done | MPD-DVB-FALLBACK |  |
| DVB-11 | `Section 11.3.0`: Audio should be 20% or less of the total stream bandwith | done | MPD-DVB-AUDIO-SHARE | dashcheck reports the combination with the largest audio share per Period, upstream every combination |
| DVB-12 | `Codec Information`: The codec should be supported by the specification | done | MPD-DVB-CODEC | the codecs of TS 103 285 V1.4.1, including VVC, AVS3, DTS-UHD and MPEG-H, which upstream lacks (bug 51) |
| DVB-13 | `Section 8.3`: ContentProtection descriptor(s) shall be placed at the AdaptationSet level | done | MPD-DVB-CP |  |
| DVB-14 | `Section 4.5`: The MPD size SHALL NOT exceed 256 Kbytes | done | MPD-DVB-LIMITS | dashcheck measures the MPD as received; upstream measures the XLink-resolved MPD |
| DVB-15 | `Section 4.5`: The MPD has a maximum of 64 periods | done | MPD-DVB-LIMITS |  |
| DVB-16 | `Section 4.5`: Each Period has a maximum of 16 Adaptation Sets | done | MPD-DVB-LIMITS |  |
| DVB-17 | `Section 4.5`: Each Adaptation Set has a maximum of 16 Representations | done | MPD-DVB-LIMITS |  |
| DVB-18 | `Section 9.1.2.1`: Events SHALL have @presentationTime set | done | MPD-DVB-EVENT |  |
| DVB-19 | `Section 9.1.2.1`: The event data shall be valid XML, either escaped or place in a CDATA section | done | MPD-DVB-EVENT |  |
| DVB-20 | `Section 10.12.3.3`: Reporting URL is required to be an absolute HTTP(S) url | done | MPD-DVB-REPORTING |  |
| DVB-21 | `Section 10.12.3.4`: Probability - if present - is required to be an integer between 1 and 1000 | done | MPD-DVB-REPORTING |  |
| DVB-22 | `Section 4.2.2`: 'The Period.SegmentList element SHALL NOT be present' | done | MPD-DVB-PERIOD |  |
| DVB-23 | `Section 4.2.2`: 'The Period.SegmentTemplate element SHALL NOT be present' | done | MPD-DVB-PERIOD |  |
| DVB-24 | `Section 4.2.2`: Each Period element shall conform to either 4.2.4 or 4.2.6 | done | MPD-DVB-PERIOD | dashcheck uses the effective @profiles of the Representations |
| DVB-25 | `Section 4.2.2`: At least one Adaptation Set SHALL cary a 'main' role | done | MPD-DVB-MAINROLE |  |
| DVB-26 | `Section 6.1.2`: At least one Adaptation Set SHALL cary a 'main' role | done | MPD-DVB-MAINROLE |  |
| DVB-27 | `Section 11.1`: Each Representation @profile SHALL include at least one of those in section 4.1 | done | MPD-DVB-PROFILE |  |
| DVB-28 | `Section 11.1`: Each representation @profile SHALL include either section 4.2.5 or section 4.2.8 | done | MPD-DVB-PROFILE |  |
| DVB-29 | `Section 11.1`: Each representation shall have only one segment | done | MPD-DVB-PROFILE |  |
| DVB-30 | `Codec Information`: The resolution of the stream should conform to table 10.3 | done | MPD-DVB-RESOLUTION |  |
| DVB-31 | `Section 9.1.9`: The @value attribute SHOULD be absent | done | MPD-DVB-SCTE-VALUE |  |
| DVB-32 | `Section 9.1.9`: The @presentationTimeOffset attribute MAY be present | n/a |  | a permissive (MAY) statement; upstream wrongly fails when the attribute is absent |
| DVB-33 | `Section 9.1.9`: The @timeScale attribute shall be set to an integer multiple of the video framerate | done | MPD-DVB-SCTE | upstream truncates fractional frame rates |
| DVB-34 | `Section 9.1.9`: The @presentationTime of an event SHALL be set to the splice time of the enclosed marker | partial | MPD-DVB-SCTE, MPD-DVB-SCTE-BINARY | like upstream, dashcheck checks that @presentationTime is present, and that the event carries a base64 splice_info_section in Signal/Binary; the splice time is an MPEG-2 TS PTS that wraps every 26.5 hours, and the MPD does not give its relation to the media timeline, so it cannot be compared with @presentationTime |
| DVB-35 | `Section 9.1.9`: The @duration of an event SHALL be defined [..] and SHALL NOT be 0xFFFFFFFF | done | MPD-DVB-SCTE | upstream compares with the string 0xFFFFFFFF and never detects 4294967295 |
| DVB-36 | `Section 9.1.9`: The @messageData of an event SHALL be absent | done | MPD-DVB-SCTE |  |
| DVB-37 | `Section 9.1.9`: The @contentEncoding of an event is absent | done | MPD-DVB-SCTE |  |
| DVB-38 | `Section 7.1.1`: The @codecs attribute shall begin with 'stpp' to indicate the use of XML subtitles | done | MPD-DVB-SUBTITLE |  |
| DVB-39 | `Bitrate information`: Terminal which does not support UHD video (max 12 Mbit/s) | done | MPD-DVB-TLS-BITRATE | reported once per Period for the highest total bandwidth |
| DVB-40 | `Bitrate information`: Terminal which does not support UHD, but not UHD HFR video (max 39 Mbit/s) | done | MPD-DVB-TLS-BITRATE | reported once per Period for the highest total bandwidth |
| DVB-41 | `Bitrate information`: Terminal which does supports UHD HFR video (max 51 Mbit/s) | done | MPD-DVB-TLS-BITRATE | reported once per Period for the highest total bandwidth |
| DVB-42 | `Section 4.7.2`: The MPD SHOULD contain at least one UTCTiming element with the @schemeIdUri attribute set to [...] | done | MPD-DVB-UTCTIMING |  |
| DVB-43 | `Section 4.8.3`: MPD@timeShiftBufferDepth shall not change | done | MPD-DVB-UPDATE | dashcheck fetches a dynamic MPD again (-updates, default once) and compares the versions |
| DVB-44 | `Section 4.8.3`: MPD@availabilityStartTime shall not change | done | MPD-DVB-UPDATE |  |
| DVB-45 | `Section 4.8.3`: MPD@maxSegmentDuration shall not change to a larger duration | done | MPD-DVB-UPDATE | upstream compares the durations as strings (bug 44) |
| DVB-46 | `Section 4.8.3`: @id shall not change for a corresponding Period | done | MPD-UPDATE-ID | a Period starting at the same time under another @id; also an ISO/IEC 23009-1 5.4.1 rule for all dynamic MPDs |
| DVB-47 | `Section 4.8.3`: @start shall not change for a corresponding Period | done | MPD-DVB-UPDATE |  |
| DVB-48 | `Section 4.8.3`: @AssetIdentifier shall not change for a corresponding Period | done | MPD-DVB-UPDATE | upstream reads AssetIdentifier as an attribute and never fails (bug 44) |
| DVB-49 | `Section 4.8.3`: @id shall not change for a corresponding AdaptationSet | done | MPD-UPDATE-ID | AdaptationSets correspond through their Representations; upstream pairs them by @id and cannot see a change (bug 44) |
| DVB-50 | `Section 4.8.3`: @id shall not change for a corresponding Representation | done | MPD-DVB-UPDATE | a changed Representation@id shows as a removed Representation (TS 103 285 4.8.1 allows only additions) |
| DVB-51 | `Section 4.8.3`: Role Element shall not change for a corresponding AdaptationSet or corresponding Representation | done | MPD-DVB-UPDATE |  |
| DVB-52 | `Section 4.8.3`: AudioConfiguration Element shall not change for a corresponding AdaptationSet … | done | MPD-DVB-UPDATE | the AudioChannelConfiguration element; upstream looks for AudioConfiguration and never fails (bug 44) |
| DVB-53 | `Section 4.8.3`: @contentType shall not change for a corresponding AdaptationSet or corresponding Representation | done | MPD-UPDATE-REP | compares the effective value of each Representation, so moving the attribute between levels is not a change |
| DVB-54 | `Section 4.8.3`: @codecs shall not change for a corresponding AdaptationSet or corresponding Representation | done | MPD-UPDATE-REP | compares the effective value of each Representation |
| DVB-55 | `Section 4.8.3`: @lang shall not change for a corresponding AdaptationSet or corresponding Representation | done | MPD-DVB-UPDATE |  |
| DVB-56 | `Section 7.2.1.1`: A fontdownload descriptor SHALL only be placed in AdaptationSets containing subtitles | done | MPD-DVB-FONT |  |
| DVB-57 | `Section 4.4`: Elements and attributes are expected to be present [..] to enable selection and switching. | done | MPD-DVB-VIDEO |  |
| DVB-58 | `Section 4.4`: @scanType SHALL be present [..] if interlaced pictures are used in any [..representation] | done | MPD-DVB-VIDEO |  |
| DVB-59 | `Section 11.2.2`: The frame rates used should be integer multiples of eachother | done | MPD-DVB-FRAMERATES |  |
| DVB-60 | `5.4.6`: The value of the signalled ColourPrimaries Property in the MPD SHALL match … | todo |  | AVS3: mp4ff does not parse AVS3 configuration records |
| DVB-61 | `5.4.6`: The value of the signalled MatrixCoefficients Property in the MPD SHALL match … | todo |  | AVS3: mp4ff does not parse AVS3 configuration records |
| DVB-62 | `5.4.6`: The value of the signalled TransferCharacteristics Property in the MPD SHALL match … | todo |  | AVS3: mp4ff does not parse AVS3 configuration records |
| DVB-63 | `5.4.7`: Use of HLG10 within in an AdapationSet shall be signalled according to this section | todo |  | AVS3: mp4ff does not parse AVS3 configuration records |
| DVB-64 | `5.4.8`: Use of PQ10 within in an AdapationSet shall be signalled according to this section | todo |  | AVS3: mp4ff does not parse AVS3 configuration records |
| DVB-65 | `5.2.2`: The value of the @codecs attribute shall be set in accordance with ISO/IEC 14496-15 | done | SEG-DVB-HEVC-CODECS | @codecs may signal more capability than needed (5.2.2), so dashcheck checks that the media meets the signalled profile, tier, level and constraints; upstream requires an exact prefix (bug 45) |
| DVB-66 | `5.2.2`: The value of the signalled ColourPrimaries Property in the MPD SHALL match … | done | SEG-DVB-COLOUR | compared with the VUI, as 5.2.5 says; upstream uses the 'colr' box (bug 45) |
| DVB-67 | `5.2.2`: The value of the signalled MatrixCoefficients Property in the MPD SHALL match … | done | SEG-DVB-COLOUR | compared with the VUI (bug 45) |
| DVB-68 | `5.2.2`: The value of the signalled TransferCharacteristics Property in the MPD SHALL match … | done | SEG-DVB-COLOUR | compared with the VUI; the SupplementalProperty may differ (bug 45) |
| DVB-69 | `5.2.6`: Use of HLG10 within in an AdapationSet shall be signalled according to this section | done | SEG-DVB-HLG10 | HLG10 found from VUI 9/14/9 and the alternative_transfer_characteristics SEI (hvcC or first sample) or the SupplementalProperty; a warning because MPDs for BT.2020 players may omit signalling; upstream never fails (bug 45) |
| DVB-70 | `5.2.7`: Use of PQ10 within in an AdapationSet shall be signalled according to this section | done | SEG-DVB-PQ10 | PQ10 found from VUI transfer_characteristics 16; upstream never fails (bug 45) |
| DVB-71 | `Section 4.3`: The 'moof' box shall contain only one 'traf' box | done | SEG-DVB-TRAF |  |
| DVB-72 | `Section 4.3`: The segment shall contain only one 'sidx' box | done | SEG-DVB-SIDX | checked on the sidx boxes in @indexRange |
| DVB-73 | `5.1.2`: Content SHOULD be offered using 'avc3' or 'avc4' | done | SEG-DVB-AVC3 |  |
| DVB-74 | `5.1.2`: All information necessary to decode any Segment shall be provided in the init segment | done | SEG-DVB-PARAMSETS |  |
| DVB-75 | `DRM`: Content Protection SHALL contain a ContentProtectionDescriptor | done | MPD-DVB-CP |  |
| DVB-76 | `DRM`: PSSH Boxes SHOULD exist in either the MPD or the Initialization Segment | done | SEG-DVB-PSSH | reported as info: TS 103 285 V1.4.1 8.5.0 lets the initialization and/or media segments carry pssh and has no SHOULD for the MPD or initialization segment (bug 43) |
| DVB-77 | `DRM`: MPD DRM System information | n/a |  | an informational listing of the DRM systems; upstream always passes |
| DVB-78 | `DRM`: The scheme SHOULD be set to 'cenc' | done | SEG-DVB-CENC, SEG-DVB-CENC-FALLBACK | TS 103 285 V1.3.1 and V1.4.1 8.6.0 allow cenc or cbcs (cbcs not with the 2014 profile) and require a clear or cenc fallback per media type when AVC is used; upstream asks for cenc only (bug 43) |
| DVB-79 | `Section 10.5.2.3`: Continous adaptation sets shall be signalled accordingly | done | MPD-DVB-CONTINUITY | also checks the association of 10.5.2.2 (AssetIdentifier, @lang, @contentType, @par, Role, Accessibility, Viewpoint, audio Table 4 values), which upstream leaves as a TODO, and accepts the MPEG period-continuity URN too |
| DVB-80 | `Section 10.5.2.3`: All Representations in the Adaptation Set in the first Period shall share the same value EPT1 | done | SEG-DVB-CONTINUITY-EPT | compares the first segment of each Representation, so only for static MPDs where the first segment of the Period is checked |
| DVB-81 | `Section 10.5.2.3`: All Representations in the Adaptation Set in a subsequent Period shall share the same value EPT2 | done | SEG-DVB-CONTINUITY-EPT | as DVB-80 |
| DVB-82 | `6.1.1`: @mimeType SHALL match the mimetype derived from the segments | done | XCHK-HANDLER | dashcheck compares the track handler with @contentType or @mimeType |
| DVB-83 | `6.1.1`: @audioSamplingRate SHALL match the sample rate derived from the segments | done | XCHK-SAMPLERATE | dashcheck reports a warning |
| DVB-84 | `6.1.1`: AudioChannelConfiguration SHALL match the configuration derived from the segments | done | SEG-DVB-CHANNELS | only with the 23003:3 scheme; upstream fails with other schemes |
| DVB-85 | `5.x / 6.x`: For each representation, @codecs SHALL be match the codec derived from the segments | done | XCHK-CODECS, XCHK-CODECS-DETAIL |  |
| DVB-86 | `4.5`: Each subsegment shall have a duration of not more than 15 seconds | done | SEG-DVB-DURATION |  |
| DVB-87 | `4.5`: Each subsegment shall be at least 1 second | done | SEG-DVB-DURATION | the last segment of a Period may be shorter |
| DVB-88 | The codec should be supported by the specification | done | SEG-DVB-CODEC | as DVB-12 (bug 51) |
| DVB-89 | Profile used for AVC must be suported by the specification | done | SEG-DVB-AVC | TS 103 285 5.1.1 asks each Period to include video for avc_hd_50_level40 or avc_hd_60_level40 (Main or High up to level 4.0, TS 101 154 table L.1); upstream requires every Representation to be High profile (bug 38). A warning, since the spec allows other device targets |
| DVB-90 | Level used for AVC must be suported by the specification | done | SEG-DVB-AVC | checked per Period with DVB-89; upstream allows only levels 3.0, 3.2, 4.0 and 4.1 in every Representation (bug 38) |
| DVB-91 | Tier used for HEVC must be suported by the specification | done | SEG-DVB-HEVC | TS 103 285 5.2.3: at least one Representation per Period fits an HEVC conformance point of TS 101 154 table L.1 (Main tier); upstream rejects high tier in every Representation (bug 41) |
| DVB-92 | Bit depth used for HEVC must be suported by the specification | done | SEG-DVB-HEVC-SPS | per TS 101 154 L.3.3.2: Main profile 8-bit, Main 10 8 or 10 bits with equal chroma depth, progressive frames; upstream accepts 8 or 10 bits for any profile (bug 41) |
| DVB-93 | Profile used for HEVC must be suported by the specification | done | SEG-DVB-HEVC | checked per Period with DVB-91; upstream rejects Main profile above 1080p although hevc_uhd allows it (bug 41) |
| DVB-94 | Level used for HEVC must be suported by the specification | done | SEG-DVB-HEVC | checked per Period with DVB-91; levels 4.1 up to 1080p, 5.2 up to 2160p and 6.1 up to 4320p; upstream rejects 5.2 and 6.1 (bug 41) |
| DVB-95 | `5.1.2`: Segments shall start with SAP types 1 or 2 | done | SEG-DVB-SAP, SEG-SAP |  |
| DVB-96 | `5.2.4`: For HEVC streams, SEI messages are placed according to ETSI TS 101 154 - L.3.3.8 | done | SEG-DVB-SEI | all samples of the checked segments, and the hvcC array order of 5.2.4; upstream skips hvc1 (bug 46) |
| DVB-97 | `5.2.4`: For VVC streams, SEI messages are placed according to ETSI TS 101 154 - 5.15.1 | done | SEG-DVB-VVC-SEI | with the vvc package of mp4ff; upstream allows the wrong NAL unit types before prefix SEI and ignores the picture header (bug 46) |
| DVB-98 | `Subtitles`: For subtitle media, handler SHALL be 'subt' | done | SEG-DVB-SUBTITLE | upstream only checks @codecs equal to stpp |
| DVB-99 | `Subtitles`: Sample entry shall be 'stpp' | done | SEG-DVB-SUBTITLE | upstream only checks @codecs equal to stpp |
| DVB-100 | `Subtitles`: Sample entry shall countain a namespace | done | SEG-DVB-SUBTITLE | upstream only checks @codecs equal to stpp |
| DVB-101 | `4.3`: Initialization segment for Representations with an Adaptation Set SHALL have the same … | done | SEG-DVB-SWITCHABLE |  |
| DVB-102 | `4.3`: Initialization segment for Representations with an Adaptation Set SHALL have the same … | done | SEG-DVB-SWITCHABLE |  |
| DVB-103 | `8.3`: Initialization segment for Representations with an Adaptation Set SHALL have the same … | done | SEG-DVB-SWITCHABLE |  |

## Module: HbbTV

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| HBBTV-1 | `Unknown`: Inherit DVB legacy checks | n/a |  | marker in the upstream tool: HbbTV inherits the DVB checks |
| HBBTV-2 | `Codec Information`: The codec should be supported by the specification | done | MPD-HBBTV-CODECS | ETSI TS 102 796 V1.4.1 E.2.1 asks for at least one AdaptationSet in a 7.3.1 codec (AVC, HEVC, HE-AAC, E-AC-3); upstream warns for each other codec, including HEVC (bug 49) |
| HBBTV-3 | `Unknown`: Inherit DVB legacy checks | n/a |  | marker in the upstream tool: HbbTV inherits the DVB checks |
| HBBTV-4 | `Unknown`: Inherit DVB legacy checks | n/a |  | marker in the upstream tool: HbbTV inherits the DVB checks |
| HBBTV-5 | `DRM`: Inherit DVB legacy checks | n/a |  | marker in the upstream tool: HbbTV inherits the DVB checks |
| HBBTV-6 | `DRM`: Inherit DVB legacy checks | n/a |  | marker in the upstream tool: HbbTV inherits the DVB checks |
| HBBTV-7 | The codec should be supported by the specification | done | SEG-DVB-CODEC | E.4.1 treats the HbbTV 2012 profile as DVB-DASH 2014 live, so dashcheck applies the DVB segment rules |
| HBBTV-8 | Profile used for AVC must be suported by the specification | done | SEG-DVB-AVC | the DVB-DASH conformance point check per Period; upstream requires Main or High in every Representation (bug 49) |
| HBBTV-9 | Level used for AVC must be suported by the specification | done | SEG-DVB-AVC | as HBBTV-8; upstream allows only levels 3.0 to 4.0 in every Representation (bug 49) |
| HBBTV-10 | All info necessary to decode any audio segment shall be provided in the Initialization segment | done | SEG-HBBTV-AUDIO-CONFIG | requires the decoder configuration box (esds AudioSpecificConfig, dac3, dec3, dac4); the clause is from HbbTV 1.5 (TS 102 796 V1.2.1), which is not in the reference library |

## Module: CTA WAVE

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| WAVE-1 | `4.3.1`: Each WAVE audio Media profile SHALL conform to normative ref. listed in Table 1 | done | SEG-CMAF-MEDIAPROFILE | the limits of signalled media profile brands are checked by the CMAF rule; cud1 now also allows BT.2020 (bug 25) |
| WAVE-2 | `4.1`: Wave content SHALL include one or more switch sets conforming to at least one approved CMAF profile | done | SEG-WAVE-PROFILE | per selection set (media type and @lang in a Period); a track conforms by its ftyp brand, @containerProfiles or its parameters, since the brand is only a SHOULD in CMAF; for MPDs with a urn:cta:wave: profile |
| WAVE-3 | `5 `: If an audio track is included, the conforming (presentation will at least … | done | SEG-CMAF-PRESENTATIONPROFILE | CTA-5001-F 5 requires AAC Core only for the CMFHD presentation profiles, which the CMAF rule checks; upstream requires it in every audio AdaptationSet (bug 47) |
| WAVE-4 | `7.2.2`: Sequential sets SHALL conform to the same CMAF profile | done | SEG-WAVE-SPLICE | information: the Baseline Splice Constraint Profile is optional and not signalled (CTA-5001-F 6.1); upstream fails (bug 47) |
| WAVE-5 | `7.2.2`: Wave content SHALL contain one CENC scheme per program | done | SEG-WAVE-SCHEME | a warning: one scheme per presentation for CMAF presentation profiles (5) and per program for the splice profile (7.2.2) |
| WAVE-6 | `7.2.2 - Presentation Splicing`: Sample entries in Sequential Switching Sets Shall not change at Splice points | done | SEG-WAVE-SPLICE | information, as WAVE-4 |
| WAVE-7 | `7.2.2 - Presentation Splicing`: Framerate in Sequential Switching Sets Shall not change at Splice points | done | SEG-WAVE-SPLICE | table 5 only requires the media profile limits; dashcheck reports a different frame rate family as the table 6 recommendation (bug 47) |
| WAVE-8 | `7.2.2 - Presentation Splicing`: Channel count in Sequential Switching Sets Shall not change at Splice points | n/a |  | table 5 only requires the media profile limits (SEG-CMAF-MEDIAPROFILE); the table 6 channel recommendation is not testable (bug 47) |
| WAVE-9 | `7.2.2 - Presentation Splicing`: Picture Aspect Ration (PAR) in Sequential Switching Sets Shall not change at Splice points | done | SEG-WAVE-SPLICE | reported as the table 6 recommendation (bug 47) |
| WAVE-10 | `7.2.2 - Presentation Splicing`: Track Ids can change at Splice points | n/a |  | a permissive statement: track IDs can change at splice points |
| WAVE-11 | `7.2.2 - Presentation Splicing`: Timescale can change at Splice points | n/a |  | a permissive statement: the timescale can change at splice points |
| WAVE-12 | `7.2.2 - Presentation Splicing`: Encryption scheme can change at Splice points | n/a |  | a permissive statement: the encryption can change at splice points (one scheme per program is WAVE-5) |
| WAVE-13 | `7.2.2 - Presentation Splicing`: Default KID can change at Splice points | n/a |  | a permissive statement: the default KID can change at splice points |
| WAVE-14 | `4.4.1`: Each WAVE Subtitle Media profile SHALL conform to normative ref. listed in Table 1 | done | SEG-CMAF-SUBTITLEPROFILE | checked by the CMAF subtitle profile rule |
| WAVE-15 | `4.1`: Wave content SHALL include one or more switch sets conforming to at least one approved CMAF profile | done | SEG-WAVE-PROFILE | as WAVE-2 |
| WAVE-16 | `5 `: If a subtitle track is included, the conforming (presentation will at least … | done | SEG-CMAF-PRESENTATIONPROFILE | as WAVE-3, for IMSC1 Text (bug 47) |
| WAVE-17 | `7.2.2`: Sequential sets SHALL conform to the same CMAF profile | done | SEG-WAVE-SPLICE | as WAVE-4 |
| WAVE-18 | `4.2.1`: Each WAVE video Media profile SHALL conform to normative ref. listed in Table 1 | done | SEG-CMAF-MEDIAPROFILE | as WAVE-1 |
| WAVE-19 | `4.1`: Wave content SHALL include one or more switch sets conforming to at least one approved CMAF profile | done | SEG-WAVE-PROFILE | as WAVE-2 |
| WAVE-20 | `5 `: If a video tracks is included, the conforming (presentation will at least … | done | SEG-CMAF-PRESENTATIONPROFILE | as WAVE-3, for AVC HD (bug 47) |
| WAVE-21 | `7.2.2`: Sequential sets SHALL conform to the same CMAF profile | done | SEG-WAVE-SPLICE | as WAVE-4 |

## Module: WAVE HLS interop

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| WAVEHLS-1 | `4.1.2 - Basic On-Demand and Live Streaming`: CMAF Track Files [..] SHALL contain a single 'sidx' following the CMAF header … | done | SEG-HLS-SIDX | for SegmentBase track files, which are the CMAF track files delivered to clients; upstream requires a sidx in every media segment (bug 48). Checks of CTA-5005 run with -hls-interop |
| WAVEHLS-2 | `4.1.2 -  Basic On-Demand and Live Streaming`: The Average Bitrate of a CMAF Fragment [..] SHOULD be within 10% of the Average Bitrate … | done | SEG-HLS-BITRATE | on demand, from the sidx of SegmentBase track files, or from the sizes of all segments when every segment of a SegmentTemplate or SegmentList Representation is checked (-n -1 or few segments); live, when the checked segments cover an hour, or the track from its start (-n -1 with a long time-shift buffer), each fragment at most 110% of the average of the newest hour; segments count as fragments; upstream applies the on-demand rule to the downloaded segments of any MPD (bug 54) |
| WAVEHLS-3 | The common encryption `cbcs` scheme SHALL be used for encryption | done | SEG-HLS-ENCRYPTION |  |
| WAVEHLS-4 | Constant 16-byte Initialization Vectors SHALL be used | done | SEG-HLS-ENCRYPTION | tenc default_Per_Sample_IV_Size 0 and a 16-byte constant IV |
| WAVEHLS-5 | An alternative using the `cenc` scheme MAY be produced | done | SEG-HLS-CBCS-VARIANT | per Period and media type, a cenc AdaptationSet needs a cbcs one (4.3.3); upstream leaves this as a TODO |
| WAVEHLS-6 | Constant 8-byte Initialization Vectors SHALL be used for 'cenc' encrypted material | done | SEG-HLS-ENCRYPTION | 8-byte IVs as CMAF 8.2.3.1 says; upstream requires a constant 8-byte IV, which fails normal cenc (bug 48) |
| WAVEHLS-7 | Sample auxiliary information, if present, SHALL be addressed by [.. a 'saio' box] | done | SEG-HLS-ENCRYPTION | a saio box wherever senc or saiz is present |
| WAVEHLS-8 | Video components SHALL be encrypted with a 1:9 pattern | done | SEG-HLS-ENCRYPTION | 1:9 for cbcs video; upstream also tests cenc tracks (bug 48) |
| WAVEHLS-9 | Audio components SHALL be encrypted with a 10:0 pattern | done | SEG-HLS-ENCRYPTION | 10:0 for cbcs audio; 0:0 is accepted as the equivalent whole-block full-sample encryption of ISO/IEC 23001-7 10.4.1 |
| WAVEHLS-10 | `4.6.2 - Rotation of Encryption Keys`: CMAF Segments [.. with differing encryption keys] SHALL provide an 'sbgp' box [..] | done | SEG-HLS-KEY-ROTATION | a seig sample group with another KID needs a seig sbgp box |
| WAVEHLS-11 | `4.6.2 - Rotation of Encryption Keys`: CMAF Segments [.. with differing encryption keys] SHALL provide a 'pssh' box [..] | done | SEG-HLS-KEY-ROTATION | pssh boxes in the segment when the CMAF header has them |
| WAVEHLS-12 | `4.3.2 - Encrypted Media Presentations`: Any individual CMAF Segment SHALL have a single encryption key | done | SEG-HLS-SEGMENT-KEY | compares the KID of each fragment (seig or tenc); upstream counts pssh KIDs (bug 48) |
| WAVEHLS-13 | `4.3.2 - Encrypted Media Presentations`: Any individual CMAF Segment SHALL have a single Initialization Vector | done | SEG-HLS-SEGMENT-KEY | compares the constant IV of each fragment; upstream rejects per-sample IVs (bug 48) |
| WAVEHLS-14 | `4.4.2 - Presentation Splicing`: CMAF Fragment boundaries SHALL be created at all splice points. | done | SEG-HLS-SPLICE | the last segment of each Period that another follows must end at the Period end within one sample duration (static MPDs, with -hls-interop) |
| WAVEHLS-15 | `4.4.2 - Presentation Splicing`: CMAF Fragment [duration difference] SHALL be within one ISOBMFF sample duration | done | SEG-HLS-SPLICE | each media type ends within one sample duration of the splice point, which keeps them within that of each other |
| WAVEHLS-16 | `4.1.2 - Basic On-Demand and Live Streaming`: Text components SHALL be packaged in ISMC1, ISMC1.1 or WebVTT Tracks | done | SEG-HLS-TEXT | IMSC1/IMSC1.1 Text from @codecs, or wvtt |
| WAVEHLS-17 | `4.5.2 - Carriage of Timed Event Data`: All 'emsg' boxes inserted [..] after the start of the first CMAF chunk … | done | SEG-EMSG-REPEAT | as LL-28, with -hls-interop |
| WAVEHLS-18 | Track roles SHALL be stored in one or more 'kind' boxes [..] | done | SEG-HLS-ROLES | every AdaptationSet Role needs a kind box with the same scheme and value |
| WAVEHLS-19 | Track roles SHALL be representaed by the DASH Role scheme when possible | done | SEG-HLS-ROLES | kind boxes without the DASH Role scheme |

## Module: Dolby

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| DOLBY-1 | `E.6.3 [13197]`: Bitstream version must match between TOC and DAC4 | done | SEG-AC4-TOC | the TOC of the first AC-4 frame of each checked segment against dac4 (unencrypted tracks) |
| DOLBY-2 | `E.6.4 [14309]`: FS index must match between TOC and DAC4 | done | SEG-AC4-TOC |  |
| DOLBY-3 | `E.6.5 [14209]`: Framerate index version must match between TOC and DAC4 | done | SEG-AC4-TOC |  |
| DOLBY-4 | `E.6.6 [14215]`: Number of presentations must match between TOC and DAC4 | done | SEG-AC4-TOC |  |
| DOLBY-5 | `E.6.7 [14221]`: Short program id must match between TOC and DAC4 | done | SEG-AC4-TOC | compared when the TOC has bitstream_version 2 or more, where short_program_id exists |
