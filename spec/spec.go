// Package spec lists the profiles that turn on profile-specific rules, and
// lets the user choose which of them an MPD is checked against.
//
// By default (auto) dashcheck checks the profiles the MPD lists in @profiles.
// A Selection can add profiles that the MPD does not list, or check only the
// chosen ones. It works by rewriting the @profiles of the parsed MPD, so that
// every rule sees the chosen profiles as if the MPD had signalled them.
package spec

import (
	"fmt"
	"slices"
	"strings"

	m "github.com/Eyevinn/dash-mpd/mpd"
)

// A Profile is a specification with its own rules, signalled by a profile in @profiles.
type Profile struct {
	Name  string `json:"name"`  // option value, such as "dvb"
	Title string `json:"title"` // such as "DVB-DASH 2014"
	URN   string `json:"urn"`   // the profile added to @profiles when the user chooses it
	// Prefix, if set, makes every profile that starts with it signal the
	// specification, as for the many DASH-IF IOP profiles.
	Prefix string `json:"prefix,omitempty"`
}

// Profiles are the profiles a user can choose, in display order.
var Profiles = []Profile{
	{Name: "iop", Title: "DASH-IF IOP", URN: "http://dashif.org/guidelines/dash-if-main",
		Prefix: "http://dashif.org/guidelines/dash"},
	{Name: "cmaf", Title: "CMAF", URN: "urn:mpeg:dash:profile:cmaf:2019"},
	{Name: "low-latency", Title: "DASH-IF low latency", URN: "http://www.dashif.org/guidelines/low-latency-live-v5"},
	{Name: "dvb", Title: "DVB-DASH 2014", URN: "urn:dvb:dash:profile:dvb-dash:2014"},
	{Name: "dvb2017", Title: "DVB-DASH 2017", URN: "urn:dvb:dash:profile:dvb-dash:2017"},
	{Name: "hbbtv", Title: "HbbTV", URN: "urn:hbbtv:dash:profile:isoff-live:2012"},
	{Name: "wave", Title: "CTA WAVE", URN: "urn:cta:wave:test-content-media-profile:2022", Prefix: "urn:cta:wave:"},
}

// Signal describes the profiles that signal p, such as its URN.
func (p Profile) Signal() string {
	if p.Prefix != "" {
		return "a profile starting with " + p.Prefix
	}
	return p.URN
}

// Matches reports whether profile, an entry of @profiles, signals p.
func (p Profile) Matches(profile string) bool {
	if p.Prefix != "" {
		return strings.HasPrefix(profile, p.Prefix)
	}
	return profile == p.URN
}

// Selection is the choice of profiles to check. The zero value is auto: the
// profiles that the MPD signals.
type Selection struct {
	Chosen []Profile // profiles checked whether the MPD signals them or not
	// Only checks only the chosen profiles: those the MPD signals but the user
	// did not choose are removed, and CMAF brands in segments and chunked low-latency
	// Representations no longer turn on the CMAF and low-latency rules.
	Only bool
}

// Parse parses a comma-separated list of profile names. "auto" stands for the
// profiles the MPD signals, and "none" for no profile-specific rules. The
// empty string is auto. Without auto, only the listed profiles are checked.
func Parse(s string) (Selection, error) {
	sel := Selection{Only: true}
	auto, none := false, false
	for name := range strings.SplitSeq(s, ",") {
		name = strings.TrimSpace(name)
		switch name {
		case "":
		case "auto":
			auto = true
		case "none":
			none = true
		default:
			i := slices.IndexFunc(Profiles, func(p Profile) bool { return p.Name == name })
			if i < 0 {
				return Selection{}, fmt.Errorf("unknown profile %q (choose from auto, none, %s)", name, strings.Join(Names(), ", "))
			}
			if !slices.Contains(sel.Chosen, Profiles[i]) {
				sel.Chosen = append(sel.Chosen, Profiles[i])
			}
		}
	}
	switch {
	case none && (auto || len(sel.Chosen) > 0):
		return Selection{}, fmt.Errorf("none cannot be combined with other profiles")
	case auto || (!none && len(sel.Chosen) == 0):
		sel.Only = false
	}
	return sel, nil
}

// Names returns the names of the profiles.
func Names() []string {
	names := make([]string, len(Profiles))
	for i, p := range Profiles {
		names[i] = p.Name
	}
	return names
}

// IsAuto reports whether the selection is plain auto.
func (s Selection) IsAuto() bool { return !s.Only && len(s.Chosen) == 0 }

// String returns the selection in the form Parse accepts.
func (s Selection) String() string {
	var names []string
	if !s.Only {
		names = append(names, "auto")
	}
	for _, p := range s.Chosen {
		names = append(names, p.Name)
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ",")
}

// Apply rewrites the @profiles of the MPD and of its AdaptationSets,
// Representations, SubRepresentations and Preselections for the selection. It
// adds each chosen profile to every @profiles that is set, the MPD's included,
// so that inheritance keeps it. With Only, it removes the profiles of the
// specifications that were not chosen. It returns the chosen profiles that
// MPD@profiles did not list.
func (s Selection) Apply(md *m.MPD) (unlisted []Profile) {
	if s.IsAuto() {
		return nil
	}
	mpdProfiles := split(string(md.Profiles))
	for _, p := range s.Chosen {
		if !slices.ContainsFunc(mpdProfiles, p.Matches) {
			unlisted = append(unlisted, p)
		}
	}
	md.Profiles = s.rewrite(md.Profiles, true)
	for _, period := range md.Periods {
		for _, as := range period.AdaptationSets {
			as.Profiles = s.rewrite(as.Profiles, false)
			for _, rep := range as.Representations {
				rep.Profiles = s.rewrite(rep.Profiles, false)
				for _, sr := range rep.SubRepresentations {
					sr.Profiles = s.rewrite(sr.Profiles, false)
				}
			}
		}
		for _, ps := range period.Preselections {
			ps.Profiles = s.rewrite(ps.Profiles, false)
		}
	}
	return unlisted
}

func (s Selection) rewrite(list m.ListOfProfilesType, mpdLevel bool) m.ListOfProfilesType {
	if list == "" && !mpdLevel {
		return list // inherited, so it gets the profiles from the level above
	}
	var out []string
	for _, prof := range split(string(list)) {
		if s.Only && !slices.ContainsFunc(s.Chosen, func(p Profile) bool { return p.Matches(prof) }) &&
			slices.ContainsFunc(Profiles, func(p Profile) bool { return p.Matches(prof) }) {
			continue
		}
		out = append(out, prof)
	}
	for _, p := range s.Chosen {
		if !slices.ContainsFunc(out, p.Matches) {
			out = append(out, p.URN)
		}
	}
	return m.ListOfProfilesType(strings.Join(out, ","))
}

// Signalled returns the profiles that the MPD signals at any level, after Apply.
func Signalled(md *m.MPD) []Profile {
	var all []string
	add := func(list m.ListOfProfilesType) { all = append(all, split(string(list))...) }
	add(md.Profiles)
	for _, period := range md.Periods {
		for _, as := range period.AdaptationSets {
			add(as.Profiles)
			for _, rep := range as.Representations {
				add(rep.Profiles)
				for _, sr := range rep.SubRepresentations {
					add(sr.Profiles)
				}
			}
		}
		for _, ps := range period.Preselections {
			add(ps.Profiles)
		}
	}
	var out []Profile
	for _, p := range Profiles {
		if slices.ContainsFunc(all, p.Matches) {
			out = append(out, p)
		}
	}
	return out
}

func split(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
