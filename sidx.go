package dashcheck

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/fetch"
	"github.com/tobbee/dashcheck/segcheck"
)

// sidxSegments reads the sidx box of a SegmentBase Representation and returns
// its subsegments as byte ranges. It switches the Representation's timescale
// to that of the sidx box.
func (rc *repChecker) sidxSegments(ctx context.Context) ([]addressing.Segment, error) {
	rep := rc.rep
	first, _, err := parseRange(rep.IndexRange)
	if err != nil {
		return nil, fmt.Errorf("@indexRange: %w", err)
	}
	resp, err := rc.opts.Fetcher.Fetch(ctx, fetch.Request{URL: rep.BaseURL, Range: rep.IndexRange})
	if err != nil {
		return nil, err
	}
	if !resp.OK() {
		return nil, fmt.Errorf("index range %s of %s: status %d", rep.IndexRange, rep.BaseURL, resp.Status)
	}
	box, err := mp4.DecodeBox(first, bytes.NewReader(resp.Body))
	if err != nil {
		return nil, fmt.Errorf("index range %s: %w", rep.IndexRange, err)
	}
	sidx, ok := box.(*mp4.SidxBox)
	if !ok {
		return nil, fmt.Errorf("index range %s holds %s, not sidx", rep.IndexRange, box.Type())
	}
	if segcheck.IsIOPOnDemand(rep) {
		if n := countSidx(first, resp.Body); n != 1 {
			rc.add(segcheck.RuleIOPSidx.New(rep.Loc(), "%d sidx boxes in @indexRange %s", n, rep.IndexRange))
		}
	}
	if sidx.Timescale == 0 {
		return nil, fmt.Errorf("sidx timescale is 0")
	}
	// Express times in the sidx timescale.
	rep.PTO = rep.PTO * uint64(sidx.Timescale) / uint64(rep.Timescale)
	rep.Timescale = sidx.Timescale

	offset := first + box.Size() + sidx.FirstOffset
	t := sidx.EarliestPresentationTime
	segs := make([]addressing.Segment, 0, len(sidx.SidxRefs))
	for i, ref := range sidx.SidxRefs {
		if ref.ReferenceType == 1 {
			return nil, fmt.Errorf("hierarchical sidx (reference %d points to another sidx) is not supported", i)
		}
		end := offset + uint64(ref.ReferencedSize) - 1
		segs = append(segs, addressing.Segment{
			Number: uint64(i + 1),
			Time:   t,
			Dur:    uint64(ref.SubSegmentDuration),
			URL:    rep.BaseURL,
			Range:  fmt.Sprintf("%d-%d", offset, end),
		})
		offset = end + 1
		t += uint64(ref.SubSegmentDuration)
	}
	return segs, nil
}

// countSidx counts the sidx boxes among the top-level boxes in data.
func countSidx(startPos uint64, data []byte) int {
	n := 0
	r := bytes.NewReader(data)
	pos := startPos
	for r.Len() > 0 {
		box, err := mp4.DecodeBox(pos, r)
		if err != nil {
			break
		}
		if box.Type() == "sidx" {
			n++
		}
		pos += box.Size()
	}
	return n
}

func parseRange(r string) (first, last uint64, err error) {
	a, b, ok := strings.Cut(r, "-")
	if !ok {
		return 0, 0, fmt.Errorf("bad range %q", r)
	}
	if first, err = strconv.ParseUint(a, 10, 64); err != nil {
		return 0, 0, fmt.Errorf("bad range %q", r)
	}
	if last, err = strconv.ParseUint(b, 10, 64); err != nil {
		return 0, 0, fmt.Errorf("bad range %q", r)
	}
	return first, last, nil
}
