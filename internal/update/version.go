package update

import (
	"fmt"
	"strconv"
	"strings"
)

// NormalizeVersion strips a leading "v"/"V" and trims space. Empty stays empty.
func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

// CompareSemver returns -1 if a < b, 0 if equal, 1 if a > b.
// Non-semver strings (e.g. "dev") compare as less than any valid semver, and
// equal only to an identical non-semver string.
func CompareSemver(a, b string) int {
	a, b = NormalizeVersion(a), NormalizeVersion(b)
	ap, aOK := parseSemver(a)
	bp, bOK := parseSemver(b)
	switch {
	case aOK && bOK:
		for i := 0; i < 3; i++ {
			if ap[i] < bp[i] {
				return -1
			}
			if ap[i] > bp[i] {
				return 1
			}
		}
		return 0
	case !aOK && !bOK:
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	case !aOK:
		return -1 // "dev" / garbage is always older than a real release
	default:
		return 1
	}
}

func parseSemver(v string) ([3]int, bool) {
	var out [3]int
	// Drop build/prerelease metadata: 1.2.0-rc.1+meta → 1.2.0
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	if len(parts) == 2 {
		out[2] = 0
	}
	return out, true
}

// AssetName builds the release tarball filename for a version + GOARCH-style arch.
func AssetName(version, arch string) string {
	return fmt.Sprintf("mhtodo_%s_linux_%s.tar.gz", NormalizeVersion(version), arch)
}
