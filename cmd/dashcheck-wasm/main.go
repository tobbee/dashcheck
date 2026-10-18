//go:build js && wasm

// Command dashcheck-wasm exposes dashcheck to JavaScript. Run it in a Web
// Worker; it sets a global object
//
//	dashcheck.check(mpdURL, optionsJSON, onProgress) -> Promise<reportJSON>
//	dashcheck.checkData(mpdBytes, mpdURL, optionsJSON, onProgress) -> Promise<reportJSON>
//	dashcheck.rules() -> rulesJSON
//
// where onProgress receives progress objects as JSON strings.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/tobbee/dashcheck"
	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/report"
)

// jsOptions are the options accepted from JavaScript.
type jsOptions struct {
	MaxSegments  int    `json:"maxSegments"`
	MPDUpdates   int    `json:"mpdUpdates"`
	Reference    string `json:"reference"`
	Concurrency  int    `json:"concurrency"`
	LiveMarginMS int    `json:"liveMarginMs"`
}

func main() {
	js.Global().Set("dashcheck", js.ValueOf(map[string]any{
		"check":     js.FuncOf(check),
		"checkData": js.FuncOf(checkData),
		"rules":     js.FuncOf(rules),
	}))
	select {}
}

func options(optsJSON string, onProgress js.Value) (dashcheck.Options, error) {
	var jo jsOptions
	if optsJSON != "" {
		if err := json.Unmarshal([]byte(optsJSON), &jo); err != nil {
			return dashcheck.Options{}, fmt.Errorf("options: %w", err)
		}
	}
	opts := dashcheck.Options{
		Fetcher:     fetch.Browser{},
		MaxSegments: jo.MaxSegments,
		MPDUpdates:  jo.MPDUpdates,
		Reference:   jo.Reference,
		Concurrency: jo.Concurrency,
		LiveMargin:  time.Duration(jo.LiveMarginMS) * time.Millisecond,
	}
	if onProgress.Type() == js.TypeFunction {
		opts.Progress = func(p dashcheck.Progress) {
			b, _ := json.Marshal(p)
			onProgress.Invoke(string(b))
		}
	}
	return opts, nil
}

// check(mpdURL, optionsJSON, onProgress)
func check(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return rejected("check needs an MPD URL")
	}
	mpdURL := args[0].String()
	optsJSON, onProgress := optArgs(args, 1)
	return promise(func() (any, error) {
		opts, err := options(optsJSON, onProgress)
		if err != nil {
			return nil, err
		}
		return marshal(dashcheck.Check(context.Background(), mpdURL, opts))
	})
}

// checkData(mpdBytes, mpdURL, optionsJSON, onProgress)
func checkData(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return rejected("checkData needs MPD bytes and a URL")
	}
	data := make([]byte, args[0].Get("length").Int())
	js.CopyBytesToGo(data, args[0])
	mpdURL := args[1].String()
	optsJSON, onProgress := optArgs(args, 2)
	return promise(func() (any, error) {
		opts, err := options(optsJSON, onProgress)
		if err != nil {
			return nil, err
		}
		return marshal(dashcheck.CheckData(context.Background(), data, mpdURL, opts))
	})
}

func rules(this js.Value, args []js.Value) any {
	b, _ := json.Marshal(report.Rules())
	return string(b)
}

func optArgs(args []js.Value, i int) (string, js.Value) {
	optsJSON := ""
	onProgress := js.Undefined()
	if len(args) > i && args[i].Type() == js.TypeString {
		optsJSON = args[i].String()
	}
	if len(args) > i+1 {
		onProgress = args[i+1]
	}
	return optsJSON, onProgress
}

func marshal(rpt *report.Report) (any, error) {
	b, err := json.Marshal(rpt)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// promise runs fn in a goroutine, since blocking calls such as fetch
// must not run inside a js.FuncOf callback.
func promise(fn func() (any, error)) js.Value {
	var executor js.Func
	executor = js.FuncOf(func(this js.Value, args []js.Value) any {
		resolve, reject := args[0], args[1]
		go func() {
			defer executor.Release()
			v, err := fn()
			if err != nil {
				reject.Invoke(js.Global().Get("Error").New(err.Error()))
				return
			}
			resolve.Invoke(v)
		}()
		return nil
	})
	return js.Global().Get("Promise").New(executor)
}

func rejected(msg string) js.Value {
	return js.Global().Get("Promise").Call("reject", js.Global().Get("Error").New(msg))
}
