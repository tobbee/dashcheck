package mpdcheck

import (
	"bytes"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"

	m "github.com/Eyevinn/dash-mpd/mpd"
	"github.com/tobbee/dashcheck/addressing"
	"github.com/tobbee/dashcheck/report"
)

// Checks that compare two versions of a dynamic MPD: the general update
// rules of ISO/IEC 23009-1 5.4.1 and the DVB-DASH constraints of ETSI TS
// 103 285 4.8 (DVB-43 to DVB-55).

const (
	refUpdate = "ISO/IEC 23009-1 5.4.1"
	resetURN  = "urn:mpeg:dash:reset:2016"
	directURN = "urn:mpeg:dash:utc:direct:2014"
	// maxUpdateWindow limits the segments compared between two versions,
	// and updateMargin keeps away from the start of the time-shift buffer.
	maxUpdateWindow = 120.0
	updateMargin    = 2.0
)

var (
	RuleUpdatePublishTime = report.Register(report.Rule{
		ID: "MPD-UPDATE-PUBLISHTIME", Severity: report.Error,
		Desc:    "A changed MPD must have a new @publishTime, and a later fetch must not return an earlier version",
		SpecRef: "ISO/IEC 23009-1 5.3.1.2, 5.4.1",
	})
	RuleUpdateID = report.Register(report.Rule{
		ID: "MPD-UPDATE-ID", Severity: report.Error,
		Desc: "MPD@id, Period@id and AdaptationSet@id must stay the same in MPD updates", SpecRef: refUpdate,
	})
	RuleUpdateRep = report.Register(report.Rule{
		ID: "MPD-UPDATE-REP", Severity: report.Error,
		Desc:    "A Representation must keep functionally equivalent attributes in MPD updates",
		SpecRef: refUpdate,
	})
	RuleUpdateSegments = report.Register(report.Rule{
		ID: "MPD-UPDATE-SEGMENTS", Severity: report.Error,
		Desc:    "Segments described by both versions of an MPD must keep their timing and must not disappear while available",
		SpecRef: refUpdate,
	})
	RuleUpdateReset = report.Register(report.Rule{
		ID: "MPD-UPDATE-RESET", Severity: report.Info,
		Desc:    "An MPD update signals a reset, so the MPD update rules do not apply to it",
		SpecRef: "ISO/IEC 23009-1 5.4.2",
	})
	RuleDVBUpdate = report.Register(report.Rule{
		ID: "MPD-DVB-UPDATE", Severity: report.Error,
		Desc: "A DVB MPD update must keep @timeShiftBufferDepth, @availabilityStartTime, Period@start, AssetIdentifier, " +
			"Role, AudioChannelConfiguration, @timescale and @lang, must not increase @maxSegmentDuration, " +
			"and must not remove AdaptationSets or Representations",
		SpecRef: refDVB + " 4.8.1, 4.8.3",
	})
)

// Version is one fetched version of a dynamic MPD.
type Version struct {
	MPD  *m.MPD
	Data []byte
	// Time is when the version was fetched, on the synchronized clock.
	Time time.Time
}

// CheckUpdate compares a dynamic MPD with a later fetch of it.
func CheckUpdate(prev, next Version, mpdURL string) []report.Finding {
	if prev.MPD == nil || next.MPD == nil || prev.MPD.GetType() != "dynamic" {
		return nil
	}
	var f []report.Finding
	mpdLoc := report.Location{URL: mpdURL, Path: "MPD"}
	changed := !sameVersion(prev.Data, next.Data)
	pt0, err0 := addressing.ParseDateTime(string(prev.MPD.PublishTime))
	pt1, err1 := addressing.ParseDateTime(string(next.MPD.PublishTime))
	if err0 == nil && err1 == nil {
		switch {
		case pt1.Before(pt0):
			f = append(f, RuleUpdatePublishTime.New(mpdLoc, "fetched at %s with @publishTime %s, earlier than %s fetched at %s",
				clockTime(next.Time), next.MPD.PublishTime, prev.MPD.PublishTime, clockTime(prev.Time)))
		case pt1.Equal(pt0) && changed:
			f = append(f, RuleUpdatePublishTime.New(mpdLoc, "the MPD fetched at %s differs from the one fetched at %s but has the same @publishTime %s",
				clockTime(next.Time), clockTime(prev.Time), next.MPD.PublishTime))
		}
	}
	if !changed {
		return f
	}
	if hasReset(next.MPD) {
		return append(f, RuleUpdateReset.New(mpdLoc, "the MPD fetched at %s has a %s descriptor", clockTime(next.Time), resetURN))
	}
	u := &updateCheck{prev: prev, next: next, url: mpdURL, dvb: isDVBMPD(next.MPD)}
	u.mpdLevel()
	u.periods()
	return append(f, u.findings...)
}

type updateCheck struct {
	prev, next Version
	url        string
	dvb        bool
	findings   []report.Finding
}

func (u *updateCheck) add(rule *report.Rule, path, format string, args ...any) {
	u.findings = append(u.findings, rule.New(report.Location{URL: u.url, Path: path}, format, args...))
}

func (u *updateCheck) mpdLevel() {
	p, n := u.prev.MPD, u.next.MPD
	if p.Id != n.Id {
		u.add(RuleUpdateID, "MPD", "MPD@id changed from %q to %q", p.Id, n.Id)
	}
	if !u.dvb {
		return
	}
	// DVB-43, DVB-44, DVB-45
	if a, b := durationString(p.TimeShiftBufferDepth), durationString(n.TimeShiftBufferDepth); a != b {
		u.add(RuleDVBUpdate, "MPD", "@timeShiftBufferDepth changed from %s to %s", a, b)
	}
	if p.AvailabilityStartTime != n.AvailabilityStartTime {
		u.add(RuleDVBUpdate, "MPD", "@availabilityStartTime changed from %q to %q", p.AvailabilityStartTime, n.AvailabilityStartTime)
	}
	if p.MaxSegmentDuration != nil && (n.MaxSegmentDuration == nil || *n.MaxSegmentDuration > *p.MaxSegmentDuration) {
		u.add(RuleDVBUpdate, "MPD", "@maxSegmentDuration increased from %s to %s",
			durationString(p.MaxSegmentDuration), durationString(n.MaxSegmentDuration))
	}
}

// periods pairs the Periods of the two versions and compares them. Periods
// correspond by @id, or by start time when they have no @id. Removed Periods
// are allowed.
func (u *updateCheck) periods() {
	p, n := u.prev.MPD, u.next.MPD
	pStarts, nStarts := periodStarts(p), periodStarts(n)
	nByID := map[string]int{}
	for i, np := range n.Periods {
		if np.Id != "" {
			nByID[np.Id] = i
		}
	}
	prevReps, _ := addressing.Reps(p, u.url)
	nextReps, _ := addressing.Reps(n, u.url)
	prevIDs := map[string]bool{}
	for _, pp := range p.Periods {
		prevIDs[pp.Id] = true
	}
	for pi, pp := range p.Periods {
		ni, ok := nByID[pp.Id]
		if pp.Id == "" || !ok {
			// A Period that starts at the same time under another @id has
			// changed its @id; otherwise the Period was removed.
			ni = slices.IndexFunc(nStarts, func(s float64) bool { return s == pStarts[pi] })
			if ni < 0 {
				continue
			}
			if id := n.Periods[ni].Id; id != pp.Id {
				if !prevIDs[id] {
					// DVB-46
					u.add(RuleUpdateID, fmt.Sprintf("Period[%d]", ni), "Period@id changed from %q to %q", pp.Id, id)
				}
				continue
			}
		}
		np := n.Periods[ni]
		path := fmt.Sprintf("Period[%d]", ni)
		if u.dvb {
			// DVB-47, DVB-48
			if !math.IsNaN(pStarts[pi]) && !math.IsNaN(nStarts[ni]) && pStarts[pi] != nStarts[ni] {
				u.add(RuleDVBUpdate, path, "Period@start changed from %gs to %gs", pStarts[pi], nStarts[ni])
			}
			if a, b := descriptorString(pp.AssetIdentifier), descriptorString(np.AssetIdentifier); a != b {
				u.add(RuleDVBUpdate, path, "AssetIdentifier changed from %s to %s", a, b)
			}
		}
		u.period(pi, ni, path, filterPeriod(prevReps, pi), filterPeriod(nextReps, ni))
	}
}

// period compares the AdaptationSets and Representations of a Period.
// Representations correspond by @id; AdaptationSets through the
// Representations they contain.
func (u *updateCheck) period(pi, ni int, path string, prevReps, nextReps []*addressing.Rep) {
	pp, np := u.prev.MPD.Periods[pi], u.next.MPD.Periods[ni]
	nextByID := map[string]*addressing.Rep{}
	for _, r := range nextReps {
		nextByID[r.ID] = r
	}
	asPairs := map[string]string{} // previous AdaptationSet path -> next one
	missing := map[string][]string{}
	for _, pr := range prevReps {
		nr := nextByID[pr.ID]
		if nr == nil {
			missing[pr.ASPath] = append(missing[pr.ASPath], pr.ID)
			continue
		}
		if _, ok := asPairs[pr.ASPath]; !ok {
			asPairs[pr.ASPath] = nr.ASPath
		}
		u.rep(pr, nr)
		u.segments(pr, nr)
	}
	// DVB-49, DVB-51, DVB-52, DVB-55 (@contentType and @codecs are compared per Representation)
	for prevPath, nextPath := range asPairs {
		pas, nas := adaptationSet(pp, prevPath), adaptationSet(np, nextPath)
		if pas == nil || nas == nil {
			continue
		}
		if a, b := uintString(pas.Id), uintString(nas.Id); a != b {
			u.add(RuleUpdateID, nextPath, "AdaptationSet@id changed from %s to %s", a, b)
		}
		if !u.dvb {
			continue
		}
		if pas.Lang != nas.Lang {
			u.add(RuleDVBUpdate, nextPath, "AdaptationSet@lang changed from %q to %q", pas.Lang, nas.Lang)
		}
		if a, b := descriptorsString(pas.Roles), descriptorsString(nas.Roles); a != b {
			u.add(RuleDVBUpdate, nextPath, "Role changed from %s to %s", a, b)
		}
		if a, b := descriptorsString(pas.AudioChannelConfigurations), descriptorsString(nas.AudioChannelConfigurations); a != b {
			u.add(RuleDVBUpdate, nextPath, "AudioChannelConfiguration changed from %s to %s", a, b)
		}
	}
	if !u.dvb {
		return
	}
	// DVB 4.8.1 allows adding AdaptationSets and Representations, not removing them.
	for _, asPath := range slices.Sorted(maps.Keys(missing)) {
		ids := missing[asPath]
		if _, kept := asPairs[asPath]; !kept {
			u.add(RuleDVBUpdate, path, "%s (Representations %s) was removed", asPath, strings.Join(ids, ", "))
			continue
		}
		u.add(RuleDVBUpdate, asPairs[asPath], "Representations %s were removed", strings.Join(ids, ", "))
	}
}

// rep compares the effective attributes of corresponding Representations.
// @bandwidth may change (TS 103 285 4.8.2).
func (u *updateCheck) rep(pr, nr *addressing.Rep) {
	type attr struct{ name, prev, next string }
	attrs := []attr{
		{"@mimeType", pr.MimeType, nr.MimeType},
		{"@codecs", pr.Codecs, nr.Codecs},
		{"@contentType", pr.ContentType, nr.ContentType},
		{"@width", fmt.Sprint(pr.Width), fmt.Sprint(nr.Width)},
		{"@height", fmt.Sprint(pr.Height), fmt.Sprint(nr.Height)},
		{"@frameRate", pr.FrameRate, nr.FrameRate},
		{"@sar", pr.Sar, nr.Sar},
		{"@audioSamplingRate", pr.AudioSamplingRate, nr.AudioSamplingRate},
	}
	for _, a := range attrs {
		if a.prev != a.next {
			u.add(RuleUpdateRep, nr.Path, "%s changed from %q to %q", a.name, a.prev, a.next)
		}
	}
	if u.dvb && pr.Timescale != nr.Timescale {
		u.add(RuleDVBUpdate, nr.Path, "@timescale changed from %d to %d", pr.Timescale, nr.Timescale)
	}
}

// segments compares the segments that were available when the previous
// version was fetched and are still in the time-shift buffer of the next one.
func (u *updateCheck) segments(pr, nr *addressing.Rep) {
	if pr.Kind != nr.Kind || pr.Kind == addressing.KindSegmentBase || pr.Kind == addressing.KindUnknown ||
		pr.Timescale != nr.Timescale || pr.Timescale == 0 {
		return
	}
	periodStart := pr.AST.Add(time.Duration(pr.PeriodStart * float64(time.Second)))
	hi := u.prev.Time.Sub(periodStart).Seconds() + pr.ATO
	lo := u.next.Time.Sub(periodStart).Seconds() - math.Min(nr.TSBD, maxUpdateWindow) + updateMargin
	lo = math.Max(lo, math.Max(0, hi-maxUpdateWindow))
	if hi <= lo {
		return
	}
	prevSegs, err0 := pr.Segments(lo, hi)
	nextSegs, err1 := nr.Segments(lo, hi)
	if err0 != nil || err1 != nil {
		return
	}
	key := func(s addressing.Segment) uint64 {
		if pr.Kind == addressing.KindTimelineTime {
			return s.Time
		}
		return s.Number
	}
	ts := float64(pr.Timescale)
	inWindow := func(s addressing.Segment) bool {
		start := float64(s.Time)/ts - float64(pr.PTO)/ts
		return start >= lo && start+float64(s.Dur)/ts <= hi
	}
	nextByKey := map[uint64]addressing.Segment{}
	for _, s := range nextSegs {
		nextByKey[key(s)] = s
	}
	for _, s := range prevSegs {
		if !inWindow(s) {
			continue
		}
		ns, ok := nextByKey[key(s)]
		id := s.ID(pr.Kind)
		switch {
		case !ok:
			u.add(RuleUpdateSegments, nr.Path, "segment %s (time %d) was available in the previous version but is missing", id, s.Time)
			return
		case ns.Time != s.Time || ns.Dur != s.Dur:
			u.add(RuleUpdateSegments, nr.Path, "segment %s changed from time %d, duration %d to time %d, duration %d",
				id, s.Time, s.Dur, ns.Time, ns.Dur)
			return
		}
	}
}

// sameVersion reports whether two fetches hold the same MPD: the same
// elements, attributes and text, apart from the time in a utc:direct
// UTCTiming, which the server writes into each response (DASH-IF IOP 4.3
// 4.7.2), so that it changes without a new @publishTime.
func sameVersion(a, b []byte) bool {
	if bytes.Equal(a, b) {
		return true
	}
	ra, rb := parseRaw(a), parseRaw(b)
	return ra != nil && rb != nil && rawEqual(ra, rb)
}

// rawEqual compares two raw trees, ignoring the value of utc:direct UTCTimings.
func rawEqual(a, b *rawElem) bool {
	if a.Name != b.Name || a.Text != b.Text || len(a.Children) != len(b.Children) {
		return false
	}
	aa, ba := a.Attrs, b.Attrs
	if a.Name == "UTCTiming" && aa["schemeIdUri"] == directURN && ba["schemeIdUri"] == directURN {
		aa, ba = maps.Clone(aa), maps.Clone(ba)
		delete(aa, "value")
		delete(ba, "value")
	}
	if !maps.Equal(aa, ba) {
		return false
	}
	for i := range a.Children {
		if !rawEqual(a.Children[i], b.Children[i]) {
			return false
		}
	}
	return true
}

func hasReset(md *m.MPD) bool {
	return slices.ContainsFunc(append(slices.Clone(md.EssentialProperties), md.SupplementalProperties...),
		func(d *m.DescriptorType) bool { return d.SchemeIdUri == resetURN })
}

func isDVBMPD(md *m.MPD) bool {
	return isDVB(profiles(splitProfiles(string(md.Profiles))))
}

// periodStarts returns the start of each Period in seconds, NaN if unknown.
func periodStarts(md *m.MPD) []float64 {
	starts := make([]float64, len(md.Periods))
	next := 0.0
	for i, p := range md.Periods {
		if p.Start != nil {
			next = time.Duration(*p.Start).Seconds()
		}
		starts[i] = next
		if p.Duration != nil {
			next += time.Duration(*p.Duration).Seconds()
		} else {
			next = math.NaN()
		}
	}
	return starts
}

func filterPeriod(reps []*addressing.Rep, pi int) []*addressing.Rep {
	var out []*addressing.Rep
	for _, r := range reps {
		if r.PeriodIndex == pi {
			out = append(out, r)
		}
	}
	return out
}

func adaptationSet(p *m.Period, path string) *m.AdaptationSetType {
	for ai, as := range p.AdaptationSets {
		if strings.HasSuffix(path, fmt.Sprintf("/AdaptationSet[%d]", ai)) {
			return as
		}
	}
	return nil
}

func durationString(d *m.Duration) string {
	if d == nil {
		return "absent"
	}
	return time.Duration(*d).String()
}

func uintString(v *uint32) string {
	if v == nil {
		return "absent"
	}
	return fmt.Sprint(*v)
}

func descriptorString(d *m.DescriptorType) string {
	if d == nil {
		return "absent"
	}
	return fmt.Sprintf("%s=%q", d.SchemeIdUri, d.Value)
}

func descriptorsString(ds []*m.DescriptorType) string {
	if len(ds) == 0 {
		return "absent"
	}
	s := make([]string, len(ds))
	for i, d := range ds {
		s[i] = descriptorString(d)
	}
	slices.Sort(s)
	return strings.Join(s, ", ")
}

func clockTime(t time.Time) string {
	return t.UTC().Format("15:04:05.000Z")
}
