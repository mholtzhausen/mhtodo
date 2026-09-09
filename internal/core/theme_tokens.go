package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Builtin theme keys (stable; used for seed + Reset).
const (
	BuiltinSlate = "slate"
	BuiltinPaper = "paper"
	BuiltinEmber = "ember"
)

// Stable UUIDs for built-in themes (migration seed + tests/docs).
const (
	BuiltinSlateID = "00000000-0000-7000-8000-000000000001"
	BuiltinPaperID = "00000000-0000-7000-8000-000000000002"
	BuiltinEmberID = "00000000-0000-7000-8000-000000000003"
)

// TokenKind selects how a token is validated and which picker Settings uses.
type TokenKind string

const (
	TokenKindColor  TokenKind = "color"
	TokenKindLength TokenKind = "length"
)

// TokenMeta describes one design token in the registry.
type TokenMeta struct {
	Key      string
	Category string // Surfaces, Borders, Text, …
	Label    string
	Kind     TokenKind
	// AllowRGBA permits rgba()/hsla() in addition to hex (color.track).
	AllowRGBA bool
	MinPx     int // length only
	MaxPx     int // length only
}

// ThemeTokenRegistry is the ordered list of known tokens. Unknown keys are
// rejected on write; missing keys are filled from Slate on read.
var ThemeTokenRegistry = []TokenMeta{
	// Surfaces
	{Key: "color.canvas", Category: "Surfaces", Label: "Canvas", Kind: TokenKindColor},
	{Key: "color.chrome", Category: "Surfaces", Label: "Chrome", Kind: TokenKindColor},
	{Key: "color.col", Category: "Surfaces", Label: "Column", Kind: TokenKindColor},
	{Key: "color.card", Category: "Surfaces", Label: "Card", Kind: TokenKindColor},
	{Key: "color.cardHi", Category: "Surfaces", Label: "Card elevated", Kind: TokenKindColor},
	{Key: "color.field", Category: "Surfaces", Label: "Field", Kind: TokenKindColor},
	// Borders
	{Key: "color.line", Category: "Borders", Label: "Line", Kind: TokenKindColor},
	{Key: "color.lineSoft", Category: "Borders", Label: "Line soft", Kind: TokenKindColor},
	// Text
	{Key: "color.ink", Category: "Text", Label: "Ink", Kind: TokenKindColor},
	{Key: "color.ink2", Category: "Text", Label: "Ink secondary", Kind: TokenKindColor},
	{Key: "color.ink3", Category: "Text", Label: "Ink muted", Kind: TokenKindColor},
	// Accent
	{Key: "color.accent", Category: "Accent", Label: "Accent", Kind: TokenKindColor},
	{Key: "color.accentHi", Category: "Accent", Label: "Accent hover", Kind: TokenKindColor},
	{Key: "color.accentInk", Category: "Accent", Label: "Accent ink", Kind: TokenKindColor},
	// Status
	{Key: "color.stPending", Category: "Status", Label: "Pending", Kind: TokenKindColor},
	{Key: "color.stWip", Category: "Status", Label: "WIP", Kind: TokenKindColor},
	{Key: "color.stWaiting", Category: "Status", Label: "Waiting", Kind: TokenKindColor},
	{Key: "color.stReview", Category: "Status", Label: "Review", Kind: TokenKindColor},
	{Key: "color.stDone", Category: "Status", Label: "Done", Kind: TokenKindColor},
	// Feedback
	{Key: "color.danger", Category: "Feedback", Label: "Danger", Kind: TokenKindColor},
	{Key: "color.track", Category: "Feedback", Label: "Progress track", Kind: TokenKindColor, AllowRGBA: true},
	// Radius (px)
	{Key: "radius.control", Category: "Radius", Label: "Control", Kind: TokenKindLength, MinPx: 0, MaxPx: 24},
	{Key: "radius.card", Category: "Radius", Label: "Card", Kind: TokenKindLength, MinPx: 0, MaxPx: 32},
	{Key: "radius.panel", Category: "Radius", Label: "Panel", Kind: TokenKindLength, MinPx: 0, MaxPx: 40},
	{Key: "radius.chip", Category: "Radius", Label: "Chip", Kind: TokenKindLength, MinPx: 0, MaxPx: 16},
	// Space (px)
	{Key: "space.gapSm", Category: "Space", Label: "Gap small", Kind: TokenKindLength, MinPx: 0, MaxPx: 48},
	{Key: "space.gapMd", Category: "Space", Label: "Gap medium", Kind: TokenKindLength, MinPx: 0, MaxPx: 64},
	{Key: "space.gapLg", Category: "Space", Label: "Gap large", Kind: TokenKindLength, MinPx: 0, MaxPx: 96},
	{Key: "space.padSm", Category: "Space", Label: "Pad small", Kind: TokenKindLength, MinPx: 0, MaxPx: 48},
	{Key: "space.padMd", Category: "Space", Label: "Pad medium", Kind: TokenKindLength, MinPx: 0, MaxPx: 64},
	{Key: "space.padLg", Category: "Space", Label: "Pad large", Kind: TokenKindLength, MinPx: 0, MaxPx: 96},
}

var themeTokenIndex map[string]TokenMeta

func init() {
	themeTokenIndex = make(map[string]TokenMeta, len(ThemeTokenRegistry))
	for _, m := range ThemeTokenRegistry {
		themeTokenIndex[m.Key] = m
	}
}

// FactoryTokens returns a copy of the factory token map for a builtin key.
func FactoryTokens(builtinKey string) (map[string]string, bool) {
	src, ok := factoryPalettes[builtinKey]
	if !ok {
		return nil, false
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out, true
}

// SlateFactoryTokens is the default palette (current GUI look).
func SlateFactoryTokens() map[string]string {
	t, _ := FactoryTokens(BuiltinSlate)
	return t
}

var factoryPalettes = map[string]map[string]string{
	BuiltinSlate: {
		"color.canvas":    "#252b37",
		"color.chrome":    "#1f242f",
		"color.col":       "#2c3340",
		"color.card":      "#343c4d",
		"color.cardHi":    "#3d465a",
		"color.field":     "#28303d",
		"color.line":      "#4a556b",
		"color.lineSoft":  "#3a4356",
		"color.ink":       "#eef1f7",
		"color.ink2":      "#b6bed0",
		"color.ink3":      "#8791a5",
		"color.accent":    "#7b8cff",
		"color.accentHi":  "#93a0ff",
		"color.accentInk": "#0e1230",
		"color.stPending": "#a3adbf",
		"color.stWip":     "#7b8cff",
		"color.stWaiting": "#e8ab4a",
		"color.stReview":  "#c084fc",
		"color.stDone":    "#4cc48e",
		"color.danger":    "#ff8492",
		"color.track":     "rgba(255, 255, 255, 0.13)",
		"radius.control":  "4px",
		"radius.card":     "6px",
		"radius.panel":    "8px",
		"radius.chip":     "3px",
		"space.gapSm":     "8px",
		"space.gapMd":     "10px",
		"space.gapLg":     "14px",
		"space.padSm":     "8px",
		"space.padMd":     "12px",
		"space.padLg":     "20px",
	},
	BuiltinPaper: {
		"color.canvas":    "#f4f6fa",
		"color.chrome":    "#e8ecf4",
		"color.col":       "#ffffff",
		"color.card":      "#ffffff",
		"color.cardHi":    "#f0f3f9",
		"color.field":     "#eef1f7",
		"color.line":      "#c5cddc",
		"color.lineSoft":  "#d8dee9",
		"color.ink":       "#1a2030",
		"color.ink2":      "#4a556b",
		"color.ink3":      "#7a8499",
		"color.accent":    "#3b5bdb",
		"color.accentHi":  "#4c6ef5",
		"color.accentInk": "#ffffff",
		"color.stPending": "#868e9c",
		"color.stWip":     "#3b5bdb",
		"color.stWaiting": "#d9480f",
		"color.stReview":  "#9c36b5",
		"color.stDone":    "#2f9e44",
		"color.danger":    "#e03131",
		"color.track":     "rgba(26, 32, 48, 0.12)",
		"radius.control":  "4px",
		"radius.card":     "6px",
		"radius.panel":    "8px",
		"radius.chip":     "3px",
		"space.gapSm":     "8px",
		"space.gapMd":     "10px",
		"space.gapLg":     "14px",
		"space.padSm":     "8px",
		"space.padMd":     "12px",
		"space.padLg":     "20px",
	},
	BuiltinEmber: {
		"color.canvas":    "#2a221c",
		"color.chrome":    "#1f1915",
		"color.col":       "#332920",
		"color.card":      "#3d3126",
		"color.cardHi":    "#4a3b2e",
		"color.field":     "#2e251e",
		"color.line":      "#5c4a3a",
		"color.lineSoft":  "#46382c",
		"color.ink":       "#f5ebe0",
		"color.ink2":      "#c9b8a4",
		"color.ink3":      "#9a8774",
		"color.accent":    "#e8a04a",
		"color.accentHi":  "#f0b56a",
		"color.accentInk": "#1a1208",
		"color.stPending": "#a89888",
		"color.stWip":     "#e8a04a",
		"color.stWaiting": "#e07040",
		"color.stReview":  "#c078d0",
		"color.stDone":    "#6abf7a",
		"color.danger":    "#f07070",
		"color.track":     "rgba(255, 235, 210, 0.12)",
		"radius.control":  "3px",
		"radius.card":     "4px",
		"radius.panel":    "6px",
		"radius.chip":     "2px",
		"space.gapSm":     "8px",
		"space.gapMd":     "10px",
		"space.gapLg":     "14px",
		"space.padSm":     "8px",
		"space.padMd":     "12px",
		"space.padLg":     "20px",
	},
}

// BuiltinThemeSeed describes one row inserted by migration v13.
type BuiltinThemeSeed struct {
	ID         string
	Name       string
	BuiltinKey string
}

// BuiltinThemeSeeds is the ordered seed list (Slate first = default active).
var BuiltinThemeSeeds = []BuiltinThemeSeed{
	{ID: BuiltinSlateID, Name: "Slate", BuiltinKey: BuiltinSlate},
	{ID: BuiltinPaperID, Name: "Paper", BuiltinKey: BuiltinPaper},
	{ID: BuiltinEmberID, Name: "Ember", BuiltinKey: BuiltinEmber},
}

// MergeThemeTokens fills missing keys from Slate factory and returns a complete map.
func MergeThemeTokens(partial map[string]string) map[string]string {
	out := SlateFactoryTokens()
	for k, v := range partial {
		if _, ok := themeTokenIndex[k]; ok {
			out[k] = v
		}
	}
	return out
}

// ValidateThemeTokens checks every entry against the registry. Unknown keys and
// invalid values fail; missing keys are allowed (filled on read).
func ValidateThemeTokens(tokens map[string]string) error {
	if tokens == nil {
		return &InvalidThemeTokensError{Msg: "tokens must not be null"}
	}
	for k, v := range tokens {
		meta, ok := themeTokenIndex[k]
		if !ok {
			return &UnknownThemeTokenError{Key: k}
		}
		if err := validateTokenValue(meta, v); err != nil {
			return err
		}
	}
	return nil
}

// NormalizeThemeTokens validates, trims values, and returns a complete map
// (all registry keys present, missing filled from Slate).
func NormalizeThemeTokens(tokens map[string]string) (map[string]string, error) {
	if err := ValidateThemeTokens(tokens); err != nil {
		return nil, err
	}
	normalized := make(map[string]string, len(tokens))
	for k, v := range tokens {
		meta := themeTokenIndex[k]
		nv, err := normalizeTokenValue(meta, v)
		if err != nil {
			return nil, err
		}
		normalized[k] = nv
	}
	return MergeThemeTokens(normalized), nil
}

var (
	hexColorRe  = regexp.MustCompile(`(?i)^#([0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$`)
	rgbaColorRe = regexp.MustCompile(`(?i)^rgba?\(\s*[\d.]+\s*,\s*[\d.]+\s*,\s*[\d.]+\s*(?:,\s*[\d.]+\s*)?\)$`)
	hslaColorRe = regexp.MustCompile(`(?i)^hsla?\(\s*[\d.]+\s*,\s*[\d.]+%\s*,\s*[\d.]+%\s*(?:,\s*[\d.]+\s*)?\)$`)
	lengthRe    = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*px$`)
)

func validateTokenValue(meta TokenMeta, raw string) error {
	v := strings.TrimSpace(raw)
	if v == "" {
		return &InvalidThemeTokenValueError{Key: meta.Key, Value: raw, Msg: "empty"}
	}
	switch meta.Kind {
	case TokenKindColor:
		if hexColorRe.MatchString(v) {
			return nil
		}
		if meta.AllowRGBA && (rgbaColorRe.MatchString(v) || hslaColorRe.MatchString(v)) {
			return nil
		}
		return &InvalidThemeTokenValueError{Key: meta.Key, Value: raw, Msg: "want #hex" + allowRGBAHint(meta)}
	case TokenKindLength:
		m := lengthRe.FindStringSubmatch(v)
		if m == nil {
			// bare number → treat as px later in normalize
			if _, err := strconv.ParseFloat(v, 64); err == nil {
				return nil
			}
			return &InvalidThemeTokenValueError{Key: meta.Key, Value: raw, Msg: "want Npx"}
		}
		f, _ := strconv.ParseFloat(m[1], 64)
		if int(f) < meta.MinPx || int(f) > meta.MaxPx {
			return &InvalidThemeTokenValueError{
				Key: meta.Key, Value: raw,
				Msg: fmt.Sprintf("out of range %d–%d px", meta.MinPx, meta.MaxPx),
			}
		}
		return nil
	default:
		return &InvalidThemeTokenValueError{Key: meta.Key, Value: raw, Msg: "unknown kind"}
	}
}

func normalizeTokenValue(meta TokenMeta, raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if err := validateTokenValue(meta, v); err != nil {
		return "", err
	}
	switch meta.Kind {
	case TokenKindColor:
		if hexColorRe.MatchString(v) {
			return strings.ToLower(v), nil
		}
		return v, nil
	case TokenKindLength:
		if m := lengthRe.FindStringSubmatch(v); m != nil {
			f, _ := strconv.ParseFloat(m[1], 64)
			n := int(f + 0.5)
			if n < meta.MinPx {
				n = meta.MinPx
			}
			if n > meta.MaxPx {
				n = meta.MaxPx
			}
			return fmt.Sprintf("%dpx", n), nil
		}
		f, _ := strconv.ParseFloat(v, 64)
		n := int(f + 0.5)
		if n < meta.MinPx {
			n = meta.MinPx
		}
		if n > meta.MaxPx {
			n = meta.MaxPx
		}
		return fmt.Sprintf("%dpx", n), nil
	default:
		return v, nil
	}
}

func allowRGBAHint(meta TokenMeta) string {
	if meta.AllowRGBA {
		return " or rgba()/hsla()"
	}
	return ""
}

// TokenToCSSVar maps a dotted token key to a CSS custom property name.
// color.canvas → --color-canvas; color.cardHi → --color-card-hi;
// space.gapMd → --spacing-gap-md; radius.card → --radius-card.
func TokenToCSSVar(key string) string {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return "--" + kebabToken(key)
	}
	prefix, rest := parts[0], parts[1]
	rest = kebabToken(rest)
	switch prefix {
	case "color":
		return "--color-" + rest
	case "radius":
		return "--radius-" + rest
	case "space":
		return "--spacing-" + rest
	default:
		return "--" + prefix + "-" + rest
	}
}

func kebabToken(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r - 'A' + 'a')
			continue
		}
		if r >= '0' && r <= '9' && i > 0 {
			// ink2 → ink-2
			prev := s[i-1]
			if prev >= 'a' && prev <= 'z' {
				b.WriteByte('-')
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

// MarshalThemeTokens encodes tokens as a JSON object (stable for SQLite).
func MarshalThemeTokens(tokens map[string]string) (string, error) {
	b, err := json.Marshal(tokens)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UnmarshalThemeTokens decodes a JSON object into a token map.
func UnmarshalThemeTokens(raw string) (map[string]string, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("decode theme tokens: %w", err)
	}
	if m == nil {
		m = map[string]string{}
	}
	return m, nil
}

// UnknownThemeTokenError is returned for keys outside the registry.
type UnknownThemeTokenError struct{ Key string }

func (e *UnknownThemeTokenError) Error() string {
	return fmt.Sprintf("unknown theme token %q", e.Key)
}

// InvalidThemeTokenValueError is returned when a token value fails validation.
type InvalidThemeTokenValueError struct {
	Key, Value, Msg string
}

func (e *InvalidThemeTokenValueError) Error() string {
	return fmt.Sprintf("invalid theme token %q (%q): %s", e.Key, e.Value, e.Msg)
}

// InvalidThemeTokensError is a generic tokens payload error.
type InvalidThemeTokensError struct{ Msg string }

func (e *InvalidThemeTokensError) Error() string { return e.Msg }
