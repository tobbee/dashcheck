package main

import (
	"bytes"
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
