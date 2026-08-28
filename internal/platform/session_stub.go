//go:build !linux

package platform

// PreferX11Backend is a no-op on non-Linux builds.
func PreferX11Backend() {}
