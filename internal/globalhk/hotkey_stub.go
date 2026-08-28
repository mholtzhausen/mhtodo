//go:build !linux || !cgo

package globalhk

import "errors"

// ErrUnsupported is returned when this build cannot grab global hotkeys.
var ErrUnsupported = errors.New("global hotkeys unsupported on this platform/build")

// Handle is a no-op stub.
type Handle struct{}

// Register fails on non-Linux or CGO-disabled builds.
func Register(_ []Modifier, _ Key, _ func()) (*Handle, error) {
	return nil, ErrUnsupported
}

// Close is a no-op.
func (h *Handle) Close() {}

// Regrab is a no-op on stub builds.
func (h *Handle) Regrab() error { return ErrUnsupported }

// Modifier / Key match the Linux API surface so call sites compile everywhere.
type Modifier uint32
type Key uint32

const (
	ModCtrl  Modifier = 1 << 2
	ModShift Modifier = 1 << 0
	ModAlt   Modifier = 1 << 3 // Mod1
	KeyT     Key      = 0x0074
)
