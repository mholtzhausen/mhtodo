package core

import (
	"regexp"
	"strings"
)

// MinSearchIDLen is the minimum number of hex characters (dashes ignored) a
// bare search query needs before it is also matched against ticket IDs.
// UUIDv7 IDs share their leading timestamp digits, so shorter fragments would
// match most of the board. Deep links (mhtodo://task/…) only need MinPrefixLen.
const MinSearchIDLen = 6

// ExactSearchIDLen is the fragment length (the short ID) from which a bare
// hex query counts as a reference to one ticket rather than a fragment.
const ExactSearchIDLen = 8

var (
	searchLinkRe = regexp.MustCompile(`(?i)mhtodo:(?://)?task/([0-9a-f][0-9a-f-]*)`)
	searchHexRe  = regexp.MustCompile(`^[0-9a-fA-F-]+$`)
)

// SearchIDFragment extracts a ticket-ID fragment from a search query, or ""
// when the query does not look like an ID. It accepts a mhtodo://task/{ref}
// link anywhere in the query (e.g. a pasted "MHTodo Ticket: …" line) or a
// bare hex string of at least MinSearchIDLen characters. The fragment is
// lowercase with dashes removed; explicit reports that it came from a link.
func SearchIDFragment(q string) (frag string, explicit bool) {
	if m := searchLinkRe.FindStringSubmatch(q); m != nil {
		frag = normalizeIDFragment(m[1])
		if len(frag) >= MinPrefixLen && len(frag) <= 32 {
			return frag, true
		}
		return "", false
	}
	s := strings.TrimSpace(q)
	if !searchHexRe.MatchString(s) {
		return "", false
	}
	frag = normalizeIDFragment(s)
	if len(frag) < MinSearchIDLen || len(frag) > 32 {
		return "", false
	}
	return frag, false
}

// ExactSearchID returns the one ID in ids that the query names — a link, a
// short ID (ShortID: the last 8 hex chars) or a full UUID, dashes and case
// ignored, that is a prefix or suffix of exactly one of ids (as ResolveID) —
// or "" when the query is a plain fragment, text, or ambiguous.
// The GUI uses it to open the ticket a pasted reference points at.
func ExactSearchID(q string, ids []string) string {
	frag, explicit := SearchIDFragment(q)
	if frag == "" || (!explicit && len(frag) < ExactSearchIDLen) {
		return ""
	}
	hit := ""
	for _, id := range ids {
		n := normalizeIDFragment(id)
		if strings.HasPrefix(n, frag) || strings.HasSuffix(n, frag) {
			if hit != "" && hit != id {
				return ""
			}
			hit = id
		}
	}
	return hit
}

// MatchesSearchID reports whether id contains the query's ID fragment.
func MatchesSearchID(q, id string) bool {
	frag, _ := SearchIDFragment(q)
	return frag != "" && strings.Contains(normalizeIDFragment(id), frag)
}

func normalizeIDFragment(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "-", ""))
}
