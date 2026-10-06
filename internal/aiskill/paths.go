package aiskill

import (
	"os"
	"path/filepath"
)

// homeDir is a test seam for $HOME.
var homeDir = os.UserHomeDir

// HomeDirForTest swaps the home directory resolver.
func HomeDirForTest(f func() (string, error)) (restore func()) {
	prev := homeDir
	homeDir = f
	return func() { homeDir = prev }
}

func home() (string, error) {
	h, err := homeDir()
	if err != nil {
		return "", err
	}
	if h == "" {
		if e := os.Getenv("HOME"); e != "" {
			return e, nil
		}
		return "", os.ErrNotExist
	}
	return h, nil
}

func stateHome() (string, error) {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return x, nil
	}
	h, err := home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".local", "state"), nil
}

// SkillDir returns ~/.claude/skills/mhtodo.
func SkillDir() (string, error) {
	h, err := home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".claude", "skills", "mhtodo"), nil
}

// SkillPath returns ~/.claude/skills/mhtodo/SKILL.md.
func SkillPath() (string, error) {
	d, err := SkillDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "SKILL.md"), nil
}

func claudeHooksDir() (string, error) {
	h, err := home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".claude", "hooks"), nil
}

func claudeSettingsPath() (string, error) {
	h, err := home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".claude", "settings.json"), nil
}

func agentStateDir() (string, error) {
	s, err := stateHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(s, "mhtodo-agent"), nil
}
