//go:build !linux

package integrations

import (
	"errors"
	"os"
)

var errNoTerminal = errors.New("terminal launch is only supported on Linux")

func launchInTerminal(string) (int, error) {
	return 0, errNoTerminal
}

func launchInTerminalPreferred(string, string, string) (int, error) {
	return 0, errNoTerminal
}

func activateHerdrWindow() error {
	return errHerdrWindowNotFound
}

func activateWindowForPID(int) error {
	return errHerdrWindowNotFound
}

func activateWindowForPIDPreferTitle(int, string) error {
	return errHerdrWindowNotFound
}

func activateWindowByTitle(string) error {
	return errHerdrWindowNotFound
}

func processParentPID(int) (int, error) {
	return 0, errNoTerminal
}

func execCommandOutput(string, ...string) ([]byte, error) {
	return nil, errNoTerminal
}

func processSignalZero(*os.Process) bool {
	return false
}

func killProcessBestEffort(int) error {
	return nil
}
