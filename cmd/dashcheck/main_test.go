package main

import (
	"bytes"
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
