package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tobbee/dashcheck/internal"
)

func TestVersionFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code %d, stderr %q", code, stderr.String())
	}
	if got, want := stdout.String(), "dashcheck "+internal.Version()+"\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBadProfiles(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-profiles", "dvb3000", "Manifest.mpd"}, &stdout, &stderr); code != 2 {
		t.Errorf("exit code %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), `unknown profile "dvb3000"`) {
		t.Errorf("stderr %q", stderr.String())
	}
}

const (
	goodMPD = "../../testdata/testpic_2s/Manifest.mpd"
	badMPD  = "../../testdata/malformed.mpd"
)

func TestShort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{badMPD, goodMPD}, &stdout, &stderr); code != 2 {
		t.Errorf("several MPDs without -short: exit code %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "several MPDs need -short") {
		t.Errorf("stderr %q", stderr.String())
	}

	stdout.Reset()
	if code := run([]string{"-short", badMPD, goodMPD}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code %d, want 1; stderr %q", code, stderr.String())
	}
	want := "FAIL  " + badMPD + "  (3 errors, 0 warnings, 0 info)\n" +
		"OK    " + goodMPD + "  (0 errors, 0 warnings, 0 info)\n" +
		"\n2 MPDs: 1 with errors, 0 with warnings, 1 OK\n"
	if got := stdout.String(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}

	// -level adds the findings, and one MPD has no total line.
	stdout.Reset()
	if code := run([]string{"-short", "-level", "error", badMPD}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code %d, want 1", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 4 || !strings.HasPrefix(lines[1], "      error   MPD-XML:") {
		t.Errorf("got\n%s", stdout.String())
	}
}

func TestShortJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	run([]string{"-short", "-json", goodMPD, badMPD}, &stdout, &stderr)
	var reports []struct {
		MPDURL   string            `json:"mpdURL"`
		Findings []json.RawMessage `json:"findings"`
		Counts   struct {
			Errors int `json:"errors"`
		} `json:"counts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &reports); err != nil {
		t.Fatalf("%v\n%s", err, stdout.String())
	}
	if len(reports) != 2 || !strings.HasSuffix(reports[0].MPDURL, "Manifest.mpd") ||
		reports[0].Counts.Errors != 0 || reports[1].Counts.Errors != 3 ||
		reports[1].Findings == nil || len(reports[1].Findings) != 0 {
		t.Errorf("reports %+v", reports)
	}
}
