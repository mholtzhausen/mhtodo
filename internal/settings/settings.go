package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"mhtodo/internal/store"
)

// MetaGUISettings is the legacy DB meta key (migrated to config.yml on first load).
const MetaGUISettings = "gui_settings"

const defaultHerdrSpaceName = "mhtodo"

// IntegrationConfig holds one external agent integration (Wails/API surface).
type IntegrationConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Binary   string `json:"binary" yaml:"binary"`
	EnvStart string `json:"env_start" yaml:"env_start"`
}

// HerdrConfig is the Herdr integration (space name is Herdr-specific).
type HerdrConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	Binary    string `json:"binary" yaml:"binary"`
	EnvStart  string `json:"env_start" yaml:"env_start"`
	SpaceName string `json:"space_name" yaml:"space_name"`
}

// GUISettings are user preferences exposed to the GUI.
type GUISettings struct {
	DefaultCwd       string            `json:"default_cwd" yaml:"default_cwd"`
	DefaultHumanOnly bool              `json:"default_human_only" yaml:"default_human_only"`
	Claude           IntegrationConfig `json:"claude" yaml:"claude"`
	Herdr            HerdrConfig       `json:"herdr" yaml:"herdr"`
}

type integrationFile struct {
	Enabled  bool   `yaml:"enabled"`
	Binary   string `yaml:"binary"`
	EnvStart string `yaml:"env_start"`
	UserSet  bool   `yaml:"user_set,omitempty"`
}

type herdrFile struct {
	Enabled   bool   `yaml:"enabled"`
	Binary    string `yaml:"binary"`
	EnvStart  string `yaml:"env_start"`
	SpaceName string `yaml:"space_name"`
	UserSet   bool   `yaml:"user_set,omitempty"`
}

type configFile struct {
	DefaultCwd       string          `yaml:"default_cwd"`
	DefaultHumanOnly bool            `yaml:"default_human_only"`
	Claude           integrationFile `yaml:"claude"`
	Herdr            herdrFile       `yaml:"herdr"`
}

// Default returns factory defaults for a fresh install.
func Default() GUISettings {
	return toGUI(defaultConfigFile())
}

func defaultConfigFile() configFile {
	return configFile{
		Claude: integrationFile{Binary: "claude"},
		Herdr:  herdrFile{Binary: "herdr", SpaceName: defaultHerdrSpaceName},
	}
}

// Load reads settings from config.yml, migrating legacy meta when needed.
// Integrations that have not been user-configured are auto-detected via PATH.
func Load(repo *store.TaskRepo) (GUISettings, error) {
	path := ConfigPath()
	cf, err := readConfigFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Default(), err
		}
		if repo != nil {
			if migrated, ok := migrateFromMeta(context.Background(), repo); ok {
				cf = migrated
			} else {
				cf = defaultConfigFile()
			}
		} else {
			cf = defaultConfigFile()
		}
		normalizeConfigFile(&cf)
		autodetectIntegrations(&cf)
		if werr := writeConfigFile(path, cf); werr != nil {
			return toGUI(cf), werr
		}
		return toGUI(cf), nil
	}

	changed := autodetectIntegrations(&cf)
	if changed {
		if err := writeConfigFile(path, cf); err != nil {
			return toGUI(cf), err
		}
	}
	return toGUI(cf), nil
}

// Save persists settings to config.yml and marks integrations as user-configured.
func Save(s GUISettings) error {
	path := ConfigPath()
	cf := defaultConfigFile()
	if existing, err := readConfigFile(path); err == nil {
		cf = existing
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	applyGUI(&cf, s)
	cf.Claude.UserSet = true
	cf.Herdr.UserSet = true
	return writeConfigFile(path, cf)
}

func autodetectIntegrations(cf *configFile) bool {
	changed := false
	if !cf.Claude.UserSet {
		if p, ok := resolveBinary("claude"); ok {
			p = fullBinaryPath(p)
			if cf.Claude.Binary != p || !cf.Claude.Enabled {
				cf.Claude.Binary = p
				cf.Claude.Enabled = true
				changed = true
			}
		}
	}
	if !cf.Herdr.UserSet {
		if p, ok := resolveBinary("herdr"); ok {
			p = fullBinaryPath(p)
			if cf.Herdr.Binary != p || !cf.Herdr.Enabled {
				cf.Herdr.Binary = p
				cf.Herdr.Enabled = true
				changed = true
			}
		}
	}
	return changed
}

// expandIntegrationBinaries rewrites bare names to absolute executable paths when found.
func expandIntegrationBinaries(cf *configFile) bool {
	changed := false
	if p := fullBinaryPath(cf.Claude.Binary); p != cf.Claude.Binary {
		cf.Claude.Binary = p
		changed = true
	}
	if p := fullBinaryPath(cf.Herdr.Binary); p != cf.Herdr.Binary {
		cf.Herdr.Binary = p
		changed = true
	}
	return changed
}

// fullBinaryPath resolves a command name or path to an absolute executable path.
func fullBinaryPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if p, ok := resolveBinary(path); ok {
		return absClean(p)
	}
	if strings.Contains(path, string(os.PathSeparator)) || strings.HasPrefix(path, ".") {
		if abs, err := filepath.Abs(path); err == nil && isExecutableFile(abs) {
			return absClean(abs)
		}
		if isExecutableFile(path) {
			return absClean(path)
		}
	}
	return path
}

func absClean(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	p = filepath.Clean(p)
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}

// resolveBinary locates an executable on PATH (equivalent to `which`).
func resolveBinary(names ...string) (string, bool) {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if p, err := exec.LookPath(name); err == nil && isExecutableFile(p) {
			return p, true
		}
	}
	return "", false
}

// BinaryFound reports whether path resolves to an executable file.
func BinaryFound(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if !strings.Contains(path, string(os.PathSeparator)) {
		if p, err := exec.LookPath(path); err == nil {
			return isExecutableFile(p)
		}
		return false
	}
	if isExecutableFile(path) {
		return true
	}
	if p, err := exec.LookPath(filepathBase(path)); err == nil {
		return isExecutableFile(p)
	}
	return false
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

func filepathBase(path string) string {
	if i := strings.LastIndex(path, string(os.PathSeparator)); i >= 0 {
		return path[i+1:]
	}
	return path
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readConfigFile(path string) (configFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return configFile{}, err
	}
	var cf configFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return configFile{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	before := cf
	normalizeConfigFile(&cf)
	if cf != before {
		if err := writeConfigFile(path, cf); err != nil {
			return cf, err
		}
	}
	return cf, nil
}

func writeConfigFile(path string, cf configFile) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
	}
	normalizeConfigFile(&cf)
	data, err := yaml.Marshal(&cf)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}

func normalizeConfigFile(cf *configFile) {
	if cf.Claude.Binary == "" {
		cf.Claude.Binary = "claude"
	}
	if cf.Herdr.Binary == "" {
		cf.Herdr.Binary = "herdr"
	}
	if cf.Herdr.SpaceName == "" {
		cf.Herdr.SpaceName = defaultHerdrSpaceName
	}
	expandIntegrationBinaries(cf)
}

func toGUI(cf configFile) GUISettings {
	return GUISettings{
		DefaultCwd:       cf.DefaultCwd,
		DefaultHumanOnly: cf.DefaultHumanOnly,
		Claude: IntegrationConfig{
			Enabled:  cf.Claude.Enabled,
			Binary:   cf.Claude.Binary,
			EnvStart: cf.Claude.EnvStart,
		},
		Herdr: HerdrConfig{
			Enabled:   cf.Herdr.Enabled,
			Binary:    cf.Herdr.Binary,
			EnvStart:  cf.Herdr.EnvStart,
			SpaceName: cf.Herdr.SpaceName,
		},
	}
}

func applyGUI(cf *configFile, s GUISettings) {
	cf.DefaultCwd = s.DefaultCwd
	cf.DefaultHumanOnly = s.DefaultHumanOnly
	cf.Claude.Enabled = s.Claude.Enabled
	cf.Claude.Binary = s.Claude.Binary
	cf.Claude.EnvStart = s.Claude.EnvStart
	cf.Herdr.Enabled = s.Herdr.Enabled
	cf.Herdr.Binary = s.Herdr.Binary
	cf.Herdr.EnvStart = s.Herdr.EnvStart
	cf.Herdr.SpaceName = s.Herdr.SpaceName
	normalizeConfigFile(cf)
}

func migrateFromMeta(ctx context.Context, repo *store.TaskRepo) (configFile, bool) {
	raw, ok, err := repo.GetMeta(ctx, MetaGUISettings)
	if err != nil || !ok || strings.TrimSpace(raw) == "" {
		return configFile{}, false
	}
	var legacy GUISettings
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
		return configFile{}, false
	}
	cf := defaultConfigFile()
	applyGUI(&cf, legacy)
	cf.Claude.UserSet = integrationConfigured(legacy.Claude, "claude")
	cf.Herdr.UserSet = herdrIntegrationConfigured(legacy.Herdr, "herdr")
	return cf, true
}

func integrationConfigured(c IntegrationConfig, defaultBinary string) bool {
	return c.Enabled || c.EnvStart != "" || (c.Binary != "" && c.Binary != defaultBinary)
}

func herdrIntegrationConfigured(c HerdrConfig, defaultBinary string) bool {
	return integrationConfigured(IntegrationConfig{
		Enabled:  c.Enabled,
		Binary:   c.Binary,
		EnvStart: c.EnvStart,
	}, defaultBinary) || (c.SpaceName != "" && c.SpaceName != defaultHerdrSpaceName)
}
