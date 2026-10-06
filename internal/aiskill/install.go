package aiskill

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed SKILL.md
var skillFS embed.FS

// EmbeddedSkill returns the embedded SKILL.md bytes.
func EmbeddedSkill() ([]byte, error) {
	return skillFS.ReadFile("SKILL.md")
}

// EmbeddedHash returns a short hex prefix of the embedded skill content hash.
func EmbeddedHash() (string, error) {
	b, err := EmbeddedSkill()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8]), nil
}

// Result is the outcome of Install (also the --json envelope for `mhtodo ai`).
type Result struct {
	Path     string   `json:"path"`
	Action   string   `json:"action"` // installed | updated
	Hash     string   `json:"hash"`
	Removed  []string `json:"removed"`
	CheckOnly bool    `json:"check_only,omitempty"`
	// Check fields (when CheckOnly or after Install for reporting)
	SkillPresent bool     `json:"skill_present,omitempty"`
	SkillStale   bool     `json:"skill_stale,omitempty"`
	Leftovers    []string `json:"leftovers,omitempty"`
	Message      string   `json:"message,omitempty"`
}

// Install writes/overwrites the Claude skill and strips legacy artifacts.
func Install() (Result, error) {
	content, err := EmbeddedSkill()
	if err != nil {
		return Result{}, err
	}
	hash, err := EmbeddedHash()
	if err != nil {
		return Result{}, err
	}
	path, err := SkillPath()
	if err != nil {
		return Result{}, err
	}
	dir := filepath.Dir(path)
	action := "installed"
	if _, err := os.Stat(path); err == nil {
		action = "updated"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create skill dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return Result{}, fmt.Errorf("write skill: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return Result{}, fmt.Errorf("install skill: %w", err)
	}

	removed, err := StripLegacy()
	if err != nil {
		return Result{}, err
	}
	msg := fmt.Sprintf("skill %s at %s", action, path)
	if len(removed) == 0 {
		msg += " (nothing to clean)"
	} else {
		msg += fmt.Sprintf(" (removed %d legacy artifact(s))", len(removed))
	}
	return Result{
		Path:    path,
		Action:  action,
		Hash:    hash,
		Removed: removed,
		Message: msg,
	}, nil
}
