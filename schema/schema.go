// Package schema validates MPDs against the MPEG-DASH XML schema.
//
// The schema files in xsd/ are the 6th edition DASH-MPD.xsd from
// github.com/MPEGGroup/DASHSchema with its internal DTD entities expanded
// (xmllint --noent --dropdtd), since the validator does not support DTDs,
// plus the W3C xlink.xsd and xml.xsd that it imports.
package schema

import (
	"bytes"
	"embed"
	"errors"
	"path"
	"sync"

	"github.com/jacoelho/xsd"
	"github.com/jacoelho/xsd/xsderrors"
	"github.com/tobbee/dashcheck/report"
)

//go:embed xsd/*.xsd
var xsdFiles embed.FS

var (
	RuleWellFormed = report.Register(report.Rule{
		ID: "MPD-XML", Severity: report.Error,
		Desc:    "The MPD must be well-formed XML",
		SpecRef: "W3C XML 1.0",
	})
	RuleSchema = report.Register(report.Rule{
		ID: "MPD-XSD", Severity: report.Error,
		Desc:    "The MPD must validate against the DASH MPD XML schema",
		SpecRef: "ISO/IEC 23009-1 6th ed. DASH-MPD.xsd",
	})
)

var (
	engineOnce sync.Once
	engine     *xsd.Engine
	engineErr  error
)

func source(name string) xsd.SchemaSource {
	name = path.Base(name)
	data, err := xsdFiles.ReadFile("xsd/" + name)
	if err != nil {
		// The resolver below reports a missing import through Compile.
		data = nil
	}
	return xsd.Bytes(name, data).WithResolver(xsd.ResolverFunc(
		func(base, location string) (xsd.SchemaSource, error) {
			return source(location), nil
		}))
}

func compiled() (*xsd.Engine, error) {
	engineOnce.Do(func() {
		engine, engineErr = xsd.Compile(source("DASH-MPD.xsd"))
	})
	return engine, engineErr
}

// MaxErrors limits the number of schema errors reported for one MPD.
const MaxErrors = 100

// Validate checks that data is well-formed and valid according to the schema.
func Validate(data []byte, mpdURL string) ([]report.Finding, error) {
	eng, err := compiled()
	if err != nil {
		return nil, err
	}
	err = eng.ValidateWithOptions(bytes.NewReader(data), xsd.ValidateOptions{MaxErrors: MaxErrors})
	if err == nil {
		return nil, nil
	}
	var findings []report.Finding
	for _, e := range flatten(err) {
		var xe *xsderrors.Error
		if !errors.As(e, &xe) {
			findings = append(findings, RuleSchema.New(report.Location{URL: mpdURL}, "%s", e.Error()))
			continue
		}
		rule := RuleSchema
		if xe.Code() == xsderrors.CodeValidationXML {
			rule = RuleWellFormed
		}
		loc := report.Location{URL: mpdURL, Line: xe.Line(), Col: xe.Column()}
		if xe.Path() != "/" {
			loc.Path = xe.Path()
		}
		findings = append(findings, rule.New(loc, "%s", message(xe)))
	}
	return findings, nil
}

// message combines the diagnostic message with its cause.
func message(xe *xsderrors.Error) string {
	msg := xe.Message()
	if c := xe.Cause(); c != nil {
		if msg != "" {
			msg += ": "
		}
		msg += c.Error()
	}
	return msg
}

// flatten unpacks an xsderrors.Errors aggregate into its children.
func flatten(err error) []error {
	var agg xsderrors.Errors
	if errors.As(err, &agg) {
		return agg.Unwrap()
	}
	return []error{err}
}
