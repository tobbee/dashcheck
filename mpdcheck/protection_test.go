package mpdcheck

import (
	"bytes"
	"encoding/base64"
	"slices"
	"strings"
	"testing"

	"github.com/Eyevinn/mp4ff/mp4"
)

const (
	widevineID  = "edef8ba9-79d6-4ace-a3c8-27dcd51d21ed"
	playReadyID = "9a04f079-9840-4286-ab92-e65be0885f95"
)

// psshBase64 returns a pssh box for systemID, followed by extra bytes, in base64.
func psshBase64(t *testing.T, systemID string, extra []byte) string {
	t.Helper()
	pssh, err := mp4.NewPsshBox(systemID, []string{"9eb4050d-e44b-4802-932e-27d75083e266"}, []byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := pssh.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(append(buf.Bytes(), extra...))
}

func TestProtectionRules(t *testing.T) {
	var free bytes.Buffer
	if err := mp4.NewFreeBox([]byte("data")).Encode(&free); err != nil {
		t.Fatal(err)
	}
	mp4p := `<ContentProtection schemeIdUri="urn:mpeg:dash:mp4protection:2011" value="cenc" cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"/>`
	withDRM := func(scheme, pssh string) string {
		return mp4p + `
      <ContentProtection schemeIdUri="` + scheme + `" value="DRM 1.0"><cenc:pssh>` + pssh + `</cenc:pssh></ContentProtection>`
	}
	wv := psshBase64(t, widevineID, nil)
	cases := []struct {
		name     string
		old, new string
		want     []string
	}{
		{"clean", "", "", nil},
		{"versioned value", `value="cenc"`, `value="cenc:00010000"`, nil},
		{"percent-encoded value", `value="cenc"`, `value="c%65nc"`, nil},
		{"cbc1", `value="cenc"`, `value="cbc1"`, nil},
		{"version too long", `value="cenc"`, `value="cenc:123456789"`, []string{"MPD-CP-VALUE"}},
		{"KID without hyphens", `9eb4050d-e44b-4802-932e-27d75083e266`, `9eb4050de44b4802932e27d75083e266`, []string{"MPD-CP-CENC"}},
		{"KID list", `9eb4050d-e44b-4802-932e-27d75083e266`, `9eb4050d-e44b-4802-932e-27d75083e266 11111111-2222-3333-4444-555555555555`, nil},
		{"pssh", mp4p, withDRM("urn:uuid:"+widevineID, wv), nil},
		{"pssh with spaces", mp4p, withDRM("urn:uuid:"+strings.ToUpper(widevineID), "\n        "+wv[:20]+"\n        "+wv[20:]+"\n      "), nil},
		{"pssh of another system", mp4p, withDRM("urn:uuid:"+playReadyID, wv), []string{"MPD-CP-CENC"}},
		{"pssh not base64", mp4p, withDRM("urn:uuid:"+widevineID, "!!!"), []string{"MPD-CP-CENC"}},
		{"pssh of a free box", mp4p, withDRM("urn:uuid:"+widevineID, base64.StdEncoding.EncodeToString(free.Bytes())), []string{"MPD-CP-CENC"}},
		{"pssh with bytes after it", mp4p, withDRM("urn:uuid:"+widevineID, psshBase64(t, widevineID, []byte{0, 0})), []string{"MPD-CP-CENC"}},
		{"pssh in mp4protection", `cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"/>`,
			`cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"><cenc:pssh>` + wv + `</cenc:pssh></ContentProtection>`, []string{"MPD-CP-CENC"}},
		{"robustness on mp4protection", `value="cenc"`, `value="cenc" robustness="HW"`, []string{"MPD-CP-ROBUSTNESS"}},
		{"robustness on a DRM", mp4p, withDRM("urn:uuid:"+widevineID, wv) + `<ContentProtection schemeIdUri="urn:uuid:` + playReadyID + `" robustness="3000"/>`, nil},
		{"MPD level without refId", `<Period id="p0">`, `<ContentProtection schemeIdUri="urn:uuid:` + widevineID + `"/>
  <Period id="p0">`, []string{"MPD-CP-REF"}},
		{"Period level without refId", `<Period id="p0">`, `<Period id="p0">
    <ContentProtection schemeIdUri="urn:uuid:` + widevineID + `"/>`, []string{"MPD-CP-REF"}},
		{"ref and refId", `value="cenc"`, `value="cenc" refId="cp" ref="cp"`, []string{"MPD-CP-REF"}},
		{"source in a later Period", mp4p, `<ContentProtection schemeIdUri="urn:mpeg:dash:mp4protection:2011" ref="late"/>`,
			[]string{"MPD-CP-REF"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := staticMPD
			if c.old != "" {
				if !strings.Contains(data, c.old) {
					t.Fatalf("test MPD lacks %q", c.old)
				}
				data = strings.Replace(data, c.old, c.new, 1)
			}
			if c.name == "source in a later Period" {
				data = strings.Replace(data, "  </Period>\n", `  </Period>
  <Period id="p1">
    <ContentProtection schemeIdUri="urn:mpeg:dash:mp4protection:2011" value="cenc" refId="late" cenc:default_KID="9eb4050d-e44b-4802-932e-27d75083e266"/>
  </Period>
`, 1)
			}
			md, findings := Check([]byte(data), "https://example.com/vod.mpd")
			if md == nil {
				t.Fatalf("MPD not parsed: %v", findings)
			}
			var got []string
			for _, f := range findings {
				if strings.HasPrefix(f.Rule, "MPD-CP-") || f.Rule == "MPD-XSD" {
					got = append(got, f.Rule)
				}
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v\n%v", got, c.want, findings)
			}
		})
	}
}
