//go:build !linux

package platform

// OnResume is a no-op on non-Linux builds.
func OnResume(func()) {}
