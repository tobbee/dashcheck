//go:build !js

package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Native fetches http(s) URLs with net/http and file URLs from the local file system.
type Native struct {
	Client *http.Client
}

// NewNative returns a Native fetcher with the given per-request timeout.
func NewNative(timeout time.Duration) *Native {
	return &Native{Client: &http.Client{Timeout: timeout}}
}

func (n *Native) Fetch(ctx context.Context, req Request) (*Response, error) {
	if strings.HasPrefix(req.URL, "file:") {
		return fetchFile(req)
	}
	method := http.MethodGet
	if req.Head {
		method = http.MethodHead
	}
	hreq, err := http.NewRequestWithContext(ctx, method, req.URL, nil)
	if err != nil {
		return nil, err
	}
	if req.Range != "" {
		hreq.Header.Set("Range", "bytes="+req.Range)
	}
	start := time.Now()
	hresp, err := n.Client.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer hresp.Body.Close()
	body, err := io.ReadAll(hresp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body of %s: %w", req.URL, err)
	}
	return &Response{
		URL:    hresp.Request.URL.String(),
		Status: hresp.StatusCode,
		Header: lowerKeys(hresp.Header),
		Body:   body,
		Start:  start,
		End:    time.Now(),
	}, nil
}

func fetchFile(req Request) (*Response, error) {
	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	data, err := os.ReadFile(u.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Response{URL: req.URL, Status: 404, Start: start, End: time.Now()}, nil
		}
		return nil, err
	}
	status := 200
	if req.Range != "" {
		if data, status, err = CutRange(data, req.Range); err != nil {
			return nil, err
		}
	}
	if req.Head {
		data = nil
	}
	return &Response{URL: req.URL, Status: status, Body: data, Start: start, End: time.Now()}, nil
}

func lowerKeys(h map[string][]string) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		out[strings.ToLower(k)] = strings.Join(v, ", ")
	}
	return out
}
