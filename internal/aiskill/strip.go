package aiskill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StripLegacy removes hooks, settings entries, manifests, pointer files, and
// legacy shell helpers. Missing paths are ignored. Returns paths removed.
func StripLegacy() ([]string, error) {
	var removed []string

	hooksDir, err := claudeHooksDir()
	if err == nil {
		entries, _ := os.ReadDir(hooksDir)
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "mhtodo") || strings.Contains(name, "mhtodo") {
				p := filepath.Join(hooksDir, name)
				if err := os.RemoveAll(p); err == nil {
					removed = append(removed, p)
				}
			}
		}
	}

	if settingsPath, err := claudeSettingsPath(); err == nil {
		if n, err := stripSettingsHooks(settingsPath); err == nil && n > 0 {
			removed = append(removed, fmt.Sprintf("%s (mhtodo hook entries)", settingsPath))
		}
	}

	if stateDir, err := agentStateDir(); err == nil {
		for _, name := range []string{"integration.json", "install.json", "manifest.json"} {
			p := filepath.Join(stateDir, name)
			if err := os.Remove(p); err == nil {
				removed = append(removed, p)
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
			p := filepath.Join(stateDir, name)
			if err := os.Remove(p); err == nil {
				removed = append(removed, p)
			}
		}
	}

	if h, err := home(); err == nil {
		for _, p := range []string{
			filepath.Join(h, ".local", "bin", "claude.todo"),
			filepath.Join(h, "bin", "claude.todo"),
			filepath.Join(h, ".claude", "claude.todo"),
		} {
			if err := os.Remove(p); err == nil {
				removed = append(removed, p)
			}
		}
	}

	return removed, nil
}

func stripSettingsHooks(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return 0, nil // leave unparseable settings alone
	}
	hooks, ok := root["hooks"].(map[string]any)
	if !ok || hooks == nil {
		return 0, nil
	}
	changed := 0
	for event, val := range hooks {
		cleaned, n := filterHookValue(val)
		changed += n
		if cleaned == nil {
			delete(hooks, event)
		} else {
			hooks[event] = cleaned
		}
	}
	if changed == 0 {
		return 0, nil
	}
	if len(hooks) == 0 {
		delete(root, "hooks")
	} else {
		root["hooks"] = hooks
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return 0, err
	}
	out = append(out, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return 0, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	return changed, nil
}

func filterHookValue(val any) (any, int) {
	arr, ok := val.([]any)
	if !ok {
		if leafMentionsMhtodo(val) {
			return nil, 1
		}
		return val, 0
	}
	out := make([]any, 0, len(arr))
	removed := 0
	for _, item := range arr {
		m, isMap := item.(map[string]any)
		if !isMap {
			if leafMentionsMhtodo(item) {
				removed++
				continue
			}
			out = append(out, item)
			continue
		}
		if inner, has := m["hooks"]; has {
			cleaned, n := filterHookValue(inner)
			removed += n
			if cleaned == nil {
				delete(m, "hooks")
			} else {
				m["hooks"] = cleaned
			}
			// Drop matcher wrappers that only existed to hold mhtodo hooks.
			if len(m) == 0 || (cleaned == nil && onlyHookKeys(m)) {
				removed++
				continue
			}
			out = append(out, m)
			continue
		}
		if leafMentionsMhtodo(m) {
			removed++
			continue
		}
		out = append(out, m)
	}
	if len(out) == 0 {
		return nil, removed
	}
	return out, removed
}

func onlyHookKeys(m map[string]any) bool {
	for k := range m {
		if k != "matcher" && k != "hooks" && k != "type" {
			return false
		}
	}
	// empty hooks already deleted; if only matcher/type left with no hooks, drop
	_, hasHooks := m["hooks"]
	return !hasHooks
}

func leafMentionsMhtodo(v any) bool {
	switch t := v.(type) {
	case string:
		return strings.Contains(strings.ToLower(t), "mhtodo")
	case map[string]any:
		// Prefer command/path fields; fall back to deep scan excluding nested hooks arrays
		for _, key := range []string{"command", "path", "script", "url"} {
			if s, ok := t[key].(string); ok && strings.Contains(strings.ToLower(s), "mhtodo") {
				return true
			}
		}
		for k, x := range t {
			if k == "hooks" {
				continue
			}
			if leafMentionsMhtodo(x) {
				return true
			}
		}
	case []any:
		for _, x := range t {
			if leafMentionsMhtodo(x) {
				return true
			}
		}
	}
	return false
}
