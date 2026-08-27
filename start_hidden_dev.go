//go:build dev

package main

// wails dev injects the `dev` tag. Keep the window off-screen so hot-reload
// doesn't steal focus; tray Show / second-launch SIGUSR2 still opens it.
const startHidden = true
