package instance

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// Scheme is the custom URI scheme registered via the .desktop MimeType.
	Scheme = "mhtodo"
	// TaskHost is the URI host for task deep links (mhtodo://task/{ref}).
	TaskHost = "task"
)

// TaskURI builds a deep link for a task ref (prefer short8 or full id).
func TaskURI(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "/")
	return fmt.Sprintf("%s://%s/%s", Scheme, TaskHost, ref)
}

// ParseOpenTarget accepts a raw task id/prefix or a mhtodo://task/{ref} URI.
// Returns the task ref (id or unique prefix) to resolve via core.Get.
func ParseOpenTarget(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("empty open target")
	}

	if scheme, ok := uriSchemePrefix(s); ok && !strings.EqualFold(scheme, Scheme) {
		return "", fmt.Errorf("unsupported URI scheme %q (want %s)", scheme, Scheme)
	}

	lower := strings.ToLower(s)
	if !strings.HasPrefix(lower, Scheme+":") {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid deep link: %w", err)
	}
	if !strings.EqualFold(u.Scheme, Scheme) {
		return "", fmt.Errorf("unsupported URI scheme %q (want %s)", u.Scheme, Scheme)
	}

	host := strings.ToLower(u.Host)
	path := strings.Trim(u.Path, "/")
	opaque := strings.Trim(u.Opaque, "/")

	switch {
	case host == TaskHost && path != "":
		return path, nil
	case host == TaskHost && path == "" && opaque != "":
		return opaque, nil
	case host == "" && strings.HasPrefix(strings.ToLower(opaque), TaskHost+"/"):
		return strings.TrimPrefix(opaque, TaskHost+"/"), nil
	case host == "" && strings.HasPrefix(strings.ToLower(opaque), TaskHost):
		// mhtodo:task/<ref> after Trim may leave "task/<ref>"
		rest := opaque
		if len(rest) >= len(TaskHost) && strings.EqualFold(rest[:len(TaskHost)], TaskHost) {
			rest = strings.TrimPrefix(rest[len(TaskHost):], "/")
		}
		if rest != "" {
			return rest, nil
		}
	case host != "" && host != TaskHost && path == "":
		// Tolerate mhtodo://{ref} as a short form.
		return host, nil
	}

	return "", fmt.Errorf("unrecognized deep link %q (want %s://%s/{{id}})", s, Scheme, TaskHost)
}

// uriSchemePrefix returns the leading RFC3986 scheme if s looks like a URI
// (alpha … followed by ':'). Bare task ids have no scheme.
func uriSchemePrefix(s string) (scheme string, ok bool) {
	if len(s) < 2 {
		return "", false
	}
	c0 := s[0]
	if !((c0 >= 'a' && c0 <= 'z') || (c0 >= 'A' && c0 <= 'Z')) {
		return "", false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ':':
			return s[:i], true
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.':
			continue
		default:
			return "", false
		}
	}
	return "", false
}
