package segcheck

import (
	"fmt"
	"strings"

	"github.com/Eyevinn/mp4ff/bits"
	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/tobbee/dashcheck/report"
)

// AC-4 checks ported from the Dolby module of the DASH-IF Conformance tool
// (DOLBY-1 to DOLBY-5): the dac4 box must repeat values of the AC-4 TOC.

var RuleAC4TOC = report.Register(report.Rule{
	ID: "SEG-AC4-TOC", Severity: report.Error,
	Desc: "The dac4 box must match the TOC of the AC-4 frames: bitstream_version, fs_index, frame_rate_index, " +
		"n_presentations and short_program_id",
	SpecRef: "ETSI TS 103 190-2 E.6.3 to E.6.7",
})

// ac4TOC holds the start of an ac4_toc() (ETSI TS 103 190-2 6.2.1.1).
type ac4TOC struct {
	bitstreamVersion, fsIndex, frameRateIndex, nPresentations int
	hasProgramID                                              bool
	shortProgramID                                            int
}

// variableBits reads variable_bits(n) of ETSI TS 103 190-1 4.2.2.
func variableBits(r *bits.Reader, n int) int {
	value := 0
	for {
		value += int(r.Read(n))
		if !r.ReadFlag() || r.AccError() != nil {
			return value
		}
		value = (value << n) + (1 << n)
	}
}

// parseAC4TOC reads the TOC at the start of a raw_ac4_frame, as AC-4
// samples are stored in ISOBMFF.
func parseAC4TOC(sample []byte) (ac4TOC, error) {
	r := bits.NewReaderFromSlice(sample)
	var t ac4TOC
	t.bitstreamVersion = int(r.Read(2))
	if t.bitstreamVersion == 3 {
		t.bitstreamVersion += variableBits(r, 2)
	}
	r.Read(10)        // sequence_counter
	if r.ReadFlag() { // b_wait_frames
		if r.Read(3) > 0 {
			r.Read(2) // br_code
		}
	}
	t.fsIndex = int(r.Read(1))
	t.frameRateIndex = int(r.Read(4))
	r.Read(1) // b_iframe_global
	switch {
	case r.ReadFlag(): // b_single_presentation
		t.nPresentations = 1
	case r.ReadFlag(): // b_more_presentations
		t.nPresentations = variableBits(r, 2) + 2
	}
	if r.ReadFlag() { // b_payload_base
		if r.Read(5)+1 == 0x20 {
			variableBits(r, 3)
		}
	}
	if t.bitstreamVersion > 1 && r.ReadFlag() { // b_program_id
		t.hasProgramID = true
		t.shortProgramID = int(r.Read(16))
	}
	return t, r.AccError()
}

// ac4MediaRules compares the TOC of the first AC-4 frame of a segment with
// the dac4 box (DOLBY-1 to DOLBY-5).
func ac4MediaRules(init *Init, mf *mp4.File, loc report.Location) []report.Finding {
	d := init.dac4
	if d == nil || init.SampleEntry != "ac-4" || d.AC4DSIVersion > 1 {
		return nil
	}
	var sample []byte
	for _, ms := range mf.Segments {
		for _, frag := range ms.Fragments {
			if samples, err := frag.GetFullSamples(init.trex); err == nil && len(samples) > 0 && sample == nil {
				sample = samples[0].Data
			}
		}
	}
	if len(sample) == 0 {
		return nil
	}
	if init.Encrypted {
		return nil // the TOC may be encrypted
	}
	if problem := compareAC4TOC(d, sample); problem != "" {
		return []report.Finding{RuleAC4TOC.New(loc, "%s", problem)}
	}
	return nil
}

// compareAC4TOC compares a dac4 box with the TOC of an AC-4 frame.
func compareAC4TOC(d *mp4.Dac4Box, sample []byte) string {
	t, err := parseAC4TOC(sample)
	if err != nil {
		return fmt.Sprintf("cannot read the AC-4 TOC: %v", err)
	}
	var diffs []string
	add := func(name string, dac4, toc int) {
		if dac4 != toc {
			diffs = append(diffs, fmt.Sprintf("%s %d in dac4, %d in the TOC", name, dac4, toc))
		}
	}
	add("bitstream_version", int(d.BitstreamVersion), t.bitstreamVersion)
	add("fs_index", int(d.FSIndex), t.fsIndex)
	add("frame_rate_index", int(d.FrameRateIndex), t.frameRateIndex)
	add("n_presentations", int(d.NPresentations), t.nPresentations)
	if t.bitstreamVersion > 1 {
		switch {
		case (d.BProgramID == 1) != t.hasProgramID:
			diffs = append(diffs, fmt.Sprintf("b_program_id %d in dac4, %v in the TOC", d.BProgramID, t.hasProgramID))
		case t.hasProgramID:
			add("short_program_id", int(d.ShortProgramID), t.shortProgramID)
		}
	}
	return strings.Join(diffs, "; ")
}
