package mpdcheck

import (
	"encoding/base64"
	"fmt"
	"strings"

	mp4bits "github.com/Eyevinn/mp4ff/bits"
	"github.com/tobbee/dashcheck/report"
)

// checkSpliceInfo checks that data is an SCTE-35 splice_info_section (ANSI/SCTE
// 35 9.6): table_id 0xFC, a section_length that fits in the data, and a
// splice_command_length (0xFFF if unknown) that leaves room for the 14 bytes
// of fields from protocol_version to splice_command_type, the
// descriptor_loop_length and the CRC_32.
func checkSpliceInfo(section []byte) error {
	r := mp4bits.NewReaderFromSlice(section)
	id := r.Read(8)
	r.Read(4) // section_syntax_indicator, private_indicator, sap_type
	length := int(r.Read(12))
	r.Read(8 + 1 + 6 + 33 + 8 + 12) // protocol_version to tier
	command := int(r.Read(12))
	if err := r.AccError(); err != nil {
		return fmt.Errorf("truncated header: %w", err)
	}
	switch {
	case id != 0xfc:
		return fmt.Errorf("table_id 0x%02x, not 0xfc", id)
	case length+3 > len(section):
		return fmt.Errorf("section_length %d, but only %d bytes", length, len(section)-3)
	case command != 0xfff && command > length-17:
		return fmt.Errorf("splice_command_length %d exceeds the section", command)
	}
	return nil
}

// scteBinaryRules checks that an SCTE-35 event carries a base64
// splice_info_section in Signal/Binary (DVB 9.1.9). Its splice time is an
// MPEG-2 TS PTS, and the MPD does not say how that maps to the media
// timeline, so it cannot be compared with @presentationTime.
func scteBinaryRules(ev *rawElem, loc report.Location) []report.Finding {
	var bin *rawElem
	for _, s := range ev.children("Signal") {
		if b := s.children("Binary"); len(b) > 0 {
			bin = b[0]
			break
		}
	}
	if bin == nil {
		return []report.Finding{RuleDVBSCTEBinary.New(loc, "SCTE-35 event without a Signal/Binary element")}
	}
	section, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(bin.Text), ""))
	if err != nil {
		return []report.Finding{RuleDVBSCTEBinary.New(loc, "Binary is not base64: %v", err)}
	}
	if err := checkSpliceInfo(section); err != nil {
		return []report.Finding{RuleDVBSCTEBinary.New(loc, "splice_info_section: %v", err)}
	}
	return nil
}
