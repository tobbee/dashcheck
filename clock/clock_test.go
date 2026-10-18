package clock

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/fetch"
)

// fakeFetcher serves times that are offset from the local clock.
type fakeFetcher map[string]time.Duration

func (ff fakeFetcher) Fetch(ctx context.Context, req fetch.Request) (*fetch.Response, error) {
	off, ok := ff[req.URL]
	now := time.Now()
	if !ok {
		return &fetch.Response{URL: req.URL, Status: 404, Start: now, End: now}, nil
	}
	body := now.Add(off).UTC().Format("2006-01-02T15:04:05.000Z")
	return &fetch.Response{URL: req.URL, Status: 200, Body: []byte(body), Start: now, End: now}, nil
}

func TestSync(t *testing.T) {
	ff := fakeFetcher{
		"https://ref":    3 * time.Second, // the local clock is 3 s behind
		"https://good":   3 * time.Second,
		"https://skewed": 5 * time.Second,
	}
	timings := []*m.DescriptorType{
		{SchemeIdUri: "urn:mpeg:dash:utc:http-xsdate:2014", Value: "https://missing"},
		{SchemeIdUri: "urn:mpeg:dash:utc:http-iso:2014", Value: "https://skewed"},
		{SchemeIdUri: "urn:mpeg:dash:utc:ntp:2014", Value: "time.example.com"},
		{SchemeIdUri: "urn:mpeg:dash:utc:http-iso:2014", Value: "https://good"},
	}
	clk, source, findings := Sync(context.Background(), ff, timings, "https://ref", nil)
	var ids []string
	for _, f := range findings {
		ids = append(ids, f.Rule)
	}
	want := []string{"CLOCK-SOURCE", "CLOCK-UNSUPPORTED", "CLOCK-LOCAL", "CLOCK-SKEW"}
	if !slices.Equal(ids, want) {
		t.Errorf("got %v, want %v: %v", ids, want, findings)
	}
	// The first working source is followed, as a player would.
	if source != "https://skewed" {
		t.Errorf("source %s", source)
	}
	if d := clk.Now().Sub(time.Now()); d < 4900*time.Millisecond || d > 5100*time.Millisecond {
		t.Errorf("clock offset %v", d)
	}

	// No MPD sources: follow the reference.
	clk, source, _ = Sync(context.Background(), ff, nil, "https://ref", nil)
	if source != "https://ref" || clk.Now().Sub(time.Now()) < 2900*time.Millisecond {
		t.Errorf("fallback: %s %v", source, clk.Now().Sub(time.Now()))
	}
}

// countingFetcher counts requests per URL.
type countingFetcher struct {
	fakeFetcher
	calls map[string]int
}

func (cf *countingFetcher) Fetch(ctx context.Context, req fetch.Request) (*fetch.Response, error) {
	cf.calls[req.URL]++
	return cf.fakeFetcher.Fetch(ctx, req)
}

func TestSyncReusesReference(t *testing.T) {
	cf := &countingFetcher{fakeFetcher{"https://ref": -45 * time.Millisecond}, map[string]int{}}
	timings := []*m.DescriptorType{{SchemeIdUri: "urn:mpeg:dash:utc:http-xsdate:2014", Value: "https://ref"}}
	_, source, findings := Sync(context.Background(), cf, timings, "https://ref", nil)
	if cf.calls["https://ref"] != 1 {
		t.Errorf("reference fetched %d times, want 1", cf.calls["https://ref"])
	}
	if source != "https://ref" || len(findings) != 1 || findings[0].Rule != "CLOCK-LOCAL" {
		t.Fatalf("source %s, findings %v", source, findings)
	}
	// The fake server truncates to milliseconds, so the offset may read 45 or 46 ms.
	msg := findings[0].Message
	if !strings.HasPrefix(msg, "this computer's clock is 4") || !strings.Contains(msg, "ms ahead of the reference clock ref (round trip") {
		t.Errorf("message %q", msg)
	}
}
