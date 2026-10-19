package mpdcheck

import (
	"encoding/base64"
	"testing"
)

func TestCheckSpliceInfo(t *testing.T) {
	cases := []struct {
		name, b64 string
		ok        bool
	}{
		// Sample sections from ANSI/SCTE 35 14.
		{"time_signal", "/DA0AAAAAAAA///wBQb+cr0AUAAeAhxDVUVJSAAAjn/PAAGlmbAICAAAAAAsoKGKNAIAmsnRfg==", true},
		{"splice_insert", "/DAvAAAAAAAA///wFAVIAACPf+/+c2nALv4AUsz1AAAAAAAKAAhDVUVJAAABNWLbowo=", true},
		// An immediate splice_insert as written by Unified Streaming.
		{"immediate splice_insert", "/DAgAAAAAAAAAP/wDwUA4hzJf//+ADS8AMAAAAAAAORhJCQ=", true},
		{"truncated", "/DA0AAAAAAAA///wBQb+cr0AUAAeAhxDVUVJ", false},
		{"other table", "/TA0AAAAAAAA///wBQb+cr0AUAAeAhxDVUVJSAAAjn/PAAGlmbAICAAAAAAsoKGKNAIAmsnRfg==", false},
		{"too short", "/DA=", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data, err := base64.StdEncoding.DecodeString(c.b64)
			if err != nil {
				t.Fatal(err)
			}
			if err := checkSpliceInfo(data); (err == nil) != c.ok {
				t.Errorf("got %v, want ok %v", err, c.ok)
			}
		})
	}
}
