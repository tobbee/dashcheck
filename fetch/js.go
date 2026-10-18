//go:build js && wasm

package fetch

import (
	"context"
	"errors"
	"fmt"
	"syscall/js"
	"time"
)

// Browser fetches with the browser's fetch API.
// It avoids net/http, which would add about 7 MB to the wasm binary,
// and it can set cache: 'no-store', which net/http under js cannot.
type Browser struct{}

func (Browser) Fetch(ctx context.Context, req Request) (*Response, error) {
	ac := js.Global().Get("AbortController").New()
	stop := context.AfterFunc(ctx, func() { ac.Call("abort") })
	defer stop()

	opts := map[string]any{
		"method": "GET",
		"cache":  "no-store",
		"signal": ac.Get("signal"),
	}
	if req.Head {
		opts["method"] = "HEAD"
	}
	if req.Range != "" {
		opts["headers"] = map[string]any{"Range": "bytes=" + req.Range}
	}
	start := time.Now()
	resp, err := await(js.Global().Call("fetch", req.URL, opts))
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", req.URL, err)
	}
	header := map[string]string{}
	forEach := js.FuncOf(func(this js.Value, args []js.Value) any {
		header[args[1].String()] = args[0].String()
		return nil
	})
	resp.Get("headers").Call("forEach", forEach)
	forEach.Release()

	finalURL := resp.Get("url").String()
	if finalURL == "" {
		finalURL = req.URL // constructed responses, e.g. for local files, have no URL
	}
	out := &Response{
		URL:    finalURL,
		Status: resp.Get("status").Int(),
		Header: header,
		Start:  start,
	}
	if !req.Head {
		buf, err := await(resp.Call("arrayBuffer"))
		if err != nil {
			return nil, fmt.Errorf("read body of %s: %w", req.URL, err)
		}
		arr := js.Global().Get("Uint8Array").New(buf)
		out.Body = make([]byte, arr.Get("length").Int())
		js.CopyBytesToGo(out.Body, arr)
	}
	out.End = time.Now()
	return out, nil
}

// await blocks until the promise settles. It must not be called from
// inside a js.FuncOf callback, only from a goroutine.
func await(p js.Value) (js.Value, error) {
	type result struct {
		v   js.Value
		err error
	}
	ch := make(chan result, 1)
	onOK := js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- result{v: args[0]}
		return nil
	})
	onErr := js.FuncOf(func(this js.Value, args []js.Value) any {
		msg := "promise rejected"
		if len(args) > 0 && !args[0].IsUndefined() {
			msg = args[0].Call("toString").String()
		}
		ch <- result{err: errors.New(msg)}
		return nil
	})
	defer onOK.Release()
	defer onErr.Release()
	p.Call("then", onOK, onErr)
	r := <-ch
	return r.v, r.err
}
