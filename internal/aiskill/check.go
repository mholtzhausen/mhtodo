package aiskill

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Check reports skill status and leftover legacy artifacts without writing.
func Check() (Result, error) {
	hash, err := EmbeddedHash()
	if err != nil {
		return Result{}, err
	}
	path, err := SkillPath()
	if err != nil {
		return Result{}, err
	}
	res := Result{
		Path:      path,
		Hash:      hash,
		CheckOnly: true,
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			res.SkillPresent = false
			res.SkillStale = true
		} else {
			return Result{}, err
		}
	} else {
		res.SkillPresent = true
		want, err := EmbeddedSkill()
		if err != nil {
			return Result{}, err
		}
		res.SkillStale = !bytes.Equal(raw, want)
	}

	res.Leftovers = scanLeftovers()
	switch {
	case !res.SkillPresent:
		res.Message = fmt.Sprintf("skill missing at %s", path)
	case res.SkillStale:
		res.Message = fmt.Sprintf("skill stale at %s (embedded hash %s)", path, hash)
	default:
		res.Message = fmt.Sprintf("skill current at %s (hash %s)", path, hash)
	}
	if len(res.Leftovers) > 0 {
		res.Message += fmt.Sprintf("; %d leftover artifact(s)", len(res.Leftovers))
	} else {
		res.Message += "; no leftover hooks/settings/manifests"
	}
	return res, nil
}

func scanLeftovers() []string {
	var left []string
	if hooksDir, err := claudeHooksDir(); err == nil {
		entries, _ := os.ReadDir(hooksDir)
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "mhtodo") || strings.Contains(name, "mhtodo") {
				left = append(left, filepath.Join(hooksDir, name))
			}
		}
	}
	if settingsPath, err := claudeSettingsPath(); err == nil {
		if raw, err := os.ReadFile(settingsPath); err == nil && bytes.Contains(bytes.ToLower(raw), []byte("mhtodo")) {
			left = append(left, settingsPath+" (contains mhtodo)")
		}
	}
	if stateDir, err := agentStateDir(); err == nil {
		for _, name := range []string{"integration.json", "install.json", "manifest.json"} {
			p := filepath.Join(stateDir, name)
			if _, err := os.Stat(p); err == nil {
				left = append(left, p)
			}
		}
		entries, _ := os.ReadDir(stateDir)
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if name == "integration.json" || name == "install.json" || name == "manifest.json" {
				continue
			}
			left = append(left, filepath.Join(stateDir, name))
		}
	}
	if h, err := home(); err == nil {
		for _, p := range []string{
			filepath.Join(h, ".local", "bin", "claude.todo"),
			filepath.Join(h, "bin", "claude.todo"),
		} {
			if _, err := os.Stat(p); err == nil {
				left = append(left, p)
			}
		}
	}
	return left
}
