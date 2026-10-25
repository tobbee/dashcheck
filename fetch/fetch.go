// Package fetch abstracts HTTP access so that the checks run unchanged
// natively (net/http, local files) and in the browser (fetch API via wasm).
package fetch

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Request describes one GET or HEAD request.
type Request struct {
	URL  string
	Head bool
	// Range is a byte range "start-end" (inclusive), or empty for the full resource.
	Range string
}

// Response is the result of a request.
// In the browser, Header only holds headers exposed by CORS.
type Response struct {
	URL    string            // final URL after redirects
	Status int               // HTTP status code
	Header map[string]string // lower-case header names
	Body   []byte
	Start  time.Time // local wall clock when the request was sent
	End    time.Time // local wall clock when the body was fully received
}

// OK reports whether the status is 200 or 206.
func (r *Response) OK() bool {
	return r.Status == 200 || r.Status == 206
}

// Fetcher performs requests.
type Fetcher interface {
	Fetch(ctx context.Context, req Request) (*Response, error)
}

// StatusError is returned by Get for non-OK responses.
type StatusError struct {
	URL    string
	Status int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("GET %s: status %d", e.URL, e.Status)
}

// Get fetches url and returns an error for non-OK responses.
func Get(ctx context.Context, f Fetcher, url string) (*Response, error) {
	resp, err := f.Fetch(ctx, Request{URL: url})
	if err != nil {
		return nil, err
	}
	if !resp.OK() {
		return resp, &StatusError{URL: url, Status: resp.Status}
	}
	return resp, nil
}

// CutRange returns the part of a whole resource that the byte range
// "first-last" or "first-" selects, and the status of a server that answers
// the request: 206, or 416 if the range starts after the end. Like HTTP, a
// range that ends after the resource is cut at its end.
func CutRange(data []byte, r string) ([]byte, int, error) {
	a, b, ok := strings.Cut(r, "-")
	first, err := strconv.Atoi(a)
	if !ok || err != nil || first < 0 {
		return nil, 0, fmt.Errorf("bad range %q", r)
	}
	last := len(data) - 1
	if b != "" {
		if last, err = strconv.Atoi(b); err != nil || first > last {
			return nil, 0, fmt.Errorf("bad range %q", r)
		}
	}
	if first >= len(data) {
		return nil, 416, nil
	}
	return data[first : min(last, len(data)-1)+1], 206, nil
}
