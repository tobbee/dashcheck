package dashcheck

import (
	"bytes"
	"context"
	"encoding/binary"
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
	resp, err := rc.fetchRange(ctx, fetch.Request{URL: rep.BaseURL, Range: rep.IndexRange})
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
	rc.sidx, rc.nSidx = sidx, countSidx(first, resp.Body)
	if segcheck.IsIOPOnDemand(rep) && rc.nSidx != 1 {
		rc.add(segcheck.RuleIOPSidx.New(rep.Loc(), "%d sidx boxes in @indexRange %s", rc.nSidx, rep.IndexRange))
	}
	if sidx.Timescale == 0 {
		return nil, fmt.Errorf("sidx timescale is 0")
	}
	// Express times in the sidx timescale.
	rep.PTO = rep.PTO * uint64(sidx.Timescale) / uint64(rep.Timescale)
	rep.Timescale = sidx.Timescale

	offset := first + box.Size() + sidx.FirstOffset
	rc.gapStart, rc.gapEnd = first+box.Size(), offset
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
	rc.indexEnd = offset
	return segs, nil
}

// maxProbedBoxes limits the box headers that checkSidxSpan reads after the
// last subsegment.
const maxProbedBoxes = 16

// checkSidxSpan looks for movie fragments that the sidx box does not index:
// between the sidx box and its first subsegment, and after the last one.
func (rc *repChecker) checkSidxSpan(ctx context.Context) {
	rep := rc.rep
	n := len(rc.sidx.SidxRefs)
	if rc.gapStart < rc.gapEnd {
		if pos, ok := rc.findMoof(ctx, rc.gapStart, rc.gapEnd); ok {
			rc.add(segcheck.RuleSidxSpan.New(rep.Loc(), "moof box at byte %d, before the first of %d subsegments at byte %d", pos, n, rc.gapEnd))
		}
	}
	if pos, ok := rc.findMoof(ctx, rc.indexEnd, 0); ok {
		rc.add(segcheck.RuleSidxSpan.New(rep.Loc(), "moof box at byte %d, after the last of %d subsegments", pos, n))
	}
}

// findMoof reads the top-level box headers of the Representation's file from
// pos up to end (0 for the end of the file) with small range requests, and
// returns the position of the first moof box.
func (rc *repChecker) findMoof(ctx context.Context, pos, end uint64) (uint64, bool) {
	for range maxProbedBoxes {
		if end != 0 && pos >= end {
			return 0, false
		}
		resp, err := rc.fetchRange(ctx, fetch.Request{URL: rc.rep.BaseURL, Range: fmt.Sprintf("%d-%d", pos, pos+15)})
		if err != nil || !resp.OK() { // 416: pos is at the end of the file
			return 0, false
		}
		hdr := resp.Body
		if len(hdr) < 8 {
			return 0, false
		}
		if string(hdr[4:8]) == "moof" {
			return pos, true
		}
		size := uint64(binary.BigEndian.Uint32(hdr))
		switch {
		case size == 0: // the box extends to the end of the file
			return 0, false
		case size == 1 && len(hdr) >= 16:
			size = binary.BigEndian.Uint64(hdr[8:16])
		}
		if size < 8 {
			return 0, false
		}
		pos += size
	}
	return 0, false
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
