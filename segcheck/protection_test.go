package segcheck

import (
	"bytes"
	"slices"
	"testing"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
)

const (
	testKID  = "9eb4050d-e44b-4802-932e-27d75083e266"
	widevine = "urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed"
	otherKID = "11111111-2222-3333-4444-555555555555"
)

func cp(scheme, value, kid string) *m.ContentProtectionType {
	return &m.ContentProtectionType{DescriptorType: m.DescriptorType{SchemeIdUri: m.AnyURI(scheme), Value: value}, DefaultKID: kid}
}

func TestProtectionInitRules(t *testing.T) {
	cenc := &Init{Encrypted: true, Scheme: "cenc", DefaultKID: testKID}
	cases := []struct {
		name string
		init *Init
		cps  []*m.ContentProtectionType
		want []string
	}{
		{"clear", &Init{}, nil, nil},
		{"cenc", cenc, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cenc", testKID), cp(widevine, "", "")}, nil},
		{"no ContentProtection", cenc, nil, []string{"XCHK-CP-MISSING"}},
		{"no mp4protection", cenc, []*m.ContentProtectionType{cp(widevine, "", testKID)}, []string{"XCHK-CP-MISSING"}},
		{"no mp4protection, not Common Encryption", &Init{Encrypted: true, Scheme: "piff"}, []*m.ContentProtectionType{cp(widevine, "", "")}, nil},
		{"mp4protection on a clear track", &Init{}, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cenc", testKID)}, []string{"XCHK-CP-SCHEME"}},
		{"other scheme", &Init{Encrypted: true, Scheme: "cbcs", DefaultKID: testKID}, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cenc", testKID)}, []string{"XCHK-CP-SCHEME"}},
		{"versioned value", cenc, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cenc:00010000", testKID)}, nil},
		{"percent-encoded value", cenc, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "c%65nc", testKID)}, nil},
		{"no value", cenc, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "", testKID)}, nil},
		{"no default_KID", cenc, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cenc", "")}, nil},
		{"upper-case default_KID", cenc, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cenc", "9EB4050D-E44B-4802-932E-27D75083E266")}, nil},
		{"default_KID list", cenc, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cenc", otherKID+" "+testKID)}, nil},
		{"other default_KID", cenc, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cenc", otherKID)}, []string{"XCHK-CP-KID"}},
		{"other default_KID on a DRM descriptor", cenc, []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cenc", testKID), cp(widevine, "", otherKID)}, []string{"XCHK-CP-KID"}},
	}
	for _, c := range cases {
		rep := &addressing.Rep{ContentProtections: c.cps}
		if got := ruleIDs(protectionInitRules(rep, c.init, rep.Loc())); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TestProtectedInit checks the scheme and default KID that CheckInit reads
// from an initialization segment encrypted with mp4ff.
func TestProtectedInit(t *testing.T) {
	mf, err := mp4.DecodeFile(bytes.NewReader(read(t, "V300/init.mp4")))
	if err != nil {
		t.Fatal(err)
	}
	kid, err := mp4.NewUUIDFromString(testKID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mp4.InitProtect(mf.Init, make([]byte, 16), make([]byte, 16), "cbcs", kid, nil); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := mf.Init.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		cps  []*m.ContentProtectionType
		want []string
	}{
		{"matching", []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cbcs", testKID)}, nil},
		{"no ContentProtection", nil, []string{"XCHK-CP-MISSING"}},
		{"other scheme and KID", []*m.ContentProtectionType{cp(addressing.SchemeMP4Protection, "cenc", otherKID)}, []string{"XCHK-CP-SCHEME", "XCHK-CP-KID"}},
	}
	for _, c := range cases {
		video := *reps(t)[1]
		video.ContentProtections = c.cps
		_, f := CheckInit(buf.Bytes(), &video, video.Loc())
		if got := ruleIDs(f); !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, f, c.want)
		}
	}
}
