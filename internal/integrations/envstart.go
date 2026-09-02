package integrations

import (
	"strings"
	"unicode"
)

// ParseEnvStart splits a shell-style prefix into environment assignments and
// optional leading command arguments (e.g. "FOO=bar --flag" before the binary).
func ParseEnvStart(s string) (env []string, args []string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	tokens := splitShellWords(s)
	for _, tok := range tokens {
		if i := strings.IndexByte(tok, '='); i > 0 && isEnvKey(tok[:i]) {
			env = append(env, tok)
			continue
		}
		args = append(args, tok)
	}
	return env, args
}

func isEnvKey(k string) bool {
	if k == "" {
		return false
	}
	for i, r := range k {
		if i == 0 && r >= '0' && r <= '9' {
			return false
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func splitShellWords(s string) []string {
	var out []string
	var cur strings.Builder
	inSingle, inDouble := false, false
	escaped := false
	for _, r := range s {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}
		switch {
		case r == '\\' && !inSingle:
			escaped = true
		case inSingle:
			if r == '\'' {
				inSingle = false
			} else {
				cur.WriteRune(r)
			}
		case inDouble:
			if r == '"' {
				inDouble = false
			} else {
				cur.WriteRune(r)
			}
		case r == '\'':
			inSingle = true
		case r == '"':
			inDouble = true
		case unicode.IsSpace(r):
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// ShellDoubleQuote wraps s in double quotes for a POSIX shell argument.
func ShellDoubleQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\', '"', '$', '`':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
