package addressing

import "testing"

func TestContentProtectionRef(t *testing.T) {
	md := mustReadMPD(t, `<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" xmlns:cenc="urn:mpeg:cenc:2013" type="static" mediaPresentationDuration="PT4S">
  <ContentProtection schemeIdUri="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed" value="Widevine" refId="wv"><cenc:pssh>AAAA</cenc:pssh></ContentProtection>
  <Period id="p0">
    <ContentProtection schemeIdUri="urn:mpeg:dash:mp4protection:2011" value="cenc" refId="cp" cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"/>
    <AdaptationSet mimeType="video/mp4">
      <ContentProtection schemeIdUri="urn:mpeg:dash:mp4protection:2011" ref="cp"/>
      <SegmentTemplate timescale="1" duration="2" media="$Number$.m4s" initialization="init.mp4"/>
      <Representation id="v1" bandwidth="1000">
        <ContentProtection schemeIdUri="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed" value="Widevine 2" ref="wv"/>
        <ContentProtection schemeIdUri="urn:uuid:9a04f079-9840-4286-ab92-e65be0885f95" ref="missing"/>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`)
	reps, f := Reps(md, "https://example.com/a.mpd")
	if len(f) > 0 || len(reps) != 1 {
		t.Fatalf("got %d reps, %v", len(reps), f)
	}
	cps := reps[0].ContentProtections
	if len(cps) != 3 {
		t.Fatalf("got %d ContentProtection elements, want the AdaptationSet's and the Representation's", len(cps))
	}
	if cps[0].Value != "cenc" || cps[0].DefaultKID != "9eb4050d-e44b-4802-932e-27d75083e266" || cps[0].Ref != "cp" || cps[0].RefId != "" {
		t.Errorf("mp4protection not resolved: %+v", cps[0])
	}
	if cps[1].Value != "Widevine 2" || cps[1].Pssh == nil || cps[1].Pssh.Value != "AAAA" {
		t.Errorf("own @value must stay and cenc:pssh be inherited: %+v", cps[1])
	}
	if cps[2].Ref != "missing" || cps[2].Value != "" {
		t.Errorf("an element without source must stay as it is: %+v", cps[2])
	}
	if src := md.Periods[0].ContentProtections[0]; src.RefId != "cp" {
		t.Errorf("the source element was changed: %+v", src)
	}
}
