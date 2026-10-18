// Package fetch abstracts HTTP access so that the checks run unchanged
// natively (net/http, local files) and in the browser (fetch API via wasm).
package fetch

import (
	"context"
	"fmt"
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
