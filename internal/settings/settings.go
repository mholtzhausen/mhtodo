package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"

	"mhtodo/internal/store"
)

// MetaGUISettings is the legacy DB meta key (migrated to config.yml on first load).
const MetaGUISettings = "gui_settings"

// IntegrationConfig holds one external editor integration (Wails/API surface).
type IntegrationConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Binary   string `json:"binary" yaml:"binary"`
	EnvStart string `json:"env_start" yaml:"env_start"`
}

// NotificationsConfig drives tray label / status submenus and notify-send toggles.
type NotificationsConfig struct {
	TrayLabelStatuses []string `json:"tray_label_statuses" yaml:"tray_label_statuses"` // drive icon label summary
	TrayMenuStatuses  []string `json:"tray_menu_statuses" yaml:"tray_menu_statuses"`   // status submenu order
	MaxItemsPerStatus  int      `json:"max_items_per_status" yaml:"max_items_per_status"`
	NotifySendWIP      bool     `json:"notify_send_wip" yaml:"notify_send_wip"`         // →wip (default off)
	NotifySendWaiting  bool     `json:"notify_send_waiting" yaml:"notify_send_waiting"` // →waiting (default off)
	NotifySendReview   bool     `json:"notify_send_review" yaml:"notify_send_review"`   // →review (default on)
	NotifySendDone     bool     `json:"notify_send_done" yaml:"notify_send_done"`       // →done (default off)
}

// GUISettings are user preferences exposed to the GUI.
type GUISettings struct {
	DefaultCwd             string              `json:"default_cwd" yaml:"default_cwd"`
	DefaultHumanOnly       bool                `json:"default_human_only" yaml:"default_human_only"`
	DefaultIncludeInReport bool                `json:"default_include_in_report" yaml:"default_include_in_report"`
	ArchiveDoneSubtasks    bool                `json:"archive_done_subtasks" yaml:"archive_done_subtasks"`
	StartHidden            bool                `json:"start_hidden" yaml:"start_hidden"` // launch to tray without showing the window
	Notifications          NotificationsConfig `json:"notifications" yaml:"notifications"`
	Zed                    IntegrationConfig   `json:"zed" yaml:"zed"`
}

type integrationFile struct {
	Enabled  bool   `yaml:"enabled"`
	Binary   string `yaml:"binary"`
	EnvStart string `yaml:"env_start,omitempty"`
	UserSet  bool   `yaml:"user_set,omitempty"`
}

type notificationsFile struct {
	TrayLabelStatuses []string `yaml:"tray_label_statuses,omitempty"`
	TrayMenuStatuses  []string `yaml:"tray_menu_statuses,omitempty"`
	MaxItemsPerStatus  *int     `yaml:"max_items_per_status,omitempty"`
	NotifySendWIP      *bool    `yaml:"notify_send_wip,omitempty"`
	NotifySendWaiting  *bool    `yaml:"notify_send_waiting,omitempty"`
	NotifySendReview   *bool    `yaml:"notify_send_review,omitempty"`
	NotifySendDone     *bool    `yaml:"notify_send_done,omitempty"`
}

type configFile struct {
	DefaultCwd             string            `yaml:"default_cwd"`
	DefaultHumanOnly       bool              `yaml:"default_human_only"`
	DefaultIncludeInReport *bool             `yaml:"default_include_in_report,omitempty"`
	ArchiveDoneSubtasks    bool              `yaml:"archive_done_subtasks"`
	StartHidden            bool              `yaml:"start_hidden"`
	Notifications          notificationsFile `yaml:"notifications"`
	Zed                    integrationFile   `yaml:"zed"`
}

// Default returns factory defaults for a fresh install.
func Default() GUISettings {
	return toGUI(defaultConfigFile())
}

func defaultConfigFile() configFile {
	maxItems := 10
	notifyWIP := false
	notifyWaiting := false
	notifyReview := true
	notifyDone := false
	return configFile{
		StartHidden: defaultLaunchHidden(),
		Notifications: notificationsFile{
			TrayLabelStatuses: []string{"waiting", "review"},
			TrayMenuStatuses:  []string{"waiting", "review"},
			MaxItemsPerStatus:  &maxItems,
			NotifySendWIP:      &notifyWIP,
			NotifySendWaiting:  &notifyWaiting,
			NotifySendReview:   &notifyReview,
			NotifySendDone:     &notifyDone,
		},
		Zed: integrationFile{Binary: "zed"},
	}
}

// allowedNotificationStatuses are valid values for tray label/menu status lists.
var allowedNotificationStatuses = map[string]bool{
	"pending": true,
	"wip":     true,
	"waiting": true,
	"review":  true,
	"done":    true,
}

func normalizeStatusList(in []string, fallback []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" || !allowedNotificationStatuses[s] || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	if len(out) == 0 && fallback != nil {
		return append([]string(nil), fallback...)
	}
	return out
}

func normalizeNotifications(nf *notificationsFile) {
	defLabel := []string{"waiting", "review"}
	defMenu := []string{"waiting", "review"}
	nf.TrayLabelStatuses = normalizeStatusList(nf.TrayLabelStatuses, defLabel)
	nf.TrayMenuStatuses = normalizeStatusList(nf.TrayMenuStatuses, defMenu)
	if nf.MaxItemsPerStatus == nil || *nf.MaxItemsPerStatus <= 0 {
		v := 10
		nf.MaxItemsPerStatus = &v
	} else if *nf.MaxItemsPerStatus > 20 {
		v := 20
		nf.MaxItemsPerStatus = &v
	}
	if nf.NotifySendWIP == nil {
		v := false
		nf.NotifySendWIP = &v
	}
	if nf.NotifySendWaiting == nil {
		v := false
		nf.NotifySendWaiting = &v
	}
	if nf.NotifySendReview == nil {
		v := true
		nf.NotifySendReview = &v
	}
	if nf.NotifySendDone == nil {
		v := false
		nf.NotifySendDone = &v
	}
}

func notificationsToGUI(nf notificationsFile) NotificationsConfig {
	normalizeNotifications(&nf)
	return NotificationsConfig{
		TrayLabelStatuses: append([]string(nil), nf.TrayLabelStatuses...),
		TrayMenuStatuses:  append([]string(nil), nf.TrayMenuStatuses...),
		MaxItemsPerStatus:  *nf.MaxItemsPerStatus,
		NotifySendWIP:      *nf.NotifySendWIP,
		NotifySendWaiting:  *nf.NotifySendWaiting,
		NotifySendReview:   *nf.NotifySendReview,
		NotifySendDone:     *nf.NotifySendDone,
	}
}

func notificationsFromGUI(n NotificationsConfig) notificationsFile {
	max := n.MaxItemsPerStatus
	wip := n.NotifySendWIP
	waiting := n.NotifySendWaiting
	review := n.NotifySendReview
	done := n.NotifySendDone
	nf := notificationsFile{
		TrayLabelStatuses: n.TrayLabelStatuses,
		TrayMenuStatuses:  n.TrayMenuStatuses,
		MaxItemsPerStatus:  &max,
		NotifySendWIP:      &wip,
		NotifySendWaiting:  &waiting,
		NotifySendReview:   &review,
		NotifySendDone:     &done,
	}
	normalizeNotifications(&nf)
	return nf
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
	cf.Zed.UserSet = true
	return writeConfigFile(path, cf)
}

func autodetectIntegrations(cf *configFile) bool {
	changed := false
	if !cf.Zed.UserSet {
		if p, ok := resolveBinary("zed"); ok {
			p = fullBinaryPath(p)
			if cf.Zed.Binary != p || !cf.Zed.Enabled {
				cf.Zed.Binary = p
				cf.Zed.Enabled = true
				changed = true
			}
		}
	}
	return changed
}

// expandIntegrationBinaries rewrites bare names to absolute executable paths when found.
func expandIntegrationBinaries(cf *configFile) bool {
	changed := false
	if p := fullBinaryPath(cf.Zed.Binary); p != cf.Zed.Binary {
		cf.Zed.Binary = p
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
	if !reflect.DeepEqual(cf, before) {
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
	if cf.Zed.Binary == "" {
		cf.Zed.Binary = "zed"
	}
	normalizeNotifications(&cf.Notifications)
	expandIntegrationBinaries(cf)
}

func toGUI(cf configFile) GUISettings {
	includeInReport := true
	if cf.DefaultIncludeInReport != nil {
		includeInReport = *cf.DefaultIncludeInReport
	}
	return GUISettings{
		DefaultCwd:             cf.DefaultCwd,
		DefaultHumanOnly:       cf.DefaultHumanOnly,
		DefaultIncludeInReport: includeInReport,
		ArchiveDoneSubtasks:    cf.ArchiveDoneSubtasks,
		StartHidden:            cf.StartHidden,
		Notifications:          notificationsToGUI(cf.Notifications),
		Zed: IntegrationConfig{
			Enabled:  cf.Zed.Enabled,
			Binary:   cf.Zed.Binary,
			EnvStart: cf.Zed.EnvStart,
		},
	}
}

func applyGUI(cf *configFile, s GUISettings) {
	cf.DefaultCwd = s.DefaultCwd
	cf.DefaultHumanOnly = s.DefaultHumanOnly
	include := s.DefaultIncludeInReport
	cf.DefaultIncludeInReport = &include
	cf.ArchiveDoneSubtasks = s.ArchiveDoneSubtasks
	cf.StartHidden = s.StartHidden
	cf.Notifications = notificationsFromGUI(s.Notifications)
	cf.Zed.Enabled = s.Zed.Enabled
	cf.Zed.Binary = s.Zed.Binary
	cf.Zed.EnvStart = s.Zed.EnvStart
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
	cf.Zed.UserSet = integrationConfigured(legacy.Zed, "zed")
	return cf, true
}

func integrationConfigured(c IntegrationConfig, defaultBinary string) bool {
	return c.Enabled || c.EnvStart != "" || (c.Binary != "" && c.Binary != defaultBinary)
}
