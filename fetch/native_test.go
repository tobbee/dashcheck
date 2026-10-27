//go:build !js

package fetch

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my stream #1 50%.mpd")
	if err := os.WriteFile(path, []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	fileURL := FileURL(path)
	if !strings.HasPrefix(fileURL, "file:///") || strings.Contains(fileURL, `\`) {
		t.Errorf("FileURL(%q) = %q", path, fileURL)
	}
	u, err := url.Parse(fileURL)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "" || filePath(u) != path {
		t.Errorf("%s: host %q, path %q, want %q", fileURL, u.Host, filePath(u), path)
	}

	n := NewNative(0)
	cases := []struct {
		name   string
		req    Request
		status int
		body   string
	}{
		{"whole file", Request{URL: fileURL}, 200, "0123456789"},
		{"range", Request{URL: fileURL, Range: "2-4"}, 206, "234"},
		{"head", Request{URL: fileURL, Head: true}, 200, ""},
		{"segment", Request{URL: FileURL(filepath.Join(dir, "seg.m4s"))}, 404, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, err := n.Fetch(context.Background(), c.req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Status != c.status || string(resp.Body) != c.body {
				t.Errorf("got %d %q, want %d %q", resp.Status, resp.Body, c.status, c.body)
			}
		})
	}
}
