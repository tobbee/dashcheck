package segcheck

import (
	"bytes"
	"testing"

	"github.com/Eyevinn/mp4ff/bits"
	"github.com/Eyevinn/mp4ff/mp4"
)

// tocBytes writes an ac4_toc start: bitstream_version 2, three presentations
// and short_program_id 0x1234.
func tocBytes() []byte {
	var buf bytes.Buffer
	w := bits.NewWriter(&buf)
	for _, f := range []struct{ v, n uint }{
		{2, 2}, {1, 10}, {0, 1}, // bitstream_version, sequence_counter, b_wait_frames
		{1, 1}, {4, 4}, {1, 1}, // fs_index, frame_rate_index, b_iframe_global
		{0, 1}, {1, 1}, {1, 2}, {0, 1}, // b_single_presentation, b_more_presentations, variable_bits(2) = 1
		{0, 1}, {1, 1}, {0x1234, 16}, {0, 1}, // b_payload_base, b_program_id, short_program_id, b_program_uuid_present
	} {
		w.Write(f.v, int(f.n))
	}
	w.Flush()
	return append(buf.Bytes(), 0, 0, 0, 0)
}

func TestAC4TOC(t *testing.T) {
	toc, err := parseAC4TOC(tocBytes())
	want := ac4TOC{bitstreamVersion: 2, fsIndex: 1, frameRateIndex: 4, nPresentations: 3, hasProgramID: true, shortProgramID: 0x1234}
	if err != nil || toc != want {
		t.Fatalf("got %+v, %v", toc, err)
	}
	for _, c := range []struct {
		desc string
		dac4 mp4.Dac4Box
		bad  bool
	}{
		{"matching", mp4.Dac4Box{AC4DSIVersion: 1, BitstreamVersion: 2, FSIndex: 1, FrameRateIndex: 4, NPresentations: 3, BProgramID: 1, ShortProgramID: 0x1234}, false},
		{"other frame rate", mp4.Dac4Box{AC4DSIVersion: 1, BitstreamVersion: 2, FSIndex: 1, FrameRateIndex: 2, NPresentations: 3, BProgramID: 1, ShortProgramID: 0x1234}, true},
		{"no program id", mp4.Dac4Box{AC4DSIVersion: 1, BitstreamVersion: 2, FSIndex: 1, FrameRateIndex: 4, NPresentations: 3}, true},
	} {
		if got := compareAC4TOC(&c.dac4, tocBytes()); (got != "") != c.bad {
			t.Errorf("%s: got %q", c.desc, got)
		}
	}
}
