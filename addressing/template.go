package addressing

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrSegmentSequences reports Segment Sequences (ISO/IEC 23009-1 5.3.9.7:
// @k and $SubNumber$), which dashcheck cannot address yet.
var ErrSegmentSequences = errors.New("segment sequences (@k, $SubNumber$) are not supported yet")

// Values are the substitutions for SegmentTemplate identifiers.
type Values struct {
	RepresentationID string
	Number           uint64
	Time             uint64
	Bandwidth        uint32
}

// Substitute replaces the identifiers $RepresentationID$, $Number$, $Time$,
// $Bandwidth$ and $$ in a SegmentTemplate string (ISO/IEC 23009-1 5.3.9.4.4).
// $Number$, $Time$ and $Bandwidth$ may carry a format tag like %05d.
func Substitute(tmpl string, v Values) (string, error) {
	var b strings.Builder
	rest := tmpl
	for {
		i := strings.IndexByte(rest, '$')
		if i < 0 {
			b.WriteString(rest)
			return b.String(), nil
		}
		b.WriteString(rest[:i])
		rest = rest[i+1:]
		j := strings.IndexByte(rest, '$')
		if j < 0 {
			return "", fmt.Errorf("unterminated identifier in %q", tmpl)
		}
		ident := rest[:j]
		rest = rest[j+1:]
		if ident == "" {
			b.WriteByte('$')
			continue
		}
		name, format, _ := strings.Cut(ident, "%")
		var val uint64
		switch name {
		case "RepresentationID":
			if format != "" {
				return "", fmt.Errorf("format tag not allowed on $RepresentationID$ in %q", tmpl)
			}
			b.WriteString(v.RepresentationID)
			continue
		case "Number":
			val = v.Number
		case "Time":
			val = v.Time
		case "Bandwidth":
			val = uint64(v.Bandwidth)
		case "SubNumber":
			return "", ErrSegmentSequences
		default:
			return "", fmt.Errorf("unknown identifier $%s$ in %q", ident, tmpl)
		}
		if format == "" {
			b.WriteString(strconv.FormatUint(val, 10))
			continue
		}
		s, err := formatTag(format, val)
		if err != nil {
			return "", fmt.Errorf("%v in %q", err, tmpl)
		}
		b.WriteString(s)
	}
}

// formatTag formats val according to a format tag (without the leading %),
// e.g. "05d". The standard defines only %0[width]d, but x, X, o, i and u are
// accepted by common players.
func formatTag(tag string, val uint64) (string, error) {
	if tag == "" {
		return "", fmt.Errorf("empty format tag")
	}
	verb := tag[len(tag)-1]
	width := tag[:len(tag)-1]
	switch verb {
	case 'd', 'i', 'u':
		verb = 'd'
	case 'x', 'X', 'o':
	default:
		return "", fmt.Errorf("unsupported format tag %%%s", tag)
	}
	for _, c := range width {
		if c < '0' || c > '9' {
			return "", fmt.Errorf("unsupported format tag %%%s", tag)
		}
	}
	if width != "" && width[0] != '0' {
		width = "0" + width
	}
	return fmt.Sprintf("%"+width+string(verb), val), nil
}

// containsID reports whether tmpl contains the identifier name, with or without format tag.
func containsID(tmpl, name string) bool {
	return strings.Contains(tmpl, "$"+name+"$") || strings.Contains(tmpl, "$"+name+"%")
}
