//go:build dev

package main

// wails dev injects the `dev` tag. StartHidden keeps the window hidden so
// hot-reload doesn't steal focus; show it from the tray when needed.
const startHidden = true
