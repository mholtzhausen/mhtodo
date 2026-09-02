//go:build !linux

package integrations

import "errors"

var errNoTerminal = errors.New("terminal launch is only supported on Linux")

func launchInTerminal(string) error {
	return errNoTerminal
}

func activateHerdrWindow() error {
	return errHerdrWindowNotFound
}

func execCommandOutput(string, ...string) ([]byte, error) {
	return nil, errNoTerminal
}
