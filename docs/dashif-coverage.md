# DASH-IF Conformance tool coverage

<!-- Generated from dashif-checks.json by `go test ./internal/coverage -update`. Do not edit. -->

Checks of the [DASH-IF Conformance tool](https://github.com/Dash-Industry-Forum/DASH-IF-Conformance) at commit e311ec6 (2026-10-16), and the dashcheck rules that cover them. Statuses: **done** is covered by a dashcheck rule, **partial** is partly covered, **todo** is not ported yet, and **n/a** will not be ported (see the note).

| Group | Checks | Done | Partial | Todo | n/a |
|---|---:|---:|---:|---:|---:|
| Schematron: schematron.sch | 123 | 47 | 3 | 73 | 0 |
| Schematron: dvb-dash.sch | 13 | 0 | 0 | 13 | 0 |
| Schematron: ac4-generic.sch | 17 | 0 | 0 | 17 | 0 |
| Module: Common | 5 | 2 | 0 | 2 | 1 |
| Module: IOP | 15 | 2 | 0 | 13 | 0 |
| Module: CMAF | 23 | 1 | 0 | 22 | 0 |
| Module: Low Latency | 54 | 5 | 3 | 46 | 0 |
| Module: DVB | 103 | 2 | 3 | 98 | 0 |
| Module: HbbTV | 10 | 0 | 0 | 5 | 5 |
| Module: CTA WAVE | 21 | 0 | 0 | 21 | 0 |
| Module: WAVE HLS interop | 19 | 0 | 0 | 19 | 0 |
| Module: Dolby | 5 | 0 | 0 | 5 | 0 |
| **Total** | **408** | **59** | **9** | **334** | **6** |

## Schematron: schematron.sch

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| SCH-1 | `dash:MPD`: If MPD is of type "dynamic" availabilityStartTime shall be defined. | done | MPD-DYN-AST |  |
| SCH-2 | `dash:MPD`: If MPD is of type "dynamic" publishTime shall be defined. | done | MPD-DYN-PUBLISHTIME |  |
| SCH-3 | `dash:MPD`: If mediaPresentationDuration is not defined for the MPD minimumUpdatePeriod shall be defined or vice versa. | done | MPD-DURATION-MISSING | upstream ignores the last Period@duration; dashcheck uses the condition of SCH-6 |
| SCH-4 | `dash:MPD`: At least one unknown profile string found. Known list in schematron file. | done | MPD-PROFILE-UNKNOWN | dashcheck reports info, upstream an error; dashcheck also knows DVB interop, CMAF, low-latency and newer MPEG profiles |
| SCH-5 | `dash:MPD`: For On-Demand profile, the MPD @type shall be "static". | todo |  |  |
| SCH-6 | `dash:MPD`: If minimumUpdatePeriod is not present and the last period does not include the duration attribute the mediaPresentationDuration must be present. | done | MPD-DURATION-MISSING |  |
| SCH-7 | `dash:MPD`: For ISO BMFF extended live profile, MPD@type shall be static. | todo |  |  |
| SCH-8 | `dash:MPD`: For ISO BMFF extended on-demand profile, MPD@type shall be static. | todo |  |  |
| SCH-9 | `dash:Period`: If bitstreamSwitching is set to true all bitstreamSwitching declarations for AdaptationSet within this Period shall not be set to false. | done | MPD-BITSTREAMSWITCHING |  |
| SCH-10 | `dash:Period`: The id of each Period shall be unique. | done | MPD-PERIOD-ID |  |
| SCH-11 | `dash:Period`: At most one of SegmentBase, SegmentTemplate and SegmentList shall be defined in Period. | done | MPD-SEGINFO-MULTIPLE |  |
| SCH-12 | `dash:Period`: If the MPD is dynamic the Period element shall have an id. | done | MPD-PERIOD-ID |  |
| SCH-13 | `dash:Period`: At least one BaseURL, SegmentTemplate or SegmentList shall be defined in Period, AdaptationSet or Representation. | done | MPD-SEGINFO-NONE |  |
| SCH-14 | `dash:Period`: If the duration attribute is set to zero, there should only be a single AdaptationSet present. | done | MPD-PERIOD-ZERO | upstream fails for any Period with @duration 0 and an AdaptationSet; dashcheck follows the message and warns when there is more than one |
| SCH-15 | `dash:Period`: Neither the Period.SegmentList element nor the Period.SegmentTemplate element shall be present for On-Demand profile, violated here. | todo |  |  |
| SCH-16 | `dash:Period`: For ISO BMFF extended live profile, Period.SegmentList shall not be present. | todo |  |  |
| SCH-17 | `dash:Period`: For ISO BMFF extended live profile: "If a Period contains multiple Adaptation Sets with value of the @contentType="video" then at least one Adaptation Set shall contain a Role el-ement $&lt;$Role scheme="urn:mpeg:dash:role:2011" value="main"&gt;" | todo |  |  |
| SCH-18 | `dash:Period`: For ISO BMFF extended on-demand profile, neither the Period.SegmentList element nor the Period.SegmentTemplate element shall be present for On-Demand profile, violated here. | todo |  |  |
| SCH-19 | `dash:Period`: For ISO BMFF extended on-demand profile: "If a Period contains multiple Adaptation Sets with value of the @contentType="video" then at least one Adaptation Set shall contain a Role el-ement $&lt;$Role scheme="urn:mpeg:dash:role:2011" value="main"&gt;" | todo |  |  |
| SCH-20 | `dash:AdaptationSet`: The id of each AdaptationSet within a Period shall be unique. | done | MPD-AS-ID |  |
| SCH-21 | `dash:AdaptationSet`: Attributes from the AdaptationSet shall not be repeated in the descendanding ContentComponent elements. | done | MPD-CONTENTCOMPONENT |  |
| SCH-22 | `dash:AdaptationSet`: Common attributes for AdaptationSet and Representation shall either be in one of the elements but not in both. | done | MPD-COMMON-BOTH | dashcheck reports a warning, upstream an error |
| SCH-23 | `dash:AdaptationSet`: Each minimum value (minWidth, minHeight, minBandwidth) shall be larger than the maximum value. | done | MPD-AS-MINMAX |  |
| SCH-24 | `dash:AdaptationSet`: The value of the bandwidth attribute shall be in the range defined by the AdaptationSet. | done | MPD-AS-RANGE |  |
| SCH-25 | `dash:AdaptationSet`: The value of the width attribute shall be in the range defined by the AdaptationSet. | done | MPD-AS-RANGE | dashcheck also checks @minWidth |
| SCH-26 | `dash:AdaptationSet`: The value of the height attribute shall be in the range defined by the AdaptationSet. | done | MPD-AS-RANGE | dashcheck also checks @minHeight |
| SCH-27 | `dash:AdaptationSet`: An AdaptationSet shall have at least one Representation element. | done | MPD-AS-NOREP |  |
| SCH-28 | `dash:AdaptationSet`: At most one of SegmentBase, SegmentTemplate and SegmentList shall be defined in AdaptationSet. | done | MPD-SEGINFO-MULTIPLE |  |
| SCH-29 | `dash:AdaptationSet`: ISO/IEC 23009-1:2019, 5.3.3.1: The value of the frameRate attribute shall be in the range defined by the AdaptationSet. | done | MPD-AS-RANGE |  |
| SCH-30 | `dash:AdaptationSet`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - The MPD contains an attribute that is not part of the HbbTV profile', i.e., found 'subsegmentAlignment' as true | todo |  |  |
| SCH-31 | `dash:AdaptationSet`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - The MPD contains an attribute that is not part of the HbbTV profile', i.e., found 'subsegmentStartsWithSAP' as 1 or 2 | todo |  |  |
| SCH-32 | `dash:AdaptationSet`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - The MPD contains an attribute that is not part of the HbbTV profile', i.e., found 'subsegmentStartsWithSAP' as 3 while not containing more than one Representation | todo |  |  |
| SCH-33 | `dash:ContentComponent`: The id of each ContentComponent within an AdaptationSet shall be unique. | done | MPD-CONTENTCOMPONENT |  |
| SCH-34 | `dash:Representation`: Either the Representation or the containing AdaptationSet shall have the mimeType attribute. | done | MPD-MIMETYPE |  |
| SCH-35 | `dash:Representation`: For live profile, the SegmentTemplate element shall be present on at least one of the three levels, the Period level containing the Representation, the Adaptation Set containing the Representation, or on Representation level itself. | todo |  |  |
| SCH-36 | `dash:Representation`: At most one of SegmentBase, SegmentTemplate and SegmentList shall be defined in Representation. | done | MPD-SEGINFO-MULTIPLE |  |
| SCH-37 | `dash:Representation`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - The MPD contains an element that is not part of the HbbTV profile', i.e., found 'BaseURL' element | todo |  |  |
| SCH-38 | `dash:Representation`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - The MPD contains an attribute that is not part of the HbbTV profile', i.e., found 'subsegmentStartsWithSAP' as 3 with same value of mediaStreamStructureId in more than one Representation | todo |  |  |
| SCH-39 | `dash:Representation`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - For HbbTV profile, the SegmentTemplate element shall be present on at least one of the three levels, the Period level containing the Representation, the Adaptation Set containing the Representation, or on Representation level itself as it is based on live profile | todo |  |  |
| SCH-40 | `dash:SubRepresentation`: If the level attribute is defined for a SubRepresentation also the bandwidth attribute shall be defined. | done | MPD-SUBREP-BANDWIDTH |  |
| SCH-41 | `dash:SegmentTemplate`: If more than one Media Segment is present the duration attribute or SegmentTimeline element shall be present. | done | ADDR-TEMPLATE | checked on the inherited SegmentTemplate of each Representation, so templates split over levels are not flagged as upstream does |
| SCH-42 | `dash:SegmentTemplate`: Either the duration attribute or SegmentTimeline element shall be present but not both. | done | ADDR-TEMPLATE |  |
| SCH-43 | `dash:SegmentTemplate`: If indexRange is not present indexRangeExact shall not be present. | partial | MPD-INDEXRANGEEXACT | dash-mpd reads an absent @indexRangeExact as false, so only indexRangeExact="true" is detected |
| SCH-44 | `dash:SegmentTemplate`: Neither $Number$ nor the $Time$ identifier shall be included in the initialization attribute. | done | MPD-TEMPLATE-INIT |  |
| SCH-45 | `dash:SegmentTemplate`: Neither $Number$ nor the $Time$ identifier shall be included in the bitstreamSwitching attribute. | done | MPD-TEMPLATE-INIT |  |
| SCH-46 | `dash:SegmentTemplate`: Only identifiers such as $Bandwidth$, $Time$, $RepresentationID$, or $Number$ shall be used. | done | ADDR-TEMPLATE | checked when @media is expanded |
| SCH-47 | `dash:SegmentTemplate`: $RepresentationID$ shall not have a format tag. | done | ADDR-TEMPLATE |  |
| SCH-48 | `dash:SegmentList`: If more than one Media Segment is present the duration attribute or SegmentTimeline element shall be present. | done | ADDR-TEMPLATE |  |
| SCH-49 | `dash:SegmentList`: Either the duration attribute or SegmentTimeline element shall be present but not both. | done | ADDR-TEMPLATE |  |
| SCH-50 | `dash:SegmentList`: If indexRange is not present indexRangeExact shall not be present. | partial | MPD-INDEXRANGEEXACT | dash-mpd reads an absent @indexRangeExact as false, so only indexRangeExact="true" is detected |
| SCH-51 | `dash:SegmentBase`: If indexRange is not present indexRangeExact shall not be present. | partial | MPD-INDEXRANGEEXACT | dash-mpd reads an absent @indexRangeExact as false, so only indexRangeExact="true" is detected |
| SCH-52 | `dash:SegmentBase`: The timeShiftBufferDepth shall not be smaller than timeShiftBufferDepth specified in the MPD element | done | MPD-SEGBASE-TSBD | upstream tests @timeShiftbuffer (a typo), so its rule never fires |
| SCH-53 | `dash:SegmentTimeline`: The d attribute of a SegmentTimeline shall not exceed the value give by the MPD maxSegmentDuration attribute. | done | MPD-MAXSEGDUR | upstream adds the duration components without unit conversion, so PT1M counts as 1 s |
| SCH-54 | `dash:ProgramInformation`: If more than one ProgramInformation element is given each ProgramInformation element shall have a lang attribute. | done | MPD-PROGINFO-LANG |  |
| SCH-55 | `dash:ContentProtection`: The value of ContentProtection shall be the 4CC contained in the Scheme Type Box | done | MPD-CP-VALUE |  |
| SCH-56 | `dash:ContentProtection`: The value of ContentProtection shall be the 4-digit lower-case hexadecimal Representation. | done | MPD-CP-VALUE |  |
| SCH-57 | `dash:Role`: The value of Role (role) shall be caption, subtitle, main, alternate, supplementary, commentary or dub. | done | MPD-ROLE-VALUE | dashcheck accepts all role values of the 6th edition, upstream only the first seven |
| SCH-58 | `dash:Role`: The value of Role (stereoid) shall start with 'l' or 'r'. | done | MPD-ROLE-VALUE |  |
| SCH-59 | `dash:FramePacking`: The URI urn:mpeg:dash:14496:10:frame_packing_arrangement_type:2011 is used for Adaptation Sets or Representations that contain a video component that conforms to ISO/IEC 14496-10. | todo |  |  |
| SCH-60 | `dash:FramePacking`: The URI urn:mpeg:dash:13818:1:stereo_video_format_type:2011 is used for Adaptation Sets or Representations that contain a video component that conforms to ISO/IEC 13818-1. | todo |  |  |
| SCH-61 | `dash:FramePacking`: schemeIdUri for FramePacking descriptor shall be urn:mpeg:dash:14496:10:frame_packing_arrangement_type:2011 or urn:mpeg:dash:13818:1:stereo_video_format_type:2011. | todo |  |  |
| SCH-62 | `dash:FramePacking`: The value of FramePacking shall be 0 to 6 as defined in ISO/IEC 23091(all parts). | todo |  |  |
| SCH-63 | `dash:AudioChannelConfiguration`: If URI urn:mpeg:dash:outputChannelPositionList:2012 is used the value attribute shall be a space-delimited list as defined in ISO/IEC 23091(all parts). | todo |  |  |
| SCH-64 | `dash:EventStream`: If href is not present actuate shall not be present. | done | MPD-EVENTSTREAM |  |
| SCH-65 | `dash:EventStream`: schemeIdUri shall be present. | done | MPD-EVENTSTREAM |  |
| SCH-66 | `dash:Subset`: The id of each Subset shall be unique. | done | MPD-SUBSET-ID |  |
| SCH-67 | `dash:UTCTiming`: @schemeIdUri for UTCTiming is not one of the 7 different types specified. | done | MPD-UTCTIMING-SCHEME | dashcheck also warns for schemes that browsers cannot use |
| SCH-68 | `dash:SupplementalProperty`: If schemeIdUri urn:mpeg:dash:chaining:2016 is used, then value attribute shall be composed of the comma separated parameters (no comma needed if only first parameter is present). | done | MPD-CHAINING |  |
| SCH-69 | `dash:SupplementalProperty`: MPD fallback chaining shall be signaled by Supplemental Descriptor on MPD level with schemeIdUri urn:mpeg:dash:fallback:2016. | done | MPD-CHAINING |  |
| SCH-70 | `dash:SupplementalProperty`: If schemeIdUri urn:mpeg:dash:fallback:2016 is used, then value attribute shall be composed of one URL or whitespace separated URLs. | done | MPD-CHAINING |  |
| SCH-71 | `dash:Period`: When every Adaptation Set in a MPD has a SRD descriptor, at least one of this descriptor shall be a SupplementalProperty. | todo |  |  |
| SCH-72 | `dash:Period`: For a given source_id of the @value attribute, at least one of the EssentialProperty or SupplementalProperty in the containing Period shall specify the optional parameters W and H. | todo |  |  |
| SCH-73 | `dash:Period`: For a given source_id of the @value attribute, if two SRD elements (indistinctively EssentialProperty or SupplementalProperty) explicitly specify a different pair of values for the optional parameters (W,H) then all the remaining SRD element shall explicitly specify a pair of values for (W,H) too. | todo |  |  |
| SCH-74 | `dash:Period`: For a given source_id of the @value attribute, the values of x, w and W shall be such that, for each descriptor, the sum of x and w is smaller or equal to W. | todo |  |  |
| SCH-75 | `dash:Period`: For a given source_id of the @value attribute, the values of x, w and W shall be such that, for each descriptor, the sum of x and w is smaller or equal to W. | todo |  |  |
| SCH-76 | `dash:Period`: For a given source_id of the @value attribute, the values of y, h and H shall be such that, for each descriptor, the sum of y and h is smaller or equal to H. | todo |  |  |
| SCH-77 | `dash:Period`: For a given source_id of the @value attribute, the values of y, h and H shall be such that, for each descriptor, the sum of y and h is smaller or equal to H. | todo |  |  |
| SCH-78 | `dash:SupplementalProperty \| dash:EssentialProperty`: An EssentialProperty or a SupplementalProperty descriptor with @schemeIdUri equal to “urn:mpeg:dash:srd:2014” shall be the child element of an AdaptationSet or a SubRepresentation element. | todo |  |  |
| SCH-79 | `dash:SupplementalProperty \| dash:EssentialProperty`: If an EssentialProperty or a SupplementalProperty descriptor with @schemeIdUri equal to “urn:mpeg:dash:srd:2014” is present, then the @value attribute must be present. | todo |  |  |
| SCH-80 | `dash:SupplementalProperty \| dash:EssentialProperty`: If an EssentialProperty or a SupplementalProperty descriptor with @schemeIdUri equal to “urn:mpeg:dash:srd:2014” is present, then the @value attribute must contain at least the mandatory comma separated parameters, i.e. source_id, x, y, w, h. | todo |  |  |
| SCH-81 | `dash:SupplementalProperty \| dash:EssentialProperty`: If an EssentialProperty or a SupplementalProperty descriptor with @schemeIdUri equal to “urn:mpeg:dash:srd:2014” is present, then each parameter value has to match the expected type format i.e. non-negative integer in decimal representation. | todo |  |  |
| SCH-82 | `dash:SupplementalProperty \| dash:EssentialProperty`: If an EssentialProperty or a SupplementalProperty descriptor with @schemeIdUri equal to “urn:mpeg:dash:srd:2014” is present and the @value attribute contains the optional parameter W then the optional parameter H shall be present too. In addition, if the optional parameter spatial_set_id is present, then the optional parameters W and H shall be present. | todo |  |  |
| SCH-83 | `dash:MPD`: If MPD is of type "static" and if the profile contains a DASH-IF IOP profile, then the timeShiftBufferDepth shall not be defined. | todo |  |  |
| SCH-84 | `dash:MPD`: If MPD is of type "static" and if the profile contains a DASH-IF IOP profile, then the minimumUpdatePeriod shall not be defined. | todo |  |  |
| SCH-85 | `dash:MPD`: DASH-IF IOP Section 3.2.2.2: For dynamic MPD, the @profile shall include urn:mpeg:dash:profile:isoff-live:2011. | todo |  |  |
| SCH-86 | `dash:MPD`: DASH-IF IOP Section 3.10.2/3.10.3: "MPD@minBufferTime shall be present", not found. | todo |  |  |
| SCH-87 | `dash:Period`: DASH-IF IOP Section 3.2.2: "the Period.SegmentList element shall not be present" violated here | todo |  |  |
| SCH-88 | `dash:Period`: DASH-IF IOP Section 3.2.2: "If a Period contains multiple Adaptation Sets with value of the @contentType="video" then at least one Adaptation Set shall contain a Role el-ement $&lt;$Role scheme="urn:mpeg:dash:role:2011" value="main"&gt;" violated here | todo |  |  |
| SCH-89 | `dash:Period`: DASH-IF IOP Section 3.10.4: "All Representations in one Period shall either conform to a DASH Live profile or DASH On-Demand profile", both Live and On-Demand profiles are signaled for the Period. | todo |  |  |
| SCH-90 | `dash:AdaptationSet`: If the content author signals the ability of Adaptation Set switching and as @segmentAlignment or @subsegmentAlignment are set to TRUE for one Adaptation Set, the (Sub)Segment alignment shall hold for all Representations in all Adaptation Sets for which the @id value is included in the @value attribute of the Supplemental descriptor. | todo |  |  |
| SCH-91 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with value of the @contentType="video" the following attributes shall be present: @par" violated here | todo |  |  |
| SCH-92 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For Adaptation Set or for any Representation within an Adaptation Set with value of the @contentType="video" the attribute @scanType shall either not be present or shall be set to 'progressive' ", violated here | todo |  |  |
| SCH-93 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with value of the @contentType="audio" the following attributes shall be present: @lang" violated here | todo |  |  |
| SCH-94 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with @contentType="video" the following attributes shall be present: @maxWidth or @width" violated here | todo |  |  |
| SCH-95 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with @contentType="video" the following attributes shall be present: @maxHeight or @height" violated here | todo |  |  |
| SCH-96 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with @contentType="video" the following attributes shall be present: @maxFrameRate or @frameRate" violated here | todo |  |  |
| SCH-97 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.2.2: For Live Profile @segmentAlignment shall be set to true for all Adaptation Sets | todo |  |  |
| SCH-98 | `dash:AdaptationSet`: DASH-IF IOP Section 3.2.2.2: For On-Demand Profile @subSegmentAlignment shall be set to true for all Adaptation Sets | todo |  |  |
| SCH-99 | `dash:AdaptationSet`: If a DASH-IF profile identifier is present, for the Adaptation Sets the mimeType shall be one of the six following type: "video/mp4", "audio/mp4", "application/mp4", "application/ttml+xml", "text/vtt" or "image/jpeg" | todo |  |  |
| SCH-100 | `dash:AdaptationSet`: The value of @profiles shall be a subset of the respective value in any higher level of the document hierarchy | todo |  |  |
| SCH-101 | `dash:AdaptationSet`: DASH-IF IOP Section 3.10.3: "AdaptationSet@subsegmentAlignment SHALL be present and set to true", not set appropriately. | todo |  |  |
| SCH-102 | `dash:AdaptationSet`: DASH-IF IOP Section 3.10.3: "AdaptationSet@subsegmentStartsWithSAP SHALL be present and set to 1 or 2", not set appropriately. | todo |  |  |
| SCH-103 | `dash:AdaptationSet`: HbbTV-DVB DASH Validation Requirements check violated for HbbTV: Section 'MPD' - For HbbTV profile, @segmentAlignment shall be set to true for all Adaptation Sets as it is based on live profile | todo |  |  |
| SCH-104 | `dash:Representation`: DASH-IF IOP (v3.3), Section 3.2.13 : "In contrast to MPEG-DASH which does not prohibit the use of multiplexed Representations, in the DASH-IF IOPs one Adaptation Set always contains exactly a single media type.". | todo |  |  |
| SCH-105 | `dash:Representation`: DASH-IF IOP Section 3.2.4: "For any Adaptation Sets with value of the @contentType="video" the following attributes shall be present: @maxWidth (or @width if all Representations have the same width), @maxHeight (or @height if all Representations have the same width), @maxFrameRate (or @frameRate if all Representations have the same width)" violated here | todo |  |  |
| SCH-106 | `dash:Representation`: DASH-IF IOP Section 3.2.4: "For any Representation within an Adaptation Set with value of the @contentType="video" the following attributes shall be present: @width, if not present in AdaptationSet element; @height, if not present in AdaptationSet element; @frameRate, if not present in AdaptationSet element; @sar" violated here | todo |  |  |
| SCH-107 | `dash:Representation`: DASH-IF IOP Section 3.2.4: "For Adaptation Set or for any Representation within an Adaptation Set with value of the @contentType="video" the attribute @scanType shall either not be present or shall be set to 'progressive' ", violated here | todo |  |  |
| SCH-108 | `dash:Representation`: DASH-IF IOP Section 3.2.4: "For any Representation within an Adaptation Set with value of the @contentType="audio" the following elements and attributes shall be present: @audioSamplingRate, if not present in AdaptationSet element; AudioChannelConfiguration, if not present in AdaptationSet element" violated here | todo |  |  |
| SCH-109 | `dash:Representation`: If profiles contain dash-if profile identifier, "codecs" attribute on AdaptationSet level OR Representation level shall not contain more than one identifiers as a comma separated list | todo |  |  |
| SCH-110 | `dash:Representation`: The @dependencyId attribute on the Enhancement Layer Representation shall refer to the Base Layer Representation @id attribute. | todo |  |  |
| SCH-111 | `dash:Representation`: The value of @profiles shall be a subset of the respective value in any higher level of the document hierarchy. | todo |  |  |
| SCH-112 | `dash:Representation`: DASH-IF IOP 4.3 Section 3.2.1 - "For on-demand profiles, @indexRange attribute shall be present. | todo |  |  |
| SCH-113 | `dash:SubRepresentation`: The value of @profiles shall be a subset of the respective value in any higher level of the document hierarchy | todo |  |  |
| SCH-114 | `dash:SegmentBase`: DASH-IF IOP 4.3 Section 3.2.1 - "For on-demand profiles, @indexRange attribute shall be present. | todo |  |  |
| SCH-115 | `dash:ContentProtection`: The ContentProtection descriptors shall always be present in the AdaptationSet element and apply to all contained Representations. | todo |  |  |
| SCH-116 | `dash:ContentProtection`: The ContentProtection descriptor for the mp4 protection scheme with @schemeIdUri 'urn:mpeg:dash:mp4protection:2011' and @value 'cenc' shall be present in the AdaptationSet element. | todo |  |  |
| SCH-117 | `dash:ContentProtection`: The ContentProtection Descriptor for the mp4protection scheme shall contain the attribute @cenc:default_KID. | todo |  |  |
| SCH-118 | `dash:AudioChannelConfiguration`: If profile http://dashif.org/guidelines/dashif#ac-4 is used, then schemeIdUri attribute shall be tag:dolby.com,2014:dash:audio_channel_configuration:2011. | todo |  |  |
| SCH-119 | `dash:AudioChannelConfiguration`: If profile http://dashif.org/guidelines/dashif#mpeg-h-3da is used, then schemeIdUri attribute shall be urn:mpeg:mpegB:cicp:ChannelConfiguration. | todo |  |  |
| SCH-120 | `dash:EssentialProperty`: If EssentialProperty descriptor with @schemeIdUri set to http://dashif.org/guidelines/thumbnail_tile is present, then value shall provide horizontal and vertical number of tiles separated by an 'x'. | todo |  |  |
| SCH-121 | `dash:EssentialProperty`: If EssentialProperty descriptor with @schemeIdUri set to http://dashif.org/guidelines/thumbnail_tile is present, then bandwidth shall be used to describe the tiling.'. | todo |  |  |
| SCH-122 | `dash:EssentialProperty`: If EssentialProperty descriptor with @schemeIdUri set to http://dashif.org/guidelines/thumbnail_tile is present, then height shall be used to describe the tiling. | todo |  |  |
| SCH-123 | `dash:EssentialProperty`: If EssentialProperty descriptor with @schemeIdUri set to http://dashif.org/guidelines/thumbnail_tile is present, then width shall be used to describe the tiling. | todo |  |  |

## Schematron: dvb-dash.sch

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| SCH-DVB-1 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:AdaptationSet[dlb:isAdaptationSetAudio(.)][not(dlb:isAuxiliaryStream(.))]/dash:Representation`: @mimeType shall be common between all Representations in an Adaptation Set | todo |  |  |
| SCH-DVB-2 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:AdaptationSet[dlb:isAdaptationSetAudio(.)][not(dlb:isAuxiliaryStream(.))]/dash:Representation`: @codecs should be common between all Representations in an Adaptation Set | todo |  |  |
| SCH-DVB-3 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:AdaptationSet[dlb:isAdaptationSetAudio(.)][not(dlb:isAuxiliaryStream(.))]/dash:Representation`: @audioSamplingRate should be common between all Representations in an Adaptation Set | todo |  |  |
| SCH-DVB-4 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:AdaptationSet[dlb:isAdaptationSetAudio(.)][not(dlb:isAuxiliaryStream(.))]/dash:Representation/dash:AudioChannelConfiguration`: audioChannelConfiguration should be common between all Representations in an Adaptation Set | todo |  |  |
| SCH-DVB-5 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:AdaptationSet[dlb:isAdaptationSetAudio(.)][dlb:isAuxiliaryStream(.)]`: All Adaptation Sets that refer to Auxiliary Audio streams may not contain the @lang attribute and Role, Accessibility, AudioChannelConfiguration descriptors | todo |  |  |
| SCH-DVB-6 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:Preselection[dlb:isAdaptationSetAudio(.)]`: Every AC-4 or MPEG-H Audio Preselection element shall include at least one Role element using the scheme "urn:mpeg:dash:role:2011" as defined in ISO/IEC 23009-1:2019 [1]. | todo |  |  |
| SCH-DVB-7 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:Preselection[dlb:isAdaptationSetAudio(.)]`: If there is more than one audio Preselection associated with an audio bundle, at least one of the Preselection elements shall be tagged with an @value set to "main". | todo |  |  |
| SCH-DVB-8 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]/dash:Period/dash:Preselection[dlb:isAdaptationSetAudio(.)]`: @preselectionComponents specifies the ids of the contained Adaptation Sets or Content Components that belong to this Preselection as white space separated list in processing order. | todo |  |  |
| SCH-DVB-9 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]//*[self::dash:AdaptationSet or self::dash:Preselection or self::Representation][dlb:isAdaptationSetAC4(.)]`: AC-4 audio should be encoded using bitstream_version = 2 (is ). | todo |  |  |
| SCH-DVB-10 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]//*[self::dash:AdaptationSet or self::dash:Preselection or self::Representation][dlb:isAdaptationSetAC4(.)]`: The presentation_version field according to clause 6.2.1.3 of ETSI TS 103 190-2 [46] shall be set to the value 1. | todo |  |  |
| SCH-DVB-11 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]//*[self::dash:AdaptationSet or self::dash:Preselection or self::Representation][dlb:isAdaptationSetAC4(.)]`: The md_compat field as defined by clause 6.3.2.2.3 of ETSI TS 103 190-2 [46] shall be less than or equal to three. | todo |  |  |
| SCH-DVB-12 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]//*[self::dash:AdaptationSet or self::dash:Preselection or self::Representation][dlb:isAdaptationSetAC4(.)]`: An AC-4 elementary stream shall be encoded with a sampling rate of 48 kHz, 96 kHz or 192 kHz. | todo |  |  |
| SCH-DVB-13 | `dash:MPD[dlb:isProfile(.,$dvbdash-profile-2017)]//*[self::dash:AdaptationSet or self::dash:Preselection or self::Representation][dlb:isAdaptationSetAC4(.)]`: An Adaptation Set shall not contain audio with a frame rate &gt; 60 Hz unless all video adaptationSets in the Period contain only video with a frame rate &gt; 60 Hz. | todo |  |  |

## Schematron: ac4-generic.sch

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| SCH-AC4-1 | `*[self::dash:AdaptationSet or self::dash:Preselection][dlb:isAdaptationSetAC4(.)]`: The @codecs attribute shall conform to the syntax described in IETF RFC 6381. The value of the parameter shall be set to a dot-separated list of four parts of which the last three are two-digit hexadecimal numbers. | todo |  |  |
| SCH-AC4-2 | `*[self::dash:AdaptationSet or self::dash:Preselection][dlb:isAdaptationSetAC4(.)]`: The value of the mimeType attribute shall be set to 'audio/mp4' or 'video/mp4'. | todo |  |  |
| SCH-AC4-3 | `*[self::dash:AdaptationSet or self::dash:Preselection][dlb:isAdaptationSetAC4(.)]`: @audioSamplingRate shall be set to the sampling frequency derived from the parameters fs_index and dsi_sf_multiplier, contained in ac4_dsi_v1. | todo |  |  |
| SCH-AC4-4 | `*[self::dash:AdaptationSet or self::dash:Preselection][dlb:isAdaptationSetAC4(.)]`: The value of the mimeType attribute shall be set to 'audio/mp4'. | todo |  |  |
| SCH-AC4-5 | `*[self::dash:AdaptationSet or self::dash:Preselection][dlb:isAdaptationSetAC4(.)]`: The @startWithSAP value shall be set to '1'. | todo |  |  |
| SCH-AC4-6 | `dash:Representation[matches(dlb:getNearestCodecString(.), 'ac-4\.00','i')]`: The representation DASH element shall include an AudioChannelConfiguration DASH descriptor. | todo |  |  |
| SCH-AC4-7 | `dash:Role[matches(dlb:getNearestCodecString(.), 'ac-4\.00')]`: The value of Role (role) shall be main, alternate, commentary. | todo |  |  |
| SCH-AC4-8 | `dash:SupplementalProperty[@schemeIdUri = 'tag:dolby.com,2016:dash:virtualized_content:2016'][matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')]`: The @value attribute of the immersive audio for headphones descriptor shall be 1 | todo |  |  |
| SCH-AC4-9 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.00')][@schemeIdUri eq $NSDLB_acc2014]`: It is recommended to use exactly one AudioChannelConfiguration element with a schemeIdURI of or | todo |  |  |
| SCH-AC4-10 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.00')][@schemeIdUri eq $NSDLB_acc2014]`: The value element shall contain a four-digit hexadecimal representation of the 16-bit field which describes the channel assignment of the referenced AC-4 elementary stream | todo |  |  |
| SCH-AC4-11 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.00')][@schemeIdUri eq $NSDLB_acc2014]`: Use &lt; schemeIdUri="" value=""/&gt; | todo |  |  |
| SCH-AC4-12 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][not(@schemeIdUri = ($NSDLB_acc2015,$NSMPEG_acc))]`: Unspecified schemeIdUri in AudioChannelConfiguration element | todo |  |  |
| SCH-AC4-13 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][@schemeIdUri eq $NSDLB_acc2015]`: It is recommended to use exactly one AudioChannelConfiguration element with a schemeIdURI of or | todo |  |  |
| SCH-AC4-14 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][@schemeIdUri eq $NSDLB_acc2015]`: The value element shall contain a six-digit hexadecimal representation of the 24-bit field which describes the channel assignment of the referenced AC-4 elementary stream | todo |  |  |
| SCH-AC4-15 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][@schemeIdUri eq $NSDLB_acc2015]`: For all AC-4 channel configurations that are mappable to the MPEG channel configuration scheme, the scheme described by @schemeIdUri="urn:mpeg:mpegB:cicp:ChannelConfiguration" shall be used | todo |  |  |
| SCH-AC4-16 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][@schemeIdUri eq $NSMPEG_acc]`: It is recommended to use exactly one AudioChannelConfiguration element with a schemeIdURI of or | todo |  |  |
| SCH-AC4-17 | `dash:AudioChannelConfiguration[matches(dlb:getNearestCodecString(.),'ac-4\.(01\|02)')][@schemeIdUri eq $NSMPEG_acc]`: Valid values are 1-7, 9-12, 14, 16-17, and 19 | todo |  |  |

## Module: Common

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| COMMON-1 | `Conformance Tool`: Schematron shall be able to run | n/a |  | checks that the upstream Schematron engine runs |
| COMMON-2 | `DASH 5th`: MPD SHALL be valid | done | MPD-XSD, MPD-XML | dashcheck uses the 6th edition schema |
| COMMON-3 | `DASH 2nd (DVB 2018)`: MPD SHALL be valid | todo |  |  |
| COMMON-4 | `DASH 4th-amd1 (DVB 2019)`: MPD SHALL be valid | todo |  |  |
| COMMON-5 | Segment analysis should not result in errors | done | INIT-PARSE, SEG-PARSE |  |

## Module: IOP

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| IOP-1 | `3.2.7.4`: MPD fields having datatype xs:duration shall not use year or month units | done | MPD-DURATION-YM |  |
| IOP-2 | `3.10.3`: SegmentTemplate@media attribute SHALL be present | done | ADDR-TEMPLATE |  |
| IOP-3 | `3.10.4`: @profiles signalling SHALL be present in each Period | todo |  |  |
| IOP-4 | `3.10.3`: SegmentTemplate@indexRange attribute SHALL be present | todo |  |  |
| IOP-5 | `6.2.5.2`: All AVC representations SHALL be encoded using avc3 | todo |  |  |
| IOP-6 | `6.2.5.2`: All AVC representations SHALL include an 'avcC' box containing SPS and PPS NALs | todo |  |  |
| IOP-7 | `6.2.5.2`: The level of the adaptation set SHALL match the maximum of the corresponding representations | todo |  |  |
| IOP-8 | `6.2.5.2`: The profile of the adaptation set SHALL match the maximum of the corresponding representations | todo |  |  |
| IOP-9 | `6.2.5.2`: The edit list, if present in one, SHALL be present in all representations | todo |  |  |
| IOP-10 | `6.2.5.2`: All HEVC representations SHALL be encoded using hev1 | todo |  |  |
| IOP-11 | `6.2.5.2`: All HEVC representations SHALL include an 'hvcC' box containing SPS, PPS and VPS NALs | todo |  |  |
| IOP-12 | `3.10.3.2`: Only a single 'sidx' box shall be present | todo |  |  |
| IOP-13 | `3.10.3.2`: Each representation SHALL have on segment | todo |  |  |
| IOP-14 | `6.2.5.2`: Edit lists SHALL NOT be present unless On-Demand profile is signalled | todo |  |  |
| IOP-15 | `6.2.5.2`: The composition time and decoded time for the first sample SHALL match | todo |  |  |

## Module: CMAF

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| CMAF-1 | `Section A.3`: The maximum encoding parameters [..as per the table..] SHALL not be exceeded | todo |  |  |
| CMAF-2 | `Section A.1.2`: Audio adaptation sets SHALL include at least one 'caac' representation | todo |  |  |
| CMAF-3 | `Section 9.2.1`: Video tracks SHALL contain either 'trun(v1)' OR 'elst' combined with 'trun(v0)', but not both | todo |  |  |
| CMAF-4 | `Section 7.3.5`: All media samples in a CMAF Fragment SHALL be addressed by byte offsets in the 'trun' box | todo |  |  |
| CMAF-5 | `Section 7.3.2.2`: baseMediaDecodeTime SHALL be equal to the sum of all prior segments | done | SEG-CONTINUITY | dashcheck checks continuity between consecutive checked segments |
| CMAF-6 | `Section A.1.2`: Tracks SHALL NOT contain a track encryption box | todo |  |  |
| CMAF-7 | `Section A.1.3`: Tracks SHALL be available in 'cenc' encryption | todo |  |  |
| CMAF-8 | `Section A.1.4`: Tracks SHALL be available in 'cbcs' encryption | todo |  |  |
| CMAF-9 | `Section B.2.4`: CMAF Switching sets SHALL be constrained to include identical SPS information | todo |  |  |
| CMAF-10 | `Section B.2.4`: CMAF Switching sets SHALL be constrained to include identical SEI nals | todo |  |  |
| CMAF-11 | `Section 7.3.5`: Each AVC CMAF Fragment SHALL be independently accessible | todo |  |  |
| CMAF-12 | `Section 7.3.5`: Each HEVC CMAF Fragment SHALL be independently accessible | todo |  |  |
| CMAF-13 | `Section 7.3.5`: Each Audio CMAF Fragment SHALL be independently accessible | todo |  |  |
| CMAF-14 | `Section 7.3.2.4`: The HEVCSampleEntry SHALL contain a 'colr' box with type 'nclx' | todo |  |  |
| CMAF-15 | `Section 7.3.2.4`: Each encrypted Fragment SHALL be independently decryptable | todo |  |  |
| CMAF-16 | `Section 7.5.2`: When metadata is carried in a 'meta' box, it SHALL NOT occur at the file level | todo |  |  |
| CMAF-17 | `Section 7.5.2`: When metadata is carried in a 'udta' box, it SHALL NOT occur at the file level | todo |  |  |
| CMAF-18 | `Section 7.3.9.1`: A CMAF switching set SHALL have only media type | todo |  |  |
| CMAF-19 | `Section 7.3.3.3`: Each subsegment referenced in the 'sidx' box SHALL be a single fragment | todo |  |  |
| CMAF-20 | `Section 7.3.4.1`: All CMAF subtitle tracks in a CMAF Switching Set SHALL conform to one CMAF Media Profile | todo |  |  |
| CMAF-21 | `Section 7.5.20`: All CMAF fragments in a 'im1i' track SHALL contain a 'subs' box | todo |  |  |
| CMAF-22 | `Section A.2 (AVC) / B.6 (HEVC)`: The maximum encoding parameters [..as per the table..] SHALL not be exceeded | todo |  |  |
| CMAF-23 | `Section A.1.2`: Video adaptation sets SHALL include at least one 'cfhd' representation | todo |  |  |

## Module: Low Latency

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| LL-1 | `9.X.4.1`: Media presentation offering Low Latency SHOULD be signalled with … | todo |  |  |
| LL-2 | `9.X.4.2`: Inband Event streams carrying MPD validity expiration events SHOULD be present | todo |  |  |
| LL-3 | `9.X.4.2`: If Inband Event streams carrying MPD validity expiration events are used, @value SHALL be set to 1 | todo |  |  |
| LL-4 | `9.X.4.2`: LeapSecondInformation SHOULD be provided | todo |  |  |
| LL-5 | `9.X.4.3`: At least one ProducerTimeReference SHALL be present | todo |  |  |
| LL-6 | `9.X.4.3`: Each ProducerTimeReference SHALL have a unique @id | todo |  |  |
| LL-7 | `9.X.4.3`: Either (constrained) SegmentTemplate or SegmentTimeline shall be present | todo |  |  |
| LL-8 | `9.X.4.3`: SegmentTemplate will have @duration set and have @media containing $Number$ | todo |  |  |
| LL-9 | `9.X.4.3`: SegmentTimeline will have @media containing $Number$ and $Time$ | todo |  |  |
| LL-10 | `9.X.4.2`: At least one valid ServiceDescription element SHALL be present | todo |  |  |
| LL-11 | `9.X.4.2`: At least one UTC timing description element SHALL be present with a correct @schemIdUri | partial | MPD-DYN-UTCTIMING, MPD-UTCTIMING-SCHEME | not tied to low-latency signalling |
| LL-12 | `9.X.4.5`: Each Segment SHALL Conform to a CMAF Fragment | todo |  |  |
| LL-13 | `9.X.4.5`: EMSG box constraints | todo |  |  |
| LL-14 | `9.X.4.5`: Each Segment MAY (and typically SHOULD) contain more than one CMAF chunk | todo |  |  |
| LL-15 | `9.X.4.5`: A Resync element SHOULD be assigned to each Representation | todo |  |  |
| LL-16 | `9.X.4.5`: @availabilityTimeOffset SHALL be present for each represention | todo |  |  |
| LL-17 | `9.X.4.5`: @availabilityTimeComple SHALL be present and set to 'FALSE' for each representation | todo |  |  |
| LL-18 | `9.X.4.5`: One of the following should hold | todo |  |  |
| LL-19 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @contentType shall correspond with the hdlr type | done | XCHK-HANDLER |  |
| LL-20 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @mimeType SHALL be either '&lt;@contentType&gt;/mp4' or &lt;#contentType&gt;/mp4, profiles='cmfc' | todo |  |  |
| LL-21 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @maxWidth SHOULD correspond to the width in the 'tkhd' box | partial | XCHK-SIZE | dashcheck compares @width with the sample entry |
| LL-22 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @maxHeight SHOULD correspond to the height in the 'tkhd' box | partial | XCHK-SIZE | dashcheck compares @height with the sample entry |
| LL-23 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @codecs shall correspond with the sample descriptor type | done | XCHK-CODECS, XCHK-CODECS-DETAIL |  |
| LL-24 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: If the content is protected, a ContentProtection element SHALL be present and set appropriately | todo |  |  |
| LL-25 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: @timescale SHALL correspond to the timescale in the 'mdhd' box | todo |  |  |
| LL-26 | `9.X.4.5 =&gt; MPEG-DASH 8.X.4`: Either segmentAlignment or subsegmentAlignment SHALL be set | todo |  |  |
| LL-27 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: Event message streams MAY be signalled with InbandEventStream elements | todo |  |  |
| LL-28 | `9.X.4.5`: All 'emsg' boxes inserted [..] after the start of the first CMAF chunk … | todo |  |  |
| LL-29 | `9.X.4.5`: Resync element SHALL/SHOULD be present (conditional) | todo |  |  |
| LL-30 | `9.X.4.5`: Resync element type SHALL be valid | todo |  |  |
| LL-31 | `9.X.4.5`: Resync element @marker SHALL be 'true' | todo |  |  |
| LL-32 | `9.X.4.5`: Resync element @dT SHOULD be lower than half the latency, and SHALL not exceed the target latency | todo |  |  |
| LL-33 | `9.X.4.5`: @qualityRanking should be used | todo |  |  |
| LL-34 | `9.X.4.4`: Each Segment SHOULD include only a single 'moof' box | todo |  |  |
| LL-35 | `9.X.4.4`: A segment with a single 'moof' box may carry an 'smds' brand | todo |  |  |
| LL-36 | `9.X.4.4`: A segment SHALL NOT exceed 50% and SHOULD not exceed 30% of the tareget latency | todo |  |  |
| LL-37 | `9.X.4.3`: A Low Latency Adaptation Set SHALL either be a Segment or a Chunked Set | todo |  |  |
| LL-38 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The tolerance for the EPT relative to the first stegment SHALL NOT exceed 50% | done | SEG-EPT | for all number-based streams, not only low latency |
| LL-39 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: If an 'S' elements and it predecessor share the same duration, they SHOULD be combined | todo |  |  |
| LL-40 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: If each chunk is an addressable object, '@k SHALL be present in the SegmentTimeline | todo |  |  |
| LL-41 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 't' attribute SHALL correspond to the EPT of the corresponding segment | done | SEG-EPT, SEG-EPT-CTO | for all SegmentTimeline streams, not only low latency |
| LL-42 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'd' attribute SHALL correspond to the duration of the corresponding segment | done | SEG-DURATION | for all SegmentTimeline streams, not only low latency |
| LL-43 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: Exactly one 'sidx' box shall be used | todo |  |  |
| LL-44 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box shall be placed before any 'moof' boxes | todo |  |  |
| LL-45 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box reference_ID SHALL be equal to the track id | todo |  |  |
| LL-46 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box timescale SHALL be identical to the one in the 'mdhd' box | todo |  |  |
| LL-47 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box earliest presentation time SHALL match the segment EPT | todo |  |  |
| LL-48 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box reference count SHALL match the number of 'mmof' boxes | todo |  |  |
| LL-49 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box reference_type shall be set to 0 | todo |  |  |
| LL-50 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box startwithsap shall be set to 1 | todo |  |  |
| LL-51 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box sapType shall be set to 1 or 2 | todo |  |  |
| LL-52 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The 'sidx' box sapDeltaTime shall be set to 0 | todo |  |  |
| LL-53 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The baseMediaDecodeTime shall be 0 | todo |  |  |
| LL-54 | `9.X.4.5 =&gt; MPEG-DASH 8.X.3`: The list of compatible brands SHALL contain 'dash' | todo |  |  |

## Module: DVB

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| DVB-1 | `Unknown`: MPD@minimumUpdatePeriod SHOULD have a value of 1 second or higher | todo |  |  |
| DVB-2 | `Section 7.2.1.1`: A fontdownload descriptor SHALL only be placed in AdaptationSets containing subtitles | todo |  |  |
| DVB-3 | `Section 6.3.1`: The AudioChannelConfiguration shall be compliant with the dolby scheme | todo |  |  |
| DVB-4 | `Section 6.1.2`: Each audio AdaptationSet shall include at least one Role Element | todo |  |  |
| DVB-5 | `Section 6.1.1`: All audio Representations SHALL have the attributes and elements in Table 4 | todo |  |  |
| DVB-6 | `Section 6.1.1`: @mimeType SHALL be common for all audio Representations within an AdaptationSet | todo |  |  |
| DVB-7 | `Section 6.1.1`: @codec SHOULD be common for all audio Representations within an Adapation Set | todo |  |  |
| DVB-8 | `Section 6.1.1`: @audioSamplingRate SHOULD be common for all audio Representations within an Adapation Set | todo |  |  |
| DVB-9 | `Section 6.1.1`: AudioChannelConfiguration SHOULD be common for all audio Representations within an Adapation Set | todo |  |  |
| DVB-10 | `Section 6.6.3`: An audio fallback set shall have @value equal to the @id of the base set | todo |  |  |
| DVB-11 | `Section 11.3.0`: Audio should be 20% or less of the total stream bandwith | todo |  |  |
| DVB-12 | `Codec Information`: The codec should be supported by the specification | todo |  |  |
| DVB-13 | `Section 8.3`: ContentProtection descriptor(s) shall be placed at the AdaptationSet level | todo |  |  |
| DVB-14 | `Section 4.5`: The MPD size SHALL NOT exceed 256 Kbytes | todo |  |  |
| DVB-15 | `Section 4.5`: The MPD has a maximum of 64 periods | todo |  |  |
| DVB-16 | `Section 4.5`: Each Period has a maximum of 16 Adaptation Sets | todo |  |  |
| DVB-17 | `Section 4.5`: Each Adaptation Set has a maximum of 16 Representations | todo |  |  |
| DVB-18 | `Section 9.1.2.1`: Events SHALL have @presentationTime set | todo |  |  |
| DVB-19 | `Section 9.1.2.1`: The event data shall be valid XML, either escaped or place in a CDATA section | todo |  |  |
| DVB-20 | `Section 10.12.3.3`: Reporting URL is required to be an absolute HTTP(S) url | todo |  |  |
| DVB-21 | `Section 10.12.3.4`: Probability - if present - is required to be an integer between 1 and 1000 | todo |  |  |
| DVB-22 | `Section 4.2.2`: 'The Period.SegmentList element SHALL NOT be present' | todo |  |  |
| DVB-23 | `Section 4.2.2`: 'The Period.SegmentTemplate element SHALL NOT be present' | todo |  |  |
| DVB-24 | `Section 4.2.2`: Each Period element shall conform to either 4.2.4 or 4.2.6 | todo |  |  |
| DVB-25 | `Section 4.2.2`: At least one Adaptation Set SHALL cary a 'main' role | todo |  |  |
| DVB-26 | `Section 6.1.2`: At least one Adaptation Set SHALL cary a 'main' role | todo |  |  |
| DVB-27 | `Section 11.1`: Each Representation @profile SHALL include at least one of those in section 4.1 | todo |  |  |
| DVB-28 | `Section 11.1`: Each representation @profile SHALL include either section 4.2.5 or section 4.2.8 | todo |  |  |
| DVB-29 | `Section 11.1`: Each representation shall have only one segment | todo |  |  |
| DVB-30 | `Codec Information`: The resolution of the stream should conform to table 10.3 | todo |  |  |
| DVB-31 | `Section 9.1.9`: The @value attribute SHOULD be absent | todo |  |  |
| DVB-32 | `Section 9.1.9`: The @presentationTimeOffset attribute MAY be present | todo |  |  |
| DVB-33 | `Section 9.1.9`: The @timeScale attribute shall be set to an integer multiple of the video framerate | todo |  |  |
| DVB-34 | `Section 9.1.9`: The @presentationTime of an event SHALL be set to the splice time of the enclosed marker | todo |  |  |
| DVB-35 | `Section 9.1.9`: The @duration of an event SHALL be defined [..] and SHALL NOT be 0xFFFFFFFF | todo |  |  |
| DVB-36 | `Section 9.1.9`: The @messageData of an event SHALL be absent | todo |  |  |
| DVB-37 | `Section 9.1.9`: The @contentEncoding of an event is absent | todo |  |  |
| DVB-38 | `Section 7.1.1`: The @codecs attribute shall begin with 'stpp' to indicate the use of XML subtitles | todo |  |  |
| DVB-39 | `Bitrate information`: Terminal which does not support UHD video (max 12 Mbit/s) | todo |  |  |
| DVB-40 | `Bitrate information`: Terminal which does not support UHD, but not UHD HFR video (max 39 Mbit/s) | todo |  |  |
| DVB-41 | `Bitrate information`: Terminal which does supports UHD HFR video (max 51 Mbit/s) | todo |  |  |
| DVB-42 | `Section 4.7.2`: The MPD SHOULD contain at least one UTCTiming element with the @schemeIdUri attribute set to [...] | partial | MPD-DYN-UTCTIMING | dashcheck only checks dynamic MPDs and accepts any 2014 HTTP scheme |
| DVB-43 | `Section 4.8.3`: MPD@timeShiftBufferDepth shall not change | todo |  |  |
| DVB-44 | `Section 4.8.3`: MPD@availabilityStartTime shall not change | todo |  |  |
| DVB-45 | `Section 4.8.3`: MPD@maxSegmentDuration shall not change to a larger duration | todo |  |  |
| DVB-46 | `Section 4.8.3`: @id shall not change for a corresponding Period | todo |  |  |
| DVB-47 | `Section 4.8.3`: @start shall not change for a corresponding Period | todo |  |  |
| DVB-48 | `Section 4.8.3`: @AssetIdentifier shall not change for a corresponding Period | todo |  |  |
| DVB-49 | `Section 4.8.3`: @id shall not change for a corresponding AdaptationSet | todo |  |  |
| DVB-50 | `Section 4.8.3`: @id shall not change for a corresponding Representation | todo |  |  |
| DVB-51 | `Section 4.8.3`: Role Element shall not change for a corresponding AdaptationSet or corresponding Representation | todo |  |  |
| DVB-52 | `Section 4.8.3`: AudioConfiguration Element shall not change for a corresponding AdaptationSet … | todo |  |  |
| DVB-53 | `Section 4.8.3`: @contentType shall not change for a corresponding AdaptationSet or corresponding Representation | todo |  |  |
| DVB-54 | `Section 4.8.3`: @codecs shall not change for a corresponding AdaptationSet or corresponding Representation | todo |  |  |
| DVB-55 | `Section 4.8.3`: @lang shall not change for a corresponding AdaptationSet or corresponding Representation | todo |  |  |
| DVB-56 | `Section 7.2.1.1`: A fontdownload descriptor SHALL only be placed in AdaptationSets containing subtitles | todo |  |  |
| DVB-57 | `Section 4.4`: Elements and attributes are expected to be present [..] to enable selection and switching. | todo |  |  |
| DVB-58 | `Section 4.4`: @scanType SHALL be present [..] if interlaced pictures are used in any [..representation] | todo |  |  |
| DVB-59 | `Section 11.2.2`: The frame rates used should be integer multiples of eachother | todo |  |  |
| DVB-60 | `5.4.6`: The value of the signalled ColourPrimaries Property in the MPD SHALL match … | todo |  |  |
| DVB-61 | `5.4.6`: The value of the signalled MatrixCoefficients Property in the MPD SHALL match … | todo |  |  |
| DVB-62 | `5.4.6`: The value of the signalled TransferCharacteristics Property in the MPD SHALL match … | todo |  |  |
| DVB-63 | `5.4.7`: Use of HLG10 within in an AdapationSet shall be signalled according to this section | todo |  |  |
| DVB-64 | `5.4.8`: Use of PQ10 within in an AdapationSet shall be signalled according to this section | todo |  |  |
| DVB-65 | `5.2.2`: The value of the @codecs attribute shall be set in accordance with ISO/IEC 14496-15 | todo |  |  |
| DVB-66 | `5.2.2`: The value of the signalled ColourPrimaries Property in the MPD SHALL match … | todo |  |  |
| DVB-67 | `5.2.2`: The value of the signalled MatrixCoefficients Property in the MPD SHALL match … | todo |  |  |
| DVB-68 | `5.2.2`: The value of the signalled TransferCharacteristics Property in the MPD SHALL match … | todo |  |  |
| DVB-69 | `5.2.6`: Use of HLG10 within in an AdapationSet shall be signalled according to this section | todo |  |  |
| DVB-70 | `5.2.7`: Use of PQ10 within in an AdapationSet shall be signalled according to this section | todo |  |  |
| DVB-71 | `Section 4.3`: The 'moof' box shall contain only one 'traf' box | todo |  |  |
| DVB-72 | `Section 4.3`: The segment shall contain only one 'sidx' box | todo |  |  |
| DVB-73 | `5.1.2`: Content SHOULD be offered using 'avc3' or 'avc4' | todo |  |  |
| DVB-74 | `5.1.2`: All information necessary to decode any Segment shall be provided in the init segment | todo |  |  |
| DVB-75 | `DRM`: Content Protection SHALL contain a ContentProtectionDescriptor | todo |  |  |
| DVB-76 | `DRM`: PSSH Boxes SHOULD exist in either the MPD or the Initialization Segment | todo |  |  |
| DVB-77 | `DRM`: MPD DRM System information | todo |  |  |
| DVB-78 | `DRM`: The scheme SHOULD be set to 'cenc' | todo |  |  |
| DVB-79 | `Section 10.5.2.3`: Continous adaptation sets shall be signalled accordingly | todo |  |  |
| DVB-80 | `Section 10.5.2.3`: All Representations in the Adaptation Set in the first Period shall share the same value EPT1 | todo |  |  |
| DVB-81 | `Section 10.5.2.3`: All Representations in the Adaptation Set in a subsequent Period shall share the same value EPT2 | todo |  |  |
| DVB-82 | `6.1.1`: @mimeType SHALL match the mimetype derived from the segments | partial | XCHK-HANDLER | dashcheck compares the track handler with @mimeType |
| DVB-83 | `6.1.1`: @audioSamplingRate SHALL match the sample rate derived from the segments | done | XCHK-SAMPLERATE | dashcheck reports a warning |
| DVB-84 | `6.1.1`: AudioChannelConfiguration SHALL match the configuration derived from the segments | todo |  |  |
| DVB-85 | `5.x / 6.x`: For each representation, @codecs SHALL be match the codec derived from the segments | done | XCHK-CODECS, XCHK-CODECS-DETAIL |  |
| DVB-86 | `4.5`: Each subsegment shall have a duration of not more than 15 seconds | todo |  |  |
| DVB-87 | `4.5`: Each subsegment shall be at least 1 second | todo |  |  |
| DVB-88 | The codec should be supported by the specification | todo |  |  |
| DVB-89 | Profile used for AVC must be suported by the specification | todo |  |  |
| DVB-90 | Level used for AVC must be suported by the specification | todo |  |  |
| DVB-91 | Tier used for HEVC must be suported by the specification | todo |  |  |
| DVB-92 | Bit depth used for HEVC must be suported by the specification | todo |  |  |
| DVB-93 | Profile used for HEVC must be suported by the specification | todo |  |  |
| DVB-94 | Level used for HEVC must be suported by the specification | todo |  |  |
| DVB-95 | `5.1.2`: Segments shall start with SAP types 1 or 2 | partial | SEG-SAP | dashcheck requires a sync sample only when @startWithSAP is 1 or 2 |
| DVB-96 | `5.2.4`: For HEVC streams, SEI messages are placed according to ETSI TS 101 154 - L.3.3.8 | todo |  |  |
| DVB-97 | `5.2.4`: For VVC streams, SEI messages are placed according to ETSI TS 101 154 - 5.15.1 | todo |  |  |
| DVB-98 | `Subtitles`: For subtitle media, handler SHALL be 'subt' | todo |  |  |
| DVB-99 | `Subtitles`: Sample entry shall be 'stpp' | todo |  |  |
| DVB-100 | `Subtitles`: Sample entry shall countain a namespace | todo |  |  |
| DVB-101 | `4.3`: Initialization segment for Representations with an Adaptation Set SHALL have the same … | todo |  |  |
| DVB-102 | `4.3`: Initialization segment for Representations with an Adaptation Set SHALL have the same … | todo |  |  |
| DVB-103 | `8.3`: Initialization segment for Representations with an Adaptation Set SHALL have the same … | todo |  |  |

## Module: HbbTV

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| HBBTV-1 | `Unknown`: Inherit DVB legacy checks | n/a |  | marker in the upstream tool: HbbTV inherits the DVB checks |
| HBBTV-2 | `Codec Information`: The codec should be supported by the specification | todo |  |  |
| HBBTV-3 | `Unknown`: Inherit DVB legacy checks | n/a |  | marker in the upstream tool: HbbTV inherits the DVB checks |
| HBBTV-4 | `Unknown`: Inherit DVB legacy checks | n/a |  | marker in the upstream tool: HbbTV inherits the DVB checks |
| HBBTV-5 | `DRM`: Inherit DVB legacy checks | n/a |  | marker in the upstream tool: HbbTV inherits the DVB checks |
| HBBTV-6 | `DRM`: Inherit DVB legacy checks | n/a |  | marker in the upstream tool: HbbTV inherits the DVB checks |
| HBBTV-7 | The codec should be supported by the specification | todo |  |  |
| HBBTV-8 | Profile used for AVC must be suported by the specification | todo |  |  |
| HBBTV-9 | Level used for AVC must be suported by the specification | todo |  |  |
| HBBTV-10 | All info necessary to decode any audio segment shall be provided in the Initialization segment | todo |  |  |

## Module: CTA WAVE

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| WAVE-1 | `4.3.1`: Each WAVE audio Media profile SHALL conform to normative ref. listed in Table 1 | todo |  |  |
| WAVE-2 | `4.1`: Wave content SHALL include one or more switch sets conforming to at least one approved CMAF profile | todo |  |  |
| WAVE-3 | `5 `: If an audio track is included, the conforming (presentation will at least … | todo |  |  |
| WAVE-4 | `7.2.2`: Sequential sets SHALL conform to the same CMAF profile | todo |  |  |
| WAVE-5 | `7.2.2`: Wave content SHALL contain one CENC scheme per program | todo |  |  |
| WAVE-6 | `7.2.2 - Presentation Splicing`: Sample entries in Sequential Switching Sets Shall not change at Splice points | todo |  |  |
| WAVE-7 | `7.2.2 - Presentation Splicing`: Framerate in Sequential Switching Sets Shall not change at Splice points | todo |  |  |
| WAVE-8 | `7.2.2 - Presentation Splicing`: Channel count in Sequential Switching Sets Shall not change at Splice points | todo |  |  |
| WAVE-9 | `7.2.2 - Presentation Splicing`: Picture Aspect Ration (PAR) in Sequential Switching Sets Shall not change at Splice points | todo |  |  |
| WAVE-10 | `7.2.2 - Presentation Splicing`: Track Ids can change at Splice points | todo |  |  |
| WAVE-11 | `7.2.2 - Presentation Splicing`: Timescale can change at Splice points | todo |  |  |
| WAVE-12 | `7.2.2 - Presentation Splicing`: Encryption scheme can change at Splice points | todo |  |  |
| WAVE-13 | `7.2.2 - Presentation Splicing`: Default KID can change at Splice points | todo |  |  |
| WAVE-14 | `4.4.1`: Each WAVE Subtitle Media profile SHALL conform to normative ref. listed in Table 1 | todo |  |  |
| WAVE-15 | `4.1`: Wave content SHALL include one or more switch sets conforming to at least one approved CMAF profile | todo |  |  |
| WAVE-16 | `5 `: If a subtitle track is included, the conforming (presentation will at least … | todo |  |  |
| WAVE-17 | `7.2.2`: Sequential sets SHALL conform to the same CMAF profile | todo |  |  |
| WAVE-18 | `4.2.1`: Each WAVE video Media profile SHALL conform to normative ref. listed in Table 1 | todo |  |  |
| WAVE-19 | `4.1`: Wave content SHALL include one or more switch sets conforming to at least one approved CMAF profile | todo |  |  |
| WAVE-20 | `5 `: If a video tracks is included, the conforming (presentation will at least … | todo |  |  |
| WAVE-21 | `7.2.2`: Sequential sets SHALL conform to the same CMAF profile | todo |  |  |

## Module: WAVE HLS interop

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| WAVEHLS-1 | `4.1.2 - Basic On-Demand and Live Streaming`: CMAF Track Files [..] SHALL contain a single 'sidx' following the CMAF header … | todo |  |  |
| WAVEHLS-2 | `4.1.2 -  Basic On-Demand and Live Streaming`: The Average Bitrate of a CMAF Fragment [..] SHOULD be within 10% of the Average Bitrate … | todo |  |  |
| WAVEHLS-3 | The common encryption `cbcs` scheme SHALL be used for encryption | todo |  |  |
| WAVEHLS-4 | Constant 16-byte Initialization Vectors SHALL be used | todo |  |  |
| WAVEHLS-5 | An alternative using the `cenc` scheme MAY be produced | todo |  |  |
| WAVEHLS-6 | Constant 8-byte Initialization Vectors SHALL be used for 'cenc' encrypted material | todo |  |  |
| WAVEHLS-7 | Sample auxiliary information, if present, SHALL be addressed by [.. a 'saio' box] | todo |  |  |
| WAVEHLS-8 | Video components SHALL be encrypted with a 1:9 pattern | todo |  |  |
| WAVEHLS-9 | Audio components SHALL be encrypted with a 10:0 pattern | todo |  |  |
| WAVEHLS-10 | `4.6.2 - Rotation of Encryption Keys`: CMAF Segments [.. with differing encryption keys] SHALL provide an 'sbgp' box [..] | todo |  |  |
| WAVEHLS-11 | `4.6.2 - Rotation of Encryption Keys`: CMAF Segments [.. with differing encryption keys] SHALL provide a 'pssh' box [..] | todo |  |  |
| WAVEHLS-12 | `4.3.2 - Encrypted Media Presentations`: Any individual CMAF Segment SHALL have a single encryption key | todo |  |  |
| WAVEHLS-13 | `4.3.2 - Encrypted Media Presentations`: Any individual CMAF Segment SHALL have a single Initialization Vector | todo |  |  |
| WAVEHLS-14 | `4.4.2 - Presentation Splicing`: CMAF Fragment boundaries SHALL be created at all splice points. | todo |  |  |
| WAVEHLS-15 | `4.4.2 - Presentation Splicing`: CMAF Fragment [duration difference] SHALL be within one ISOBMFF sample duration | todo |  |  |
| WAVEHLS-16 | `4.1.2 - Basic On-Demand and Live Streaming`: Text components SHALL be packaged in ISMC1, ISMC1.1 or WebVTT Tracks | todo |  |  |
| WAVEHLS-17 | `4.5.2 - Carriage of Timed Event Data`: All 'emsg' boxes inserted [..] after the start of the first CMAF chunk … | todo |  |  |
| WAVEHLS-18 | Track roles SHALL be stored in one or more 'kind' boxes [..] | todo |  |  |
| WAVEHLS-19 | Track roles SHALL be representaed by the DASH Role scheme when possible | todo |  |  |

## Module: Dolby

| ID | Check | Status | dashcheck rules | Note |
|---|---|---|---|---|
| DOLBY-1 | `E.6.3 [13197]`: Bitstream version must match between TOC and DAC4 | todo |  |  |
| DOLBY-2 | `E.6.4 [14309]`: FS index must match between TOC and DAC4 | todo |  |  |
| DOLBY-3 | `E.6.5 [14209]`: Framerate index version must match between TOC and DAC4 | todo |  |  |
| DOLBY-4 | `E.6.6 [14215]`: Number of presentations must match between TOC and DAC4 | todo |  |  |
| DOLBY-5 | `E.6.7 [14221]`: Short program id must match between TOC and DAC4 | todo |  |  |
