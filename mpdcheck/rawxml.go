package mpdcheck

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// rawElem is an MPD element as written, for rules that need to know whether
// an attribute is present, or attributes that the MPD model does not keep.
type rawElem struct {
	Name     string // local name
	Attrs    map[string]string
	Children []*rawElem
	Text     string // character data directly inside the element
	Line     int
}

// parseRaw parses data into a tree of rawElem. It returns nil if the XML is
// not well-formed, which the schema check reports.
func parseRaw(data []byte) *rawElem {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var root *rawElem
	var stack []*rawElem
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			line, _ := dec.InputPos()
			e := &rawElem{Name: t.Name.Local, Attrs: map[string]string{}, Line: line}
			for _, a := range t.Attr {
				name := a.Name.Local
				if a.Name.Space != "" && a.Name.Space != "xmlns" {
					name = shortNS(a.Name.Space) + ":" + name
				}
				e.Attrs[name] = a.Value
			}
			if len(stack) == 0 {
				root = e
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, e)
			}
			stack = append(stack, e)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(t)
			}
		}
	}
	return root
}

// shortNS gives a stable prefix for the namespaces that rules refer to.
func shortNS(space string) string {
	switch space {
	case "http://www.w3.org/1999/xlink":
		return "xlink"
	case "urn:mpeg:cenc:2013":
		return "cenc"
	case "urn:dvb:dash:metricreporting:2014", "urn:dvb:dash-extensions:2014-1":
		return "dvb"
	}
	return space
}

// has reports whether the element has the attribute.
func (e *rawElem) has(attr string) bool {
	_, ok := e.Attrs[attr]
	return ok
}

// attr returns an attribute value, also accepting a namespaced variant.
func (e *rawElem) attr(name string) string {
	if v, ok := e.Attrs[name]; ok {
		return v
	}
	for k, v := range e.Attrs {
		if strings.HasSuffix(k, ":"+name) {
			return v
		}
	}
	return ""
}

// children returns the child elements with the given name.
func (e *rawElem) children(name string) []*rawElem {
	var out []*rawElem
	for _, c := range e.Children {
		if c.Name == name {
			out = append(out, c)
		}
	}
	return out
}

// walk calls fn for every element in the tree with the path of names from the root.
func (e *rawElem) walk(fn func(e *rawElem, ancestors []*rawElem)) {
	var rec func(e *rawElem, anc []*rawElem)
	rec = func(e *rawElem, anc []*rawElem) {
		fn(e, anc)
		anc = append(anc, e)
		for _, c := range e.Children {
			rec(c, anc)
		}
	}
	rec(e, nil)
}
