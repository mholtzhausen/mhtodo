package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mhtodo/internal/update"
)

const installStatusCacheTTL = 60 * time.Minute

// InstallStatus drives the conditional header Install control.
type InstallStatus struct {
	Show           bool   `json:"show"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpToDate       bool   `json:"up_to_date"`
	HasService     bool   `json:"has_service"`
	Ephemeral      bool   `json:"ephemeral"`
	InstallPath    string `json:"install_path"`
	Prefix         string `json:"prefix"`
	Message        string `json:"message"`
	// CachedAt is when the latest-version check was last fetched (RFC3339).
	CachedAt string `json:"cached_at"`
	// Fresh is true when CachedAt is within the 60-minute TTL.
	Fresh bool `json:"fresh"`
}

// InstallActionsInput is the confirmation dialog selection.
type InstallActionsInput struct {
	UpdateApp       bool
	InstallService  bool
	IntegrationZsh  bool
	IntegrationBash bool
}

// InstallActionsResult summarizes CLI outcomes for the toast.
type InstallActionsResult struct {
	Message     string `json:"message"`
	Updated     bool   `json:"updated"`
	Service     bool   `json:"service"`
	Integration bool   `json:"integration"`
}

var (
	installStatusMu        sync.Mutex
	installStatusCache     InstallStatus
	installStatusCachedAt  time.Time
	installStatusHaveCache bool
)

// GetInstallStatus returns local install info plus the latest GitHub release.
// Results are cached for 60 minutes. Pass force=true to bypass the cache
// (manual refresh). Hover-triggered checks should pass force=false.
func (a *App) GetInstallStatus(force bool) (InstallStatus, error) {
	installStatusMu.Lock()
	defer installStatusMu.Unlock()

	now := time.Now()
	if !force && installStatusHaveCache && now.Sub(installStatusCachedAt) < installStatusCacheTTL {
		st := installStatusCache
		st.Fresh = true
		st.CachedAt = installStatusCachedAt.UTC().Format(time.RFC3339)
		// Local detect can change without a network round-trip.
		if info, err := update.DetectInstall(); err == nil {
			st.HasService = info.HasService
			st.Ephemeral = update.IsEphemeralInstall(info.Executable)
			st.InstallPath = info.Executable
			st.Prefix = info.Prefix
		}
		st.CurrentVersion = update.NormalizeVersion(version)
		st.Show = !st.UpToDate
		return st, nil
	}

	st, err := fetchInstallStatus()
	if err != nil {
		// Keep serving a stale cache when a forced/network refresh fails.
		if installStatusHaveCache {
			stale := installStatusCache
			stale.Fresh = false
			stale.CachedAt = installStatusCachedAt.UTC().Format(time.RFC3339)
			stale.Message = err.Error()
			stale.CurrentVersion = update.NormalizeVersion(version)
			return stale, nil
		}
		return InstallStatus{
			Show:           false,
			CurrentVersion: update.NormalizeVersion(version),
			Message:        err.Error(),
			Fresh:          false,
		}, nil
	}

	installStatusCache = st
	installStatusCachedAt = now
	installStatusHaveCache = true
	st.CachedAt = now.UTC().Format(time.RFC3339)
	st.Fresh = true
	return st, nil
}

func fetchInstallStatus() (InstallStatus, error) {
	info, err := update.DetectInstall()
	if err != nil {
		return InstallStatus{}, err
	}
	ephemeral := update.IsEphemeralInstall(info.Executable)

	res, err := update.Run(update.Options{
		CurrentVersion: version,
		CheckOnly:      true,
	})
	if err != nil {
		return InstallStatus{
			Show:           false,
			CurrentVersion: update.NormalizeVersion(version),
			HasService:     info.HasService,
			Ephemeral:      ephemeral,
			InstallPath:    info.Executable,
			Prefix:         info.Prefix,
			Message:        err.Error(),
		}, err
	}

	return InstallStatus{
		Show:           !res.UpToDate,
		CurrentVersion: res.CurrentVersion,
		LatestVersion:  res.LatestVersion,
		UpToDate:       res.UpToDate,
		HasService:     info.HasService,
		Ephemeral:      ephemeral,
		InstallPath:    res.InstallPath,
		Prefix:         res.Prefix,
		Message:        res.Message,
	}, nil
}

// RunInstallActions execs this binary's CLI: `update`, `service install`, and/or
// `integration zsh|bash`. When the app is already current, update is invoked with
// --force so a deliberate reinstall still works. Service install is skipped after
// an update that already restarted an attached unit.
func (a *App) RunInstallActions(in InstallActionsInput) (InstallActionsResult, error) {
	if !in.UpdateApp && !in.InstallService && !in.IntegrationZsh && !in.IntegrationBash {
		return InstallActionsResult{}, fmt.Errorf("choose at least one action")
	}

	out := InstallActionsResult{}
	var messages []string

	if in.UpdateApp {
		args := []string{"update", "--json"}
		check, checkErr := update.Run(update.Options{CurrentVersion: version, CheckOnly: true})
		if checkErr == nil && check.UpToDate {
			args = append(args, "--force")
		}
		raw, err := execSelfCLI(args...)
		if err != nil {
			return out, err
		}
		var ur update.Result
		if jerr := json.Unmarshal([]byte(raw), &ur); jerr == nil {
			out.Updated = ur.Updated
			if ur.Message != "" {
				messages = append(messages, ur.Message)
			}
			if ur.Service && ur.Updated {
				out.Service = true
				in.InstallService = false
			}
		} else if strings.TrimSpace(raw) != "" {
			messages = append(messages, strings.TrimSpace(raw))
		}
		// Invalidate version cache after a successful update attempt.
		installStatusMu.Lock()
		installStatusHaveCache = false
		installStatusMu.Unlock()
	}

	if in.InstallService {
		raw, err := execSelfCLI("service", "install", "--json")
		if err != nil {
			if len(messages) > 0 {
				return out, fmt.Errorf("%s; service: %v", strings.Join(messages, "; "), err)
			}
			return out, err
		}
		var sr update.ServiceResult
		if jerr := json.Unmarshal([]byte(raw), &sr); jerr == nil {
			out.Service = true
			if sr.Message != "" {
				messages = append(messages, sr.Message)
			}
		} else if strings.TrimSpace(raw) != "" {
			messages = append(messages, strings.TrimSpace(raw))
		}
	}

	for _, shell := range []struct {
		on   bool
		name string
	}{
		{in.IntegrationZsh, "zsh"},
		{in.IntegrationBash, "bash"},
	} {
		if !shell.on {
			continue
		}
		raw, err := execSelfCLI("integration", shell.name)
		if err != nil {
			if len(messages) > 0 {
				return out, fmt.Errorf("%s; integration %s: %v", strings.Join(messages, "; "), shell.name, err)
			}
			return out, fmt.Errorf("integration %s: %w", shell.name, err)
		}
		out.Integration = true
		msg := firstLine(raw)
		if msg == "" {
			msg = fmt.Sprintf("updated %s integration", shell.name)
		}
		messages = append(messages, msg)
	}

	out.Message = strings.Join(messages, "; ")
	if out.Message == "" {
		out.Message = "done"
	}
	return out, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// execSelfCLI runs os.Executable() with the given CLI args (same binary as the GUI).
func execSelfCLI(args ...string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	cmd := exec.Command(exe, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = out
		}
		if msg == "" {
			msg = err.Error()
		}
		var env struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if json.Unmarshal([]byte(msg), &env) == nil && env.Message != "" {
			msg = env.Message
		}
		return out, fmt.Errorf("%s", msg)
	}
	return out, nil
}
